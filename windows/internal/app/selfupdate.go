//go:build windows

package app

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
	"golang.org/x/sys/windows"
)

// その場での更新のうち、ファイルを扱う部分。新しいHachibu.exeを取って確かめ、動いているexeと入れ替えて開き直す。
//
// Windowsは動いているexeを上書きできないが、名前は変えられる。そこで動いているexeをHachibu.exe.oldへ退け、
// 確かめた新しいexeを元の名前に置き、--after-updateを付けて開く。新しいほうは古いほうの終了を待ってから動き、.oldを消す。
// Windows向けはコード署名をしていないので、zipはリリースの本文のSHA-256と、中のexeは埋め込みの製品名とバージョンと突き合わせる

const (
	// afterUpdateFlag は置き換えた後に開くときの引数。続けて古いほうのプロセスIDを渡す
	afterUpdateFlag = "--after-update"
	// maxUpdateBytes は取得するzipと、その中のexeの上限（今はどちらも数MB）
	maxUpdateBytes = 64 << 20
	// oldExeSuffix は退けた古いexeの名前の後ろに付ける
	oldExeSuffix = ".old"
)

// stagedUpdate は取得して確かめ、置き換えを待っているexe
type stagedUpdate struct {
	version string
	exe     string
	// sum はexeのSHA-256。置き換える直前にもう一度確かめる
	sum string
}

// updateDir は取得したzipとexeを置く場所。試験用の起動（HACHIBU_SUITE）では別にする
func updateDir() string {
	name := "update"
	if suite := os.Getenv("HACHIBU_SUITE"); suite != "" {
		name += "-" + filepath.Base(suite)
	}
	return filepath.Join(dataDir(), name)
}

// currentExe は動いているHachibu.exeの場所
func currentExe() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(exe)
}

