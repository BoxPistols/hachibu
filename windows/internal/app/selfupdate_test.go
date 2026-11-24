//go:build windows

package app

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// makeZip はnames→中身のzipを作る
func makeZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, data := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		f.Write(data)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestExtractsOnlyTheTopLevelExe(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.zip")
	os.WriteFile(ok, makeZip(t, map[string][]byte{"README.txt": []byte("x"), "Hachibu.exe": []byte("exe")}), 0o644)
	dest := filepath.Join(dir, "out.exe")
	if err := extractExe(ok, dest); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(dest); string(got) != "exe" {
		t.Fatalf("%q", got)
	}
	for name, files := range map[string]map[string][]byte{
		"nested":  {"sub/Hachibu.exe": []byte("exe")},
		"missing": {"Other.exe": []byte("exe")},
	} {
		p := filepath.Join(dir, name+".zip")
		os.WriteFile(p, makeZip(t, files), 0o644)
		if err := extractExe(p, filepath.Join(dir, name+".exe")); err == nil {
			t.Errorf("%s: should fail", name)
		}
	}
	notZip := filepath.Join(dir, "not.zip")
	os.WriteFile(notZip, []byte("not a zip"), 0o644)
	if err := extractExe(notZip, filepath.Join(dir, "x.exe")); err == nil {
		t.Error("not a zip: should fail")
	}
}

func TestChecksTheProductNameAndVersion(t *testing.T) {
	// テストの実行ファイルにはバージョン情報が無い。メモ帳はMicrosoftの製品
	self, _ := os.Executable()
	notepad := filepath.Join(os.Getenv("SystemRoot"), "System32", "notepad.exe")
	for _, exe := range []string{self, notepad} {
		if err := checkExe(exe, "0.1.0"); err == nil || err.Error() != "the zip is not Hachibu" {
			t.Errorf("%s: %v", exe, err)
		}
	}
}

func TestSwapsTheExeAndPutsItBackOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "Hachibu.exe")
	staged := filepath.Join(t.TempDir(), "Hachibu-0.2.0.exe")
	os.WriteFile(target, []byte("old"), 0o644)
	os.WriteFile(staged, []byte("new"), 0o644)
	// 前回の残りがあっても上書きする
	os.WriteFile(target+oldExeSuffix, []byte("older"), 0o644)
	if err := swapExe(target, staged); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Fatalf("target %q", got)
	}
	if got, _ := os.ReadFile(target + oldExeSuffix); string(got) != "old" {
		t.Fatalf("old %q", got)
	}
	if fileExists(staged) {
		t.Fatal("staged should have moved")
	}

	// 置くものが無ければ、元の名前に戻す
	os.Remove(target + oldExeSuffix)
	if err := swapExe(target, filepath.Join(dir, "missing.exe")); err == nil {
		t.Fatal("should fail")
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Fatalf("not restored: %q", got)
	}
	if fileExists(target + oldExeSuffix) {
		t.Fatal(".old should be gone after restoring")
	}
}

func TestSwapsARunningExe(t *testing.T) {
	// 動いているexeは上書きできないが、名前は変えられる（その場での更新はこれに頼る）
	self, _ := os.Executable()
	dir := t.TempDir()
	running := filepath.Join(dir, "Hachibu.exe")
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(running, data, 0o755)
	cmd := execHelper(running)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	staged := filepath.Join(dir, "staged.exe")
	os.WriteFile(staged, []byte("new"), 0o644)
	if err := swapExe(running, staged); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(running); string(got) != "new" {
		t.Fatalf("%q", got)
	}
}

func TestReadsTheOldProcessID(t *testing.T) {
	cases := []struct {
		args []string
		pid  uint32
		ok   bool
	}{
		{[]string{"--after-update", "1234"}, 1234, true},
		{[]string{"x", "--after-update", "42"}, 42, true},
		{[]string{"--after-update"}, 0, false},
		{[]string{"--after-update", "abc"}, 0, false},
		{[]string{"--after-update", "0"}, 0, false},
		{[]string{"--after-update", strconv.Itoa(os.Getpid())}, uint32(os.Getpid()), false},
		{nil, 0, false},
	}
	for _, c := range cases {
		pid, ok := afterUpdatePID(c.args)
		if ok != c.ok || (ok && pid != c.pid) {
			t.Errorf("%v → %d %v", c.args, pid, ok)
		}
	}
	// もう無いプロセスは待たない
	if !waitForExit(0xFFFFFFF0, 0) {
		t.Error("a missing process should count as exited")
	}
}

func TestDoesNotReplaceFromTheTempFolder(t *testing.T) {
	// zipを開いたまま中のexeを起動すると、一時フォルダから動く
	if canReplaceExe(filepath.Join(t.TempDir(), "Hachibu.exe")) {
		t.Error("temp folder")
	}
	if !isWithin(`C:\Users\a\AppData\Local\Temp\x`, `c:\users\a\appdata\local\temp`) || isWithin(`C:\Tools`, `C:\Users\a\AppData\Local\Temp`) {
		t.Error("isWithin")
	}
	base, err := os.UserCacheDir()
	if err != nil {
		t.Skip(err)
	}
	dir, err := os.MkdirTemp(base, "hachibu-test-")
	if err != nil {
		t.Skip(err)
	}
	defer os.RemoveAll(dir)
	if !canReplaceExe(filepath.Join(dir, "Hachibu.exe")) {
		t.Error("a writable folder outside the temp folder")
	}
}

