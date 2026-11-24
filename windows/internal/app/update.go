//go:build windows

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// Version はビルドのときに-ldflagsで入れる（windows/build.sh）。入っていなければ開発中のビルドとして、新しいバージョンを知らせない
var Version = "dev"

// 新しいバージョンの確認と、その場での更新。1日に1回、GitHubのリリースの一覧からwindows-vのタグを探す。
// macOS向けと同じリポジトリに置くので、/releases/latest（macOS向けのリリース）ではなく一覧から探す。
// 送るのはUser-Agent（"HachibuWindows/バージョン"）だけで、ログイン情報や使用率は送らない。
// 新しいバージョンが出ていれば、裏で添付のzipを取って確かめ（selfupdate.go）、済んだらそのバージョンにつき1回だけ通知する。
// 通知かメニューの項目を押すと、Hachibu.exeを置き換えて開き直す。
// 置き換えられない場所（書き込めないフォルダ、zipを開いたままの一時フォルダ）から動いているときは、ダウンロードのページを開く
const (
	releasesURL        = "https://api.github.com/repos/BoxPistols/hachibu/releases?per_page=30"
	releaseTagPrefix   = "windows-v"
	updateInterval     = 24 * time.Hour
	firstUpdateCheck   = 10 * time.Second
	updateDuePollEvery = time.Hour
	// afterUpdateWait は置き換えた後に開いた新しいほうが、古いほうの終了を待つ時間
	afterUpdateWait = time.Minute
)

// messageBoxFlags はMB_ICONINFORMATION | MB_SETFOREGROUND | MB_TOPMOST。
// 結果は確認を頼んでから少し後に届き、そのときには前面が元のアプリに戻っているので、窓を最前面に出す
const messageBoxFlags = 0x40 | 0x10000 | 0x40000

// タスクダイアログのボタン
const (
	updateRestartID = 100
	updateLaterID   = 101
	updatePageID    = 102
	updateCloseID   = 103
)

type Updater struct {
	prefs *Prefs
	tray  *Tray
	// Available は見つけた新しいバージョン。メニューの先頭に出す
	Available   *core.Release
	LastFailure string
	// Preparing はzipの取得と検査の最中
	Preparing bool
	// Quit は置き換えて新しいほうを開いた後に、このアプリを終える
	Quit func()

	checking bool
	// exe は動いているHachibu.exe。inPlaceはそれをその場で置き換えられるか（起動のときに一度だけ確かめる）
	exe     string
	inPlace bool
	staged  *stagedUpdate
	// installWhenReady は準備ができたら置き換えまで進める（押したが、まだ取得中だったとき）
	installWhenReady bool
	// 最後に出した通知。押したときに置き換えるか、ページを開くかを決める
	notifiedReady bool
	notifiedURL   string
}

func newUpdater(prefs *Prefs, tray *Tray) *Updater {
	u := &Updater{prefs: prefs, tray: tray}
	if exe, err := currentExe(); err == nil {
		u.exe = exe
		u.inPlace = canReplaceExe(exe)
	}
	return u
}

func (u *Updater) Start() {
	// 起動直後はほかの準備（使用率APIの同意の窓など）と重ねない
	After(firstUpdateCheck, u.checkIfDue)
	// スリープをまたいでも1日を越えたら確認する。周期そのものは短くし、実際に叩くかは前回の時刻で決める
	Every(updateDuePollEvery, u.checkIfDue)
}

// AutoCheckChanged は「新しいバージョンを自動で確認」を入れたとき。すぐに一度確認する
func (u *Updater) AutoCheckChanged() {
	if u.prefs.UpdateAutoCheck() {
		u.prefs.SetLastUpdateCheck(time.Time{})
		u.checkIfDue()
	}
}

func (u *Updater) checkIfDue() {
	if !u.prefs.UpdateAutoCheck() {
		return
	}
	if last := u.prefs.LastUpdateCheck(); !last.IsZero() && time.Since(last) < updateInterval {
		return
	}
	u.Check(false)
}

