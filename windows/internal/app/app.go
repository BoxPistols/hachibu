//go:build windows

// Package app はHachibu for Windowsの本体。帯（画面の最前面の小さなバー）、トレイのアイコン、メニュー、
// ショートカット、データの読み取りをつなぐ。表示の判断はinternal/coreにあり、ここは窓とOSとのやりとりを受け持つ
package app

import (
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
	"golang.org/x/sys/windows"
)

// L は今使う文言の表。UIのスレッドだけで使う
var L = core.En

func applyLanguage(saved string) {
	L = core.StringsFor(core.DetectLanguage(preferredLanguages(), saved, os.Getenv("HACHIBU_LANG")))
}

type App struct {
	prefs   *Prefs
	strip   *Strip
	tray    *Tray
	source  *Source
	updates *Updater

	hotKeys       *core.HotKeyCenter
	cycleKeys     *core.HotKeyCenter
	recorder      *Recorder
	cycleRecorder *Recorder

	state      core.SourceState
	dirs       []claudeDir
	display    core.Display
	dirsLabel  string
	apiFailure string
	loginErr   string

	activeMenu     *menuSet
	menuClosedAt   time.Time
	lastForeground w32.HWND
	taskbarCreated uint32
	demo           bool
}

// Run はアプリを動かし、終わったら終了コードを返す。mainのゴルーチンから呼ぶ
func Run() int {
	// 窓とメッセージは、作ったスレッドでしか扱えない
	runtime.LockOSThread()
	// 常駐するので、使い終えたメモリを早めにWindowsへ返す（会話記録の末尾を読むたびに数MBを一時的に使う）。
	// 重い計算はしないので、ガベージコレクタが同時に使うCPUも2つまでにする
	debug.SetGCPercent(50)
	debug.SetMemoryLimit(16 << 20)
	runtime.GOMAXPROCS(2)
	if dir := snapshotDir(); dir != "" {
		return runSnapshot(dir)
	}
	w32.SetPerMonitorDPIAware()
	// ShellExecute（URLやログを開く）とタスクダイアログは、COMを初期化したスレッドから呼ぶ
	windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE)
	w32.InitCommonControls()
	w32.AllowDarkMenus()
	if err := w32.StartGDIPlus(); err != nil {
		w32.MessageBox(0, err.Error(), core.AppName, 0x10)
		return 1
	}
	// 置き換えた後に開かれたときは、古いほうが終わるのを待つ（先に進むと、二重起動の防止で古いほうに帯を出させて終わる）
	pid, afterUpdate := afterUpdatePID(os.Args[1:])
	if afterUpdate {
		waitForExit(pid, afterUpdateWait)
	}
	// 同じアプリの別のコピーがすでに動いていれば、そちらに帯を出させて終わる（帯を重ねて出さない）
	if handOff() {
		return 0
	}
	a := &App{prefs: loadPrefs(), demo: isDemo()}
	// 文言を読む部品を作る前に、メニューで選んだ言語にしておく
	applyLanguage(a.prefs.Language())
	if afterUpdate {
		logAction(L.LogUpdated(Version))
	}

	registerClass(mainClass, 0)
	mainHwnd = newWindow(w32.WS_EX_TOOLWINDOW, mainClass, core.AppName, w32.WS_POPUP, 0, 0, 0, 0, 0, a.handleMain)
	a.taskbarCreated = w32.RegisterWindowMessage("TaskbarCreated")
	a.strip = newStrip(a.prefs)
	a.tray = newTray(mainHwnd)

	logShortcut := func(e core.HotKeyEvent) {
		switch e.Kind {
		case core.HotKeyCommitted:
			logAction(L.LogShortcutCommitted(e.Shortcut.Display()))
		case core.HotKeyDisabled:
			logAction(L.LogShortcutDisabled)
		case core.HotKeyRegistrationFailed:
			logAction(L.LogShortcutFailed(e.Shortcut.Display()))
		}
	}
	summon := newHotKeyRegistrar(1)
	a.hotKeys = core.NewHotKeyCenter(summon, core.SummonSettings(a.prefs))
	summon.onPress = a.hotKeys.Fire
	a.hotKeys.OnPress = a.strip.ToggleSummon
	a.hotKeys.OnEvent = logShortcut
	// 表示を順に切り替えるショートカット。利用者が設定するまでは何も登録しない
	cycle := newHotKeyRegistrar(2)
	a.cycleKeys = core.NewHotKeyCenter(cycle, core.LayoutCycleSettings(a.prefs))
	cycle.onPress = a.cycleKeys.Fire
	a.cycleKeys.OnPress = a.strip.CycleLayout
	a.cycleKeys.OnEvent = logShortcut
	a.recorder = newRecorder(a.hotKeys, func() string { return L.RecorderTitle }, func() *core.Shortcut { return a.cycleKeys.Shortcut })
	a.cycleRecorder = newRecorder(a.cycleKeys, func() string { return L.RecorderCycleTitle }, func() *core.Shortcut { return a.hotKeys.Shortcut })

	a.strip.OnChange = a.refreshTray
	a.strip.UsageText = func() string { return core.UsageOnlyText(a.display.Limits) }
	a.strip.ContextMenu = func(at w32.POINT) { a.showMenu(a.buildStripMenu, at) }

	if a.demo {
		// 撮影用の起動ではトレイのアイコンを作らない（Windowsの「その他のシステム トレイ アイコン」の一覧に行を増やさないため）
		a.applyState(demoState(time.Now()), nil)
	} else {
		a.source = newSource(a.prefs.UsageAPIEnabled())
		a.source.OnUpdate = a.applyState
		a.source.Start()
		cleanUpAfterUpdate()
		a.updates = newUpdater(a.prefs, a.tray)
		a.updates.Quit = a.quit
		a.updates.Start()
		// まだ尋ねていなければ、使用率APIを使うかを最初に一度だけ尋ねる
		if a.prefs.UsageAPIConsent() == nil {
			After(800*time.Millisecond, a.askConsent)
		}
	}
	a.strip.render()
	// 開発用: 試験用の保存先（HACHIBU_SUITE）か撮影用の起動のときだけ、起動の直後にメニューや窓を開く（カーソルを動かさずに画面写真を撮るため）
	if what := os.Getenv("HACHIBU_OPEN"); what != "" && (a.demo || os.Getenv("HACHIBU_SUITE") != "") {
		After(time.Second, func() { a.devOpen(what) })
	}
	messageLoop()
	a.tray.Remove()
	flushLog(2 * time.Second)
	return 0
}

