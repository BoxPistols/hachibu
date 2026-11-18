//go:build windows

// Package app はHachibuのWindows版の本体。帯（画面の最前面の小さなバー）、トレイのアイコン、メニュー、
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
	// 同じアプリの別のコピーがすでに動いていれば、そちらに帯を出させて終わる（帯を重ねて出さない）
	if handOff() {
		return 0
	}
	a := &App{prefs: loadPrefs(), demo: isDemo()}
	// 文言を読む部品を作る前に、メニューで選んだ言語にしておく
	applyLanguage(a.prefs.Language())

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
		a.updates = newUpdater(a.prefs, a.tray)
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

// devOpen はHACHIBU_OPEN（strip-menu、tray-menu、recorder、consent）で開くもの
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
	}
}

func (a *App) quit() {
	a.tray.Remove()
	w32.PostQuitMessage(0)
}

// MARK: - メニュー

// showMenu はメニューを開き、選ばれた項目を動かす。閉じるまで戻らない。
// トレイのメニューは、アプリが前面に無いと外を押しても閉じないので、開く間だけ主な窓を前面にし、閉じたら元の窓へ戻す
func (a *App) showMenu(build func(m *menu), at w32.POINT) {
	if a.activeMenu != nil {
		return
	}
	prev := w32.GetForegroundWindow()
	if prev != mainHwnd {
		a.lastForeground = prev
	}
	scale := float32(w32.DpiForMonitor(w32.MonitorFromPoint(at, w32.MONITOR_DEFAULTTONEAREST))) / 96
	set := newMenuSet(scale)
	root := set.root()
	build(root)
	a.activeMenu = set
	a.strip.MenuOpened()
	w32.SetForegroundWindow(mainHwnd)
	cmd := w32.TrackPopupMenu(root.h, w32.TPM_RETURNCMD|w32.TPM_RIGHTBUTTON, at.X, at.Y, mainHwnd)
	w32.PostMessage(mainHwnd, w32.WM_NULL, 0, 0)
	a.activeMenu = nil
	w32.DestroyMenu(root.h)
	set.free()
	// 選んだ項目を先に動かしてから閉じた扱いにする（仮の表示→元の値→選んだ値とちらつかないように）
	if action := set.actions[cmd]; action != nil {
		action()
	}
	a.strip.MenuClosed()
	// 選んだ項目が窓を開いていなければ、メニューを開く前に使っていた窓へ戻す
	if fg := w32.GetForegroundWindow(); (fg == mainHwnd || fg == 0) && a.lastForeground != 0 && w32.IsWindow(a.lastForeground) {
		w32.SetForegroundWindow(a.lastForeground)
	}
}

// menuSelect は項目に乗せたとき。表示を切り替える項目なら帯に仮に反映し、ほかの項目に移ったら消す
func (a *App) menuSelect(wParam, lParam uintptr) {
	if a.activeMenu == nil {
		return
	}
	flags := uint32(w32.HiWord(wParam))
	id := uint32(w32.LoWord(wParam))
	switch {
	case flags == 0xFFFF && lParam == 0:
		// メニューが閉じる知らせ。選んだ項目の処理が後に続くので、ここでは消さない
	case flags&w32.MF_POPUP != 0:
		a.strip.ClearPreview()
	case flags&w32.MF_SEPARATOR != 0 || id == 0:
	default:
		if preview := a.activeMenu.previews[id]; preview != nil {
			preview()
		} else {
			a.strip.ClearPreview()
		}
	}
}