// serve はdataを返すサーバー。httpで立てるが、URLの確かめ（GitHubのhttpsだけ）はcoreで済んでいる
func serve(t *testing.T, data []byte) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	t.Cleanup(srv.Close)
	return srv.URL + "/" + core.WindowsAssetName
}

func TestRefusesAZipThatDoesNotMatch(t *testing.T) {
	self, _ := os.Executable()
	exe, _ := os.ReadFile(self)
	data := makeZip(t, map[string][]byte{"Hachibu.exe": exe})
	url := serve(t, data)
	dir := filepath.Join(t.TempDir(), "update")

	// SHA-256が本文と違う
	_, err := downloadUpdate(core.Release{Version: "9.9.9", AssetURL: url, SHA256: strings.Repeat("0", 64)}, dir)
	if err == nil || err.Error() != "SHA-256 does not match" {
		t.Fatalf("%v", err)
	}
	// SHA-256は合うが、中身がHachibuでない
	_, err = downloadUpdate(core.Release{Version: "9.9.9", AssetURL: url, SHA256: sum(data)}, dir)
	if err == nil || err.Error() != "the zip is not Hachibu" {
		t.Fatalf("%v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("left files: %v", entries)
	}
	// 添付かSHA-256が無ければ取りに行かない
	if _, err := downloadUpdate(core.Release{Version: "9.9.9", AssetURL: url}, dir); err == nil {
		t.Fatal("no SHA-256")
	}
}

// TestStagesARealBuild は組み立てたHachibu.exe（HACHIBU_TEST_EXE）で、取得から置き換えを待つところまでを通す
func TestStagesARealBuild(t *testing.T) {
	path := os.Getenv("HACHIBU_TEST_EXE")
	if path == "" {
		t.Skip("set HACHIBU_TEST_EXE to a Hachibu.exe built by windows/build.sh")
	}
	exe, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data := makeZip(t, map[string][]byte{"Hachibu.exe": exe})
	url := serve(t, data)
	dir := filepath.Join(t.TempDir(), "update")
	r := core.Release{Version: Version, AssetURL: url, SHA256: sum(data)}
	if info, err := readVersion(path); err == nil {
		r.Version = info
	}
	st, err := downloadUpdate(r, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.version != r.Version || st.sum != sum(exe) || !fileExists(st.exe) {
		t.Fatalf("%+v", st)
	}
	// 別のバージョンだと言われたら受け取らない
	r.Version = "0.0.1"
	if _, err := downloadUpdate(r, dir); err == nil || !strings.Contains(err.Error(), "is not 0.0.1") {
		t.Fatalf("%v", err)
	}
}

// execHelper はテストの実行ファイルのコピーを、しばらく動き続けるプロセスとして開く
func execHelper(exe string) *exec.Cmd {
	cmd := exec.Command(exe, "-test.run=^TestHelperKeepsRunning$")
	cmd.Env = append(os.Environ(), "HACHIBU_TEST_HELPER=1")
	return cmd
}

func TestHelperKeepsRunning(t *testing.T) {
	if os.Getenv("HACHIBU_TEST_HELPER") != "1" {
		t.Skip("only as a helper process")
	}
	time.Sleep(30 * time.Second)
}

func readVersion(path string) (string, error) {
	info, err := w32.FileVersionStrings(path, "ProductVersion")
	if err != nil {
		return "", err
	}
	return info["ProductVersion"], nil
}

func TestDoesNotApplyADownloadThatIsNoLongerTheNewest(t *testing.T) {
	stage := func() *stagedUpdate {
		p := filepath.Join(t.TempDir(), "Hachibu-0.3.0.exe")
		os.WriteFile(p, []byte("0.3.0"), 0o644)
		return &stagedUpdate{version: "0.3.0", exe: p, sum: sum([]byte("0.3.0"))}
	}
	// 取得の間に0.4.0が見つかり、押されていた。0.3.0は当てず、押された状態は0.4.0の取得に引き継ぐ
	// （Preparingにしておき、0.4.0の取得はここでは始めない）
	newer := core.Release{Version: "0.4.0", URL: "https://github.com/BoxPistols/hachibu/releases/tag/windows-v0.4.0"}
	u := &Updater{Available: &newer, installWhenReady: true, Preparing: true}
	st := stage()
	u.finishPreparing(core.Release{Version: "0.3.0"}, st, nil)
	if u.staged != nil || fileExists(st.exe) || !u.installWhenReady {
		t.Fatalf("staged=%v exists=%v installWhenReady=%v", u.staged, fileExists(st.exe), u.installWhenReady)
	}
	// 新しいバージョンが無くなっていた（最新だった）なら、押された状態も消す
	u = &Updater{installWhenReady: true}
	st = stage()
	u.finishPreparing(core.Release{Version: "0.3.0"}, st, nil)
	if u.staged != nil || fileExists(st.exe) || u.installWhenReady {
		t.Fatalf("staged=%v exists=%v installWhenReady=%v", u.staged, fileExists(st.exe), u.installWhenReady)
	}
}
