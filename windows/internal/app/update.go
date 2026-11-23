//go:build windows

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// Version はビルドのときに-ldflagsで入れる（windows/build.sh）。入っていなければ開発中のビルドとして、新しいバージョンを知らせない
var Version = "dev"

// 新しいバージョンの確認。1日に1回、GitHubのリリースの一覧からwindows-vのタグを探す。
// macOS向けと同じリポジトリに置くので、/releases/latest（macOS向けのリリース）ではなく一覧から探す。
// 送るのはUser-Agent（"HachibuWindows/バージョン"）だけで、ログイン情報や使用率は送らない
const (
	releasesURL        = "https://api.github.com/repos/BoxPistols/hachibu/releases?per_page=30"
	releaseTagPrefix   = "windows-v"
	updateInterval     = 24 * time.Hour
	firstUpdateCheck   = 10 * time.Second
	updateDuePollEvery = time.Hour
)

// messageBoxFlags はMB_ICONINFORMATION | MB_SETFOREGROUND | MB_TOPMOST。
// 結果は確認を頼んでから少し後に届き、そのときには前面が元のアプリに戻っているので、窓を最前面に出す
const messageBoxFlags = 0x40 | 0x10000 | 0x40000

type Updater struct {
	prefs *Prefs
	tray  *Tray
	// Available は見つけた新しいバージョン。メニューの先頭に出す
	Available   *core.Release
	LastFailure string
	checking    bool
	// notifiedURL は通知を押したときに開くページ
	notifiedURL string
}

func newUpdater(prefs *Prefs, tray *Tray) *Updater {
	return &Updater{prefs: prefs, tray: tray}
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
	newer := found && core.IsNewer(r.Version, Version)
	u.Available = nil
	if newer {
		release := r
		u.Available = &release
	}
	if newer && core.ShouldNotify(r, Version, u.prefs.String("notifiedUpdateVersion")) {
		u.prefs.Set("notifiedUpdateVersion", r.Version)
		logAction(s.LogUpdateAvailable(r.Version))
		u.notifiedURL = r.URL
		u.tray.Notify(s.UpdateNotificationTitle(r.Version), s.UpdateNotificationBody)
	}
	if manual {
		if newer {
			openURL(r.URL)
		} else {
			w32.MessageBox(0, s.UpdateUpToDate(Version), core.AppName, messageBoxFlags)
		}
	}
}

// NotificationClicked は通知を押したとき。ダウンロードのページを開く
func (u *Updater) NotificationClicked() {
	if u.notifiedURL != "" {
		openURL(u.notifiedURL)
	}
}