// buildStripMenu は帯の右クリック。表示の切り替えをその場で行えるよう、モードは入れ子にせず並べる。
// トレイのアイコンはWindowsに隠されることがあるので、帯だけで全部の操作に届くようにする
func (a *App) buildStripMenu(m *menu) {
	a.updateAvailableItems(m)
	a.usageItems(m)
	a.usageAPIItems(m)
	m.separator()
	a.layoutItems(m)
	a.opacityMenu(m)
	a.thresholdsMenu(m)
	m.separator()
	a.shortcutItems(m)
	m.item(L.MenuResetPosition, false, a.strip.ResetPosition)
	a.languageMenu(m)
	a.loginItems(m)
	a.updatesMenu(m)
	a.openLogItem(m)
	a.legendMenu(m)
	if !a.demo && m.set.dump == nil && trayIconHidden() {
		m.separator()
		m.item(L.TrayHidden, false, func() { openURL("ms-settings:taskbar") })
	}
	m.separator()
	m.item(L.MenuQuit, false, a.quit)
}

// buildTrayMenu はトレイのアイコンのメニュー
func (a *App) buildTrayMenu(m *menu) {
	a.updateAvailableItems(m)
	a.usageItems(m)
	a.usageAPIItems(m)
	m.separator()
	a.layoutItems(m.submenu(L.MenuLayout))
	a.opacityMenu(m)
	a.thresholdsMenu(m)
	tray := m.submenu(L.MenuTray)
	for _, style := range core.AllTrayStyles {
		tray.choice(L.TrayStyleName(style), a.prefs.TrayStyle() == style, nil, func() {
			a.prefs.Set("trayStyle", string(style))
			a.refreshTray()
		})
	}
	m.separator()
	a.shortcutItems(m)
	a.languageMenu(m)
	a.loginItems(m)
	a.updatesMenu(m)
	m.item(L.MenuResetPosition, false, a.strip.ResetPosition)
	a.openLogItem(m)
	a.legendMenu(m)
	m.separator()
	m.item(L.MenuQuit, false, a.quit)
}

// updateAvailableItems は、新しい版が出ていれば、メニューの先頭にダウンロードのページを開く項目を出す
func (a *App) updateAvailableItems(m *menu) {
	if a.updates == nil || a.updates.Available == nil {
		return
	}
	r := *a.updates.Available
	m.item(L.UpdateAvailable(r.Version), false, func() { openURL(r.URL) })
	m.separator()
}

// usageItems は使用率の行。黄や赤の段階にある枠には、その色の点を付ける
func (a *App) usageItems(m *menu) {
	// 帯では点で示しているコンテキスト長を、ここでは文字で書く。帯の点と同じ印を付ける
	if slot, ok := a.strip.StatusSlot(); ok {
		if spelled, ok := core.SpelledOut(slot.Text); ok {
			dot := contextDotColor
			m.info(spelled, &dot, 6)
		}
	}
	if len(a.display.Limits) == 0 {
		m.info(L.MenuNoUsage, nil, 0)
		return
	}
	now := time.Now()
	for _, l := range a.display.Limits {
		if color, ok := levelBackground(a.strip.Thresholds.Level(l.Percent)); ok {
			m.info(l.Line(L, now), &color, 8)
		} else {
			m.info(l.Line(L, now), nil, 0)
		}
	}
	// ターミナルでClaude Codeを使っていない間は値が更新されない。いつ時点の値かを添える
	if core.IsStale(a.display.UsageAsOf, now) {
		m.info(L.UsageAsOf(core.ResetText(a.display.UsageAsOf, now, L)), nil, 0)
	}
}

// usageAPIItems は撮影用の架空の値では出さない。有効にするときは、何を使いどんなリスクがあるかを先に示す
func (a *App) usageAPIItems(m *menu) {
	if a.demo {
		return
	}
	enabled := a.prefs.UsageAPIEnabled()
	m.item(L.MenuUsageAPI, enabled, func() {
		if enabled {
			// 先に止めてから保存する。保存できなくても、動いている間は問い合わせない
			a.source.SetUsageAPI(false)
			if err := a.prefs.SetChecked("usageAPIConsent", false); err != nil {
				// 保存できないと、次に起動したときに有効に戻ってしまう。黙って成功にしない
				logAction(L.SettingsSaveFailed(err.Error()))
				w32.MessageBox(0, L.SettingsSaveFailed(err.Error()), core.AppName, messageBoxFlags)
				return
			}
			logAction(L.LogUsageAPIDisabled)
		} else {
			a.askConsent()
		}
	})
	if enabled && a.apiFailure != "" {
		m.info(L.UsageAPIFailed(a.apiFailure), nil, 0)
	}
}