// applyState は読み取りのゴルーチンから届いた値で、帯とトレイを更新する
func (a *App) applyState(st core.SourceState, dirs []claudeDir) {
	a.state = st
	a.dirs = dirs
	// トレイへの登録に失敗したまま（エクスプローラーの起動前など）なら、表示が変わらなくても2秒ごとに試し直す
	if a.tray != nil && !a.tray.added {
		a.refreshTray()
	}
	if st.UseAPI && st.APIFailure != "" && st.APIFailure != a.apiFailure {
		logAction(L.UsageAPIFailed(st.APIFailure))
	}
	a.apiFailure = st.APIFailure
	if !a.demo {
		if label := labels(dirs); label != a.dirsLabel {
			a.dirsLabel = label
			logAction(L.LogClaudeDirs(label))
		}
	}
	d := core.Compose(st, L, time.Now())
	if reflect.DeepEqual(d, a.display) {
		return
	}
	a.display = d
	a.strip.Update([]core.Slot{d.Slot})
}

// refreshTray はトレイのアイコンとツールチップを今の値に合わせる。帯の中身が変わるたびに呼ぶ
func (a *App) refreshTray() {
	if a.demo || a.tray == nil {
		return
	}
	a.tray.Refresh(a.prefs.TrayStyle(), a.display.Limits, a.strip.Thresholds, a.trayTip())
}

