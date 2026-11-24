//go:build windows

package app

import (
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// トレイのアイコンと帯の右クリックで開くメニューの中身。並びはmacOS向けのMenuBuilderとStatusItemControllerに合わせる

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
	a.menuClosedAt = time.Now()
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

// updateAvailableItems は、新しいバージョンが出ていれば、メニューの先頭に置き換えて開き直す項目を出す。
// その場で置き換えられない場所から動いているときは、ダウンロードのページを開く項目にする
func (a *App) updateAvailableItems(m *menu) {
	if a.updates == nil || a.updates.Available == nil {
		return
	}
	r := *a.updates.Available
	switch {
	case a.updates.Preparing:
		m.info(L.UpdatePreparing(r.Version), nil, 0)
	case a.updates.CanInstall(r):
		m.item(L.UpdateInstall(r.Version), false, a.updates.Install)
	default:
		m.item(L.UpdateAvailable(r.Version), false, func() { openURL(r.URL) })
	}
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

// updatesMenu は「アップデート」。動いているバージョン、自動確認の切り替え、今すぐ確認
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