func (a *App) askConsent() {
	enabled, answered := askUsageAPIConsent(L)
	if !answered {
		return
	}
	a.prefs.Set("usageAPIConsent", enabled)
	if enabled {
		logAction(L.LogUsageAPIEnabled)
	} else {
		logAction(L.LogUsageAPIDeclined)
	}
	if a.source != nil {
		a.source.SetUsageAPI(enabled)
	}
}

// layoutItems は項目に乗せると帯をそのモードで仮に表示し、選ぶと確定する
func (a *App) layoutItems(m *menu) {
	for _, layout := range core.AllLayouts {
		m.choice(L.LayoutName(layout), a.strip.RestLayout == layout,
			func() { a.strip.PreviewLayout(layout) },
			func() { a.strip.SetRestLayout(layout) })
	}
	m.separator()
	shortcut := ""
	if s := a.cycleKeys.Shortcut; s != nil && !a.cycleKeys.RegistrationFailed {
		shortcut = s.Display()
	}
	m.itemWithShortcut(L.MenuCycleLayout, shortcut, false, a.strip.CycleLayout)
	m.item(L.MenuSetCycleShortcut, false, func() { a.showRecorder(a.cycleRecorder, a.recorder) })
}

func (a *App) opacityMenu(m *menu) {
	sub := m.submenu(L.MenuOpacity)
	for _, v := range opacityChoices {
		sub.choice(core.OpacityName(v), a.strip.Opacity == v,
			func() { a.strip.PreviewOpacity(v) },
			func() { a.strip.SetOpacity(v) })
	}
}

// thresholdsMenu は黄と赤の閾値。赤が黄以下になる選択肢は選べなくする
func (a *App) thresholdsMenu(m *menu) {
	sub := m.submenu(L.MenuThresholds)
	cur := a.strip.Thresholds
	add := func(p int, selected bool, next core.Thresholds, ok bool) {
		if !ok {
			sub.disabled(L.ThresholdChoice(p), selected)
			return
		}
		sub.choice(L.ThresholdChoice(p), selected, func() { a.strip.PreviewThresholds(next) }, func() { a.strip.SetThresholds(next) })
	}
	sub.disabled(L.ThresholdWarningHeader, false)
	for _, p := range core.WarningChoices {
		next, ok := core.NewThresholds(p, cur.Critical)
		add(p, cur.Warning == p, next, ok)
	}
	sub.separator()
	sub.disabled(L.ThresholdCriticalHeader, false)
	for _, p := range core.CriticalChoices {
		next, ok := core.NewThresholds(cur.Warning, p)
		add(p, cur.Critical == p, next, ok)
	}
}

func (a *App) shortcutItems(m *menu) {
	s := a.hotKeys.Shortcut
	switch {
	case s != nil && !a.hotKeys.RegistrationFailed:
		// 右端の組み合わせは表示のため。実際に効くのはRegisterHotKeyの登録
		m.itemWithShortcut(L.MenuSummon, s.Display(), a.strip.Summoned, a.strip.ToggleSummon)
	case s != nil:
		m.item(L.MenuSummon, a.strip.Summoned, a.strip.ToggleSummon)
		m.info(L.MenuHotKeyUnavailable(s.Display()), nil, 0)
	default:
		m.item(L.MenuSummon, a.strip.Summoned, a.strip.ToggleSummon)
		m.info(L.MenuShortcutOff, nil, 0)
	}
	m.item(L.MenuChangeShortcut, false, func() { a.showRecorder(a.recorder, a.cycleRecorder) })
}

// showRecorder は記録の窓を開く。キー入力を受ける窓は1つだけにするので、もう一方が開いていれば先に閉じる
func (a *App) showRecorder(r, other *Recorder) {
	prev := a.lastForeground
	other.Close()
	r.Show(prev)
}