// Check のmanualは「今すぐ確認」。結果が最新でも失敗でも、窓で知らせる
func (u *Updater) Check(manual bool) {
	if u.checking {
		return
	}
	u.checking = true
	u.prefs.SetLastUpdateCheck(time.Now())
	go func() {
		release, found, failure := fetchLatestRelease()
		Post(func() { u.finish(release, found, failure, manual) })
	}()
}

func fetchLatestRelease() (core.Release, bool, string) {
	req, err := http.NewRequest(http.MethodGet, releasesURL, nil)
	if err != nil {
		return core.Release{}, false, "network"
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "HachibuWindows/"+Version)
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return core.Release{}, false, "network"
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return core.Release{}, false, "HTTP " + strconv.Itoa(resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return core.Release{}, false, "network"
	}
	var list []any
	if json.Unmarshal(body, &list) != nil {
		return core.Release{}, false, "unexpected response"
	}
	r, found := core.LatestRelease(list, releaseTagPrefix)
	return r, found, ""
}

func (u *Updater) finish(r core.Release, found bool, failure string, manual bool) {
	u.checking = false
	s := L
	if failure != "" {
		// 既に知っている新しいバージョンの知らせは残す
		if u.LastFailure != failure {
			logAction(s.UpdateCheckFailed(failure))
		}
		u.LastFailure = failure
		if manual {
			w32.MessageBox(0, s.UpdateCheckFailed(failure), core.AppName, messageBoxFlags)
		}
		return
	}
	u.LastFailure = ""
	if !found || !core.IsNewer(r.Version, Version) {
		u.Available = nil
		u.installWhenReady = false
		u.discardStaged()
		if manual {
			w32.MessageBox(0, s.UpdateUpToDate(Version), core.AppName, messageBoxFlags)
		}
		return
	}
	release := r
	u.Available = &release
	if core.ShouldNotify(r, Version, u.prefs.String("notifiedUpdateVersion")) {
		logAction(s.LogUpdateAvailable(r.Version))
	}
	if !u.CanInstall(r) {
		u.notifyOnce(r, false)
		if manual {
			openURL(r.URL)
		}
		return
	}
	if manual && u.ask(r) {
		u.installWhenReady = true
	}
	u.prepare(r)
}

// CanInstall はrをその場で置き換えられるか。できなければ、メニューと通知はダウンロードのページを開く
func (u *Updater) CanInstall(r core.Release) bool {
	return u.inPlace && r.CanInstallInPlace()
}

// Install はメニューの項目、通知、手で確認したときの窓から呼ぶ。準備が済んでいれば置き換え、まだなら済んだときに置き換える
func (u *Updater) Install() {
	if u.Available == nil {
		return
	}
	r := *u.Available
	if !u.CanInstall(r) {
		openURL(r.URL)
		return
	}
	if u.staged != nil && u.staged.version == r.Version {
		u.replaceAndRelaunch(u.staged, r)
		return
	}
	u.installWhenReady = true
	u.prepare(r)
}

// prepare は裏でzipを取り、確かめたexeを置き換えを待つ場所に置く
func (u *Updater) prepare(r core.Release) {
	if u.staged != nil && u.staged.version == r.Version && fileExists(u.staged.exe) {
		u.finishPreparing(r, u.staged, nil)
		return
	}
	if u.Preparing {
		return
	}
	u.Preparing = true
	u.discardStaged()
	dir := updateDir()
	go func() {
		staged, err := downloadUpdate(r, dir)
		Post(func() {
			u.Preparing = false
			if err == nil {
				logAction(L.LogUpdateReady(r.Version))
			}
			u.finishPreparing(r, staged, err)
		})
	}()
}

func (u *Updater) finishPreparing(r core.Release, staged *stagedUpdate, err error) {
	if err != nil {
		logAction(L.UpdateInstallFailed(err.Error()))
		if u.installWhenReady {
			u.installWhenReady = false
			u.failed(err.Error(), r)
		} else {
			// 裏の準備に失敗しても、知らせは出す。押したときにもう一度取得を試みる
			u.notifyOnce(r, false)
		}
		return
	}
	u.staged = staged
	if u.installWhenReady {
		u.installWhenReady = false
		u.replaceAndRelaunch(staged, r)
	} else {
		u.notifyOnce(r, true)
	}
}

