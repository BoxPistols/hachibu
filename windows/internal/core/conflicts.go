package core

// ReservedShortcut はWindowsが先に受け取るショートカット。登録できても押したときにアプリへ届かないか、
// 届いてもWindowsの大事な操作を奪うので、登録の前に弾く
type ReservedShortcut struct {
	ID   string
	VK   uint32
	Mods Modifiers
}

// BuiltInReserved は、どのPCでも効いているWindowsのショートカット（Windowsキーを使うものは記録しないので含めない）。
var BuiltInReserved = []ReservedShortcut{
	{"builtin:taskManager", VKEscape, ModCtrl | ModShift},  // タスク マネージャー
	{"builtin:appSwitcher", VKTab, ModAlt},                 // Alt+Tab
	{"builtin:appSwitcher", VKTab, ModAlt | ModShift},      // 逆順
	{"builtin:appSwitcher", VKTab, ModCtrl | ModAlt},       // 切り替えの画面を開いたままにする
	{"builtin:startMenu", VKEscape, ModCtrl},               // スタート
	{"builtin:windowCycle", VKEscape, ModAlt},              // ウインドウを順に切り替える
	{"builtin:closeWindow", VKF4, ModAlt},                  // ウインドウを閉じる
	{"builtin:windowMenu", VKSpace, ModAlt},                // ウインドウのメニュー
	{"builtin:imeToggle", VKOem3, ModAlt},                  // 英語配列のAlt+`
	{"builtin:imeToggle", VKKanji, ModAlt},                 // 日本語配列のAlt+半角/全角
	{"builtin:imeToggle", VKSpace, ModCtrl},                // 中国語などのIME
	{"builtin:windowScreenshot", VKSnapshot, ModAlt},       // 作業中のウインドウを撮る
	{"builtin:securityScreen", VKDelete, ModCtrl | ModAlt}, // Ctrl+Alt+Del
	{"builtin:securityScreen", VKDelete, ModCtrl | ModAlt | ModShift},
}

// Conflict は重なるWindowsのショートカットの番号。重ならなければfalse。
func Conflict(s Shortcut) (string, bool) {
	for _, r := range BuiltInReserved {
		if r.VK == s.VK && r.Mods == s.Mods {
			return r.ID, true
		}
	}
	return "", false
}

// IsAppShortcutProne は、CtrlだけやAltだけ（とShift）の組み合わせか。アプリのショートカットやメニューのアクセスキーと重なりやすく、
// 登録すると、どのアプリにいてもこちらが先に受け取るので、そのアプリの機能が使えなくなる
func IsAppShortcutProne(s Shortcut) bool {
	return s.Mods&ModCtrl == 0 || s.Mods&ModAlt == 0
}