// languageMenu は画面の言語。Windowsに合わせるか、英語か日本語を選ぶ。次にメニューを開いたときから切り替わる
func (a *App) languageMenu(m *menu) {
	sub := m.submenu(L.MenuLanguage)
	saved := a.prefs.Language()
	system := core.DetectLanguage(preferredLanguages(), "", "")
	sub.choice(L.LanguageSystem(system.NativeName()), saved == "", nil, func() { a.setLanguage("") })
	for _, lang := range core.AllLanguages {
		sub.choice(lang.NativeName(), saved == string(lang), nil, func() { a.setLanguage(string(lang)) })
	}
}

func (a *App) setLanguage(code string) {
	if code == "" {
		a.prefs.Set("language", nil)
	} else {
		a.prefs.Set("language", code)
	}
	applyLanguage(code)
	logAction(L.LogLanguageChanged(L.Language.NativeName()))
	// ツールチップとトレイの文言も今の言語で組み立て直す
	a.display = core.Display{}
	a.applyState(a.state, a.dirs)
	a.refreshTray()
}

// loginItems はサインイン時の起動。失敗の理由は次にメニューを開いたときに出す
func (a *App) loginItems(m *menu) {
	enabled := loginItemEnabled()
	m.item(L.MenuLaunchAtLogin, enabled, func() {
		if err := setLoginItem(!enabled); err != nil {
			a.loginErr = err.Error()
			logAction(L.LoginItemFailed(a.loginErr))
			return
		}
		a.loginErr = ""
		if enabled {
			logAction(L.LogLoginItemOff)
		} else {
			logAction(L.LogLoginItemOn)
		}
	})
	if enabled && loginItemDisabledBySystem() {
		m.item(L.LoginItemDisabledBySystem, false, func() { openURL("ms-settings:startupapps") })
	}
	if a.loginErr != "" {
		m.info(L.LoginItemFailed(a.loginErr), nil, 0)
	}
}

// updatesMenu は「アップデート」。動いている版、自動確認の切り替え、今すぐ確認
func (a *App) updatesMenu(m *menu) {
	if a.updates == nil {
		return
	}
	sub := m.submenu(L.MenuUpdates)
	sub.info(L.UpdateCurrentVersion(Version), nil, 0)
	sub.item(L.MenuUpdateAutoCheck, a.prefs.UpdateAutoCheck(), func() {
		on := !a.prefs.UpdateAutoCheck()
		a.prefs.Set("updateAutoCheck", on)
		if on {
			logAction(L.LogUpdateAutoCheckOn)
		} else {
			logAction(L.LogUpdateAutoCheckOff)
		}
		a.updates.AutoCheckChanged()
	})
	sub.item(L.MenuUpdateCheckNow, false, func() { a.updates.Check(true) })
	if a.updates.LastFailure != "" {
		sub.info(L.UpdateCheckFailed(a.updates.LastFailure), nil, 0)
	}
}

func (a *App) openLogItem(m *menu) {
	m.item(L.MenuOpenLog, false, func() {
		logAction(L.LogOpened)
		// 書き込みは別のゴルーチンなので、ファイルができるのを待ってから開く。待つ間もメニューや帯は止めない
		go func() {
			flushLog(2 * time.Second)
			Post(func() { openURL(actionLogPath()) })
		}()
	})
}

// legendMenu は「表示の読み方」。帯に出ている記号と色だけを並べる
func (a *App) legendMenu(m *menu) {
	sub := m.submenu(L.MenuLegend)
	text := ""
	if slot, ok := a.strip.StatusSlot(); ok {
		text = slot.Text
	}
	for _, line := range core.Legend(text, a.display.Limits, a.strip.Thresholds, L) {
		switch line.Mark {
		case core.MarkContextDot:
			dot := contextDotColor
			sub.info(line.Text, &dot, 6)
		case core.MarkLevel:
			color, _ := levelBackground(line.Level)
			sub.info(line.Text, &color, 8)
		default:
			sub.info(line.Text, nil, 0)
		}
	}
}