// trayTip はツールチップの文言。トレイのツールチップは127文字までなので、入る行だけを並べる
func (a *App) trayTip() string {
	lines := []string{core.AppName}
	if slot, ok := a.strip.StatusSlot(); ok {
		head := slot.Text
		if spelled, ok := core.SpelledOut(slot.Text); ok {
			head = spelled + " · " + core.UsageOnlyText(a.display.Limits)
		}
		lines = append(lines, strings.TrimSuffix(head, " · "))
	}
	for _, l := range a.display.Limits {
		lines = append(lines, l.Line(L, time.Now()))
	}
	tip := ""
	for _, line := range lines {
		next := line
		if tip != "" {
			next = tip + "\n" + line
		}
		if len([]rune(next)) > 127 {
			break
		}
		tip = next
	}
	return tip
}

// reopenGuard はメニューを閉じてから、トレイのアイコンの押下で開き直さない時間
const reopenGuard = 300 * time.Millisecond

func (a *App) handleMain(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) (uintptr, bool) {
	if a.taskbarCreated != 0 && msg == a.taskbarCreated {
		// エクスプローラーが再起動すると、トレイのアイコンは消えている
		a.tray.added = false
		a.refreshTray()
		return 0, true
	}
	switch msg {
	case msgRun:
		drainQueue()
		return 0, true
	case msgTray:
		switch uint32(w32.LoWord(lParam)) {
		case w32.NIN_SELECT, w32.NIN_KEYSELECT, w32.WM_CONTEXTMENU:
			// 開いているメニューをアイコンを押して閉じると、閉じた直後にその押下の知らせが届く。そこで開き直さない
			// （Enterで選んだときにNIN_KEYSELECTが2回届くのも、同じ理由で1回にする）
			if time.Since(a.menuClosedAt) < reopenGuard {
				return 0, true
			}
			at := w32.POINT{X: int32(int16(w32.LoWord(wParam))), Y: int32(int16(w32.HiWord(wParam)))}
			a.showMenu(a.buildTrayMenu, at)
		case w32.NIN_BALLOONUSERCLICK:
			if a.updates != nil {
				a.updates.NotificationClicked()
			}
		}
		return 0, true
	case msgSummon:
		// 後から開いたコピー（エクスプローラーからもう一度開いたなど）からの知らせ。隠すモードでも帯を出す
		a.strip.ShowSummoned()
		return 0, true
	case w32.WM_HOTKEY:
		hotKeyPressed(int32(wParam))
		return 0, true
	case w32.WM_MENUSELECT:
		a.menuSelect(wParam, lParam)
		return 0, true
	case w32.WM_SETTINGCHANGE:
		// タスクバーの明暗が変わったら、トレイのアイコンを描き直す
		a.refreshTray()
		return 0, false
	case w32.WM_DISPLAYCHANGE:
		// 画面の構成が変わったら、帯を画面の中へ置き直す
		a.strip.render()
		return 0, false
	case w32.WM_QUERYENDSESSION:
		return 1, true
	case w32.WM_ENDSESSION:
		if wParam != 0 {
			a.tray.Remove()
			flushLog(time.Second)
		}
		return 0, true
	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
		return 0, true
	}
	return 0, false
}

// devOpen はHACHIBU_OPEN（strip-menu、tray-menu、recorder、consent、update）で開くもの
func (a *App) devOpen(what string) {
	switch what {
	case "strip-menu":
		a.showMenu(a.buildStripMenu, w32.POINT{X: a.strip.frame.Left, Y: a.strip.frame.Top})
	case "tray-menu":
		if areas := w32.WorkAreas(); len(areas) > 0 {
			a.showMenu(a.buildTrayMenu, w32.POINT{X: areas[0].Right - 40, Y: areas[0].Bottom - 4})
		}
	case "recorder":
		a.showRecorder(a.recorder, a.cycleRecorder)
	case "consent":
		a.askConsent()
	case "update":
		// 確認から取得、置き換え、開き直しまでを、メニューや通知を押さずに通す
		if a.updates != nil {
			a.updates.installWhenReady = true
			a.updates.Check(false)
		}
	}
}

func (a *App) quit() {
	a.tray.Remove()
	w32.PostQuitMessage(0)
}