// canReplaceExe はexeをその場で置き換えられるか。書き込めないフォルダ（Program Filesなど）や、
// zipを開いたまま中のexeを起動したときの一時フォルダからは置き換えず、ダウンロードのページを開く形に戻す
func canReplaceExe(exe string) bool {
	dir := filepath.Dir(exe)
	if tmp := os.TempDir(); tmp != "" && isWithin(dir, tmp) {
		return false
	}
	f, err := os.CreateTemp(dir, ".hachibu-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// isWithin はdirがbaseかその中か（大文字と小文字は区別しない）
func isWithin(dir, base string) bool {
	rel, err := filepath.Rel(base, dir)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, `..\`)
}

// downloadUpdate はzipを取り、SHA-256を確かめ、中のHachibu.exeを取り出して製品名とバージョンを確かめる。
// 添付の名前にバージョンが入っていないので、取り違えたzipで上書きしないよう中身で確かめる
func downloadUpdate(r core.Release, dir string) (*stagedUpdate, error) {
	if !r.CanInstallInPlace() {
		return nil, errors.New("the release has no zip or SHA-256")
	}
	os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	zipPath := filepath.Join(dir, core.WindowsAssetName)
	if err := downloadFile(r.AssetURL, zipPath); err != nil {
		return nil, err
	}
	defer os.Remove(zipPath)
	if sum, err := fileSHA256(zipPath); err != nil || sum != r.SHA256 {
		return nil, errors.New("SHA-256 does not match")
	}
	exe := filepath.Join(dir, "Hachibu-"+r.Version+".exe")
	if err := extractExe(zipPath, exe); err != nil {
		return nil, err
	}
	if err := checkExe(exe, r.Version); err != nil {
		os.Remove(exe)
		return nil, err
	}
	sum, err := fileSHA256(exe)
	if err != nil {
		os.Remove(exe)
		return nil, err
	}
	return &stagedUpdate{version: r.Version, exe: exe, sum: sum}, nil
}

func downloadFile(url, dest string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "HachibuWindows/"+Version)
	client := &http.Client{
		Timeout: 2 * time.Minute,
		// GitHubは添付を別のホストへ転送する。httpsのままの転送だけをたどる
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if next.URL.Scheme != "https" {
				return errors.New("redirected to a non-https address")
			}
			if len(via) >= 5 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("download failed (network)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed (HTTP %d)", resp.StatusCode)
	}
	return writeLimited(dest, resp.Body)
}

// writeLimited はrをdestに書く。上限を超えたら書いた分を消してエラーにする
func writeLimited(dest string, r io.Reader) error {
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, maxUpdateBytes+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxUpdateBytes {
		err = errors.New("the download is too large")
	}
	if err != nil {
		os.Remove(dest)
	}
	return err
}

// extractExe はzipの一番上にあるHachibu.exeだけをdestに取り出す
func extractExe(zipPath, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.New("could not open the zip")
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "Hachibu.exe" {
			continue
		}
		if f.UncompressedSize64 > maxUpdateBytes {
			return errors.New("the Hachibu.exe in the zip is too large")
		}
		rc, err := f.Open()
		if err != nil {
			return errors.New("could not unzip")
		}
		defer rc.Close()
		if err := writeLimited(dest, rc); err != nil {
			return errors.New("could not unzip")
		}
		return nil
	}
	return errors.New("the zip has no Hachibu.exe")
}

// checkExe は取り出したexeの製品名とバージョンを確かめる（windows/build.shが埋め込む）
func checkExe(exe, version string) error {
	info, err := w32.FileVersionStrings(exe, "ProductName", "ProductVersion")
	if err != nil || info["ProductName"] != core.AppName {
		return errors.New("the zip is not Hachibu")
	}
	if info["ProductVersion"] != version {
		return fmt.Errorf("the app in the zip is not %s", version)
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// swapExe は動いているtargetをtarget.oldへ退け、stagedをtargetに置く。置けなければ元に戻す
func swapExe(target, staged string) error {
	old := target + oldExeSuffix
	// 前回の更新の残り。消せなくても、下の名前の変更で上書きする
	os.Remove(old)
	if err := os.Rename(target, old); err != nil {
		return fmt.Errorf("could not move %s aside: %w", filepath.Base(target), err)
	}
	if err := moveFile(staged, target); err != nil {
		restoreExe(target)
		return fmt.Errorf("could not put the new %s in place: %w", filepath.Base(target), err)
	}
	return nil
}

// restoreExe は退けた古いexeを元の名前に戻す（新しいexeを開けなかったとき）
func restoreExe(target string) {
	os.Remove(target)
	os.Rename(target+oldExeSuffix, target)
}

// moveFile は別のドライブへもコピーして移す（取得したexeはデータのフォルダにあり、Hachibu.exeは別のドライブにありうる）
func moveFile(from, to string) error {
	f, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	t, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(f, t, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_COPY_ALLOWED|windows.MOVEFILE_WRITE_THROUGH)
}

// startAfterUpdate は置き換えたexeを開く。新しいほうは、このプロセスが終わるのを待ってから動く
// （先に動くと、二重起動の防止でこちらに帯を出させて終わってしまう）
func startAfterUpdate(exe string) error {
	cmd := exec.Command(exe, afterUpdateFlag, strconv.Itoa(os.Getpid()))
	cmd.Dir = filepath.Dir(exe)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// afterUpdatePID は--after-update <PID>で開かれたときの、古いほうのプロセスID
func afterUpdatePID(args []string) (uint32, bool) {
	for i, a := range args {
		if a == afterUpdateFlag && i+1 < len(args) {
			pid, err := strconv.ParseUint(args[i+1], 10, 32)
			return uint32(pid), err == nil && pid != 0 && uint32(pid) != uint32(os.Getpid())
		}
	}
	return 0, false
}

// waitForExit はpidのプロセスが終わるのを待つ。もう無ければすぐ戻る。待ちきれなければfalse
func waitForExit(pid uint32, timeout time.Duration) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return true
	}
	defer windows.CloseHandle(h)
	ev, _ := windows.WaitForSingleObject(h, uint32(timeout.Milliseconds()))
	return ev == windows.WAIT_OBJECT_0
}

// cleanUpAfterUpdate は前回の更新の残り（退けた古いexeと、取得に使ったフォルダ）を消す。
// 古いプロセスが終わった直後はまだexeを放していないことがあるので、少し待ちながら試し直す
func cleanUpAfterUpdate() {
	exe, err := currentExe()
	if err != nil {
		return
	}
	// 取得に使ったフォルダは、この起動の確認（10秒後）より前に消し終える
	os.RemoveAll(updateDir())
	old := exe + oldExeSuffix
	go func() {
		for range 10 {
			err := os.Remove(old)
			if err == nil || errors.Is(err, os.ErrNotExist) {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
}