func (u *Updater) replaceAndRelaunch(st *stagedUpdate, r core.Release) {
	// 置き換える直前にもう一度確かめる。待っている間に、取得したexeが消されたり書き換えられたりしうる
	if sum, err := fileSHA256(st.exe); err != nil || sum != st.sum {
		u.discardStaged()
		u.installWhenReady = true
		u.prepare(r)
		return
	}
	if err := swapExe(u.exe, st.exe); err != nil {
		u.discardStaged()
		logAction(L.UpdateInstallFailed(err.Error()))
		u.failed(err.Error(), r)
		return
	}
	u.staged = nil
	if err := startAfterUpdate(u.exe); err != nil {
		// 新しいexeを開けなければ元に戻し、このまま動き続ける
		restoreExe(u.exe)
		logAction(L.UpdateInstallFailed(err.Error()))
		u.failed(err.Error(), r)
		return
	}
	logAction(L.LogUpdateInstalling(r.Version))
	if u.Quit != nil {
		u.Quit()
	}
}

func (u *Updater) discardStaged() {
	if u.staged != nil {
		os.Remove(u.staged.exe)
		u.staged = nil
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// MARK: - 知らせ方

// notifyOnce は同じバージョンでは1回だけ通知する。readyなら押すと置き換え、そうでなければダウンロードのページを開く
func (u *Updater) notifyOnce(r core.Release, ready bool) {
	if !core.ShouldNotify(r, Version, u.prefs.String("notifiedUpdateVersion")) {
		return
	}
	u.prefs.Set("notifiedUpdateVersion", r.Version)
	u.notifiedReady = ready
	u.notifiedURL = r.URL
	if ready {
		u.tray.Notify(L.UpdateReadyTitle(r.Version), L.UpdateReadyBody)
	} else {
		u.tray.Notify(L.UpdateNotificationTitle(r.Version), L.UpdateNotificationBody)
	}
}

// NotificationClicked は通知を押したとき。準備ができていれば置き換えて開き直し、そうでなければダウンロードのページを開く
func (u *Updater) NotificationClicked() {
	if u.notifiedReady && u.Available != nil {
		u.Install()
	} else if u.notifiedURL != "" {
		openURL(u.notifiedURL)
	}
}

// ask は手で確認して新しいバージョンがあったときに、更新して開き直すかを聞く
func (u *Updater) ask(r core.Release) bool {
	s := L
	pressed, err := w32.TaskDialog(w32.TaskDialogConfig{
		Title:           core.AppName,
		MainInstruction: s.UpdateNotificationTitle(r.Version),
		Content:         s.UpdateAskBody(Version),
		Buttons:         []w32.TaskDialogButton{{ID: updateRestartID, Text: s.UpdateRestartButton}, {ID: updateLaterID, Text: s.UpdateLaterButton}},
		DefaultButton:   updateRestartID,
	})
	if err != nil {
		text := s.UpdateNotificationTitle(r.Version) + "\n\n" + s.UpdateAskBody(Version)
		return w32.MessageBox(0, text, core.AppName, 0x04|messageBoxFlags) == 6 // MB_YESNO、IDYES
	}
	return pressed == updateRestartID
}

// failed は押して更新しようとして、できなかったとき。ダウンロードのページを開けるようにする
func (u *Updater) failed(reason string, r core.Release) {
	s := L
	pressed, err := w32.TaskDialog(w32.TaskDialogConfig{
		Title:           core.AppName,
		MainInstruction: s.UpdateInstallFailed(reason),
		Content:         s.UpdateInstallFailedBody,
		Buttons:         []w32.TaskDialogButton{{ID: updatePageID, Text: s.UpdateOpenPageButton}, {ID: updateCloseID, Text: s.UpdateCloseButton}},
		DefaultButton:   updatePageID,
	})
	if err != nil {
		text := s.UpdateInstallFailed(reason) + "\n\n" + s.UpdateInstallFailedBody
		if w32.MessageBox(0, text, core.AppName, 0x04|0x30|0x10000|0x40000) == 6 { // MB_YESNO | MB_ICONWARNING
			openURL(r.URL)
		}
		return
	}
	if pressed == updatePageID {
		openURL(r.URL)
	}
}
