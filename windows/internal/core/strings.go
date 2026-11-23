package core

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Language は画面の言語。利用者がメニューで選んだ言語、無ければWindowsの表示言語（日本語なら日本語、それ以外は英語）。
type Language string

const (
	English  Language = "en"
	Japanese Language = "ja"
)

var AllLanguages = []Language{English, Japanese}

// DetectLanguage は言語を決める。overrideはHACHIBU_LANG（撮影用）で、どれよりも優先する。
func DetectLanguage(preferred []string, saved, override string) Language {
	for _, v := range []string{override, saved} {
		for _, l := range AllLanguages {
			if v == string(l) {
				return l
			}
		}
	}
	if len(preferred) > 0 && strings.HasPrefix(preferred[0], "ja") {
		return Japanese
	}
	return English
}

// NativeName は、どの言語の画面でもその言語自身の名前で出す。
func (l Language) NativeName() string {
	if l == Japanese {
		return "日本語"
	}
	return "English"
}

const (
	AppName           = "Hachibu"
	StatusPlaceholder = "CC …"
)

func OpacityName(v float64) string {
	return fmt.Sprintf("%d%%", int(math.Round(v*100)))
}

// Strings は画面に出る文言の表（i18nの対象はすべてここ）。
type Strings struct {
	Language Language

	MenuLayout              string
	MenuOpacity             string
	MenuTray                string
	MenuThresholds          string
	ThresholdWarningHeader  string
	ThresholdCriticalHeader string
	ThresholdChoice         func(int) string
	MenuSummon              string
	MenuChangeShortcut      string
	MenuCycleLayout         string
	MenuSetCycleShortcut    string
	RecorderCycleTitle      string
	MenuShortcutOff         string
	MenuResetPosition       string
	MenuOpenLog             string
	MenuQuit                string
	MenuNoUsage             string
	MenuHotKeyUnavailable   func(string) string

	RecorderTitle    string
	RecorderPrompt   string
	RecorderCurrent  func(string) string
	RecorderRejected string
	RecorderTaken    func(string) string
	RecorderReserved func(shortcut, name string) string
	RecorderVerify   func(string) string
	RecorderVerified func(string) string
	RecorderSaved    func(string) string
	RecorderSave     string
	// RecorderAppProne はCtrlだけ、Altだけの組み合わせに添える注意
	RecorderAppProne string
	// SystemShortcutName はWindowsのショートカットの名前（ShortcutConflictsの番号から）
	SystemShortcutName func(string) string
	RecorderDisable    string
	RecorderCancel     string

	LayoutName    func(StripLayout) string
	TrayStyleName func(TrayStyle) string

	BasicNoUsage   string
	MenuUsageAPI   string
	UsageAPIFailed func(string) string
	ConsentTitle   string
	ConsentWhat    string
	ConsentRisk    string
	ConsentPrivacy string
	ConsentWithout string
	// ConsentTerms は規約のページへのリンクの文言
	ConsentTerms   string
	ConsentEnable  string
	ConsentDecline string

	LimitFiveHour string
	LimitWeekly   string
	LimitLine     func(name string, percent int, resets string) string
	// UsageAsOf は使用率が古いときに添える一文（引数は時刻）
	UsageAsOf         func(string) string
	MenuLaunchAtLogin string
	// LoginItemDisabledBySystem は、Windowsの「スタートアップ」で利用者がオフにしているときにメニューに出す案内
	LoginItemDisabledBySystem string
	LoginItemFailed           func(string) string

	// ResetSameDay とResetOtherDay はリセット時刻の書式。当日は時刻だけ、それ以外は曜日と日付を添える
	ResetSameDay  func(time.Time) string
	ResetOtherDay func(time.Time) string

	MenuLanguage string
	// 「表示の読み方」。帯の記号と色の意味を並べる
	MenuLegend       string
	LegendContext    func(string) string
	LegendLimit      func(letter, name string) string
	LegendModelLimit func(letter, name string) string
	LegendWarning    func(int) string
	LegendCritical   func(int) string
	// TrayHidden はトレイのアイコンがWindowsに隠されているときに、帯のメニューに出す案内
	TrayHidden string
	// LanguageSystem の引数はWindowsの表示言語から決まる言語の名前
	LanguageSystem func(string) string

	// 操作ログ（%LOCALAPPDATA%\Hachibu\actions.log）の文言。書いた時点の画面の言語で残る
	LogShortcutCommitted func(string) string
	LogShortcutDisabled  string
	LogShortcutFailed    func(string) string
	LogUsageAPIEnabled   string
	LogUsageAPIDeclined  string
	LogUsageAPIDisabled  string
	LogLoginItemOn       string
	LogLoginItemOff      string
	LogOpened            string
	LogLanguageChanged   func(string) string
	// LogClaudeDirs は会話記録を読む場所が変わったときに残す（引数は場所の一覧）
	LogClaudeDirs func(string) string
	// SettingsSaveFailed は設定をファイルに書けなかったとき（引数は理由）
	SettingsSaveFailed func(string) string

	// 新しいバージョンの確認
	MenuUpdates             string
	UpdateCurrentVersion    func(string) string
	MenuUpdateAutoCheck     string
	MenuUpdateCheckNow      string
	UpdateAvailable         func(string) string
	UpdateNotificationTitle func(string) string
	UpdateNotificationBody  string
	UpdateUpToDate          func(string) string
	UpdateCheckFailed       func(string) string
	LogUpdateAvailable      func(string) string
	LogUpdateAutoCheckOn    string
	LogUpdateAutoCheckOff   string
}

var enWeekdays = [...]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
var jaWeekdays = [...]string{"日", "月", "火", "水", "木", "金", "土"}

func clock(t time.Time) string {
	return fmt.Sprintf("%d:%02d", t.Hour(), t.Minute())
}

var En = &Strings{
	Language:                English,
	MenuLayout:              "Display",
	MenuOpacity:             "Opacity",
	MenuTray:                "Tray Icon",
	MenuThresholds:          "Color Thresholds",
	ThresholdWarningHeader:  "Yellow when usage is",
	ThresholdCriticalHeader: "Red when usage is",
	ThresholdChoice:         func(p int) string { return fmt.Sprintf("%d%% or more (%d%% left)", p, 100-p) },
	MenuSummon:              "Show Strip",
	MenuChangeShortcut:      "Change Shortcut…",
	MenuCycleLayout:         "Switch to Next Display",
	MenuSetCycleShortcut:    "Shortcut for Switching Display…",
	RecorderCycleTitle:      "Shortcut to switch the display",
	MenuShortcutOff:         "Shortcut is off",
	MenuResetPosition:       "Reset Position",
	MenuOpenLog:             "Open Action Log",
	MenuQuit:                "Quit Hachibu",
	MenuNoUsage:             "Usage is not available yet",
	MenuHotKeyUnavailable:   func(s string) string { return s + " is already used by another app" },
	RecorderTitle:           "Shortcut to show the strip",
	RecorderPrompt:          "Press the new shortcut. It needs Ctrl or Alt. Esc cancels.",
	RecorderCurrent: func(s string) string {
		if s == "" {
			s = "none"
		}
		return "Current: " + s
	},
	RecorderRejected: "Add Ctrl or Alt to the key.",
	RecorderTaken:    func(s string) string { return s + " is already used by another app." },
	RecorderReserved: func(s, name string) string {
		return fmt.Sprintf("%s is assigned to \"%s\" in Windows, so it would never reach Hachibu.", s, name)
	},
	RecorderVerify: func(s string) string {
		return "Not saved yet. Press " + s + " once more to confirm that it arrives; then you can click Save. If nothing happens, Windows or another app takes it first. Press a different combination, or Cancel to keep the previous one."
	},
	RecorderVerified: func(s string) string {
		return s + " arrives. Click Save to use it, or Cancel to keep the previous one."
	},
	RecorderSaved:    func(s string) string { return "Saved " + s + "." },
	RecorderSave:     "Save",
	RecorderAppProne: "Shortcuts with only Ctrl or only Alt often collide with shortcuts in apps. Using both Ctrl and Alt is safer.",
	SystemShortcutName: func(id string) string {
		switch id {
		case "builtin:taskManager":
			return "Task Manager"
		case "builtin:appSwitcher":
			return "Switch between apps"
		case "builtin:startMenu":
			return "Open Start"
		case "builtin:windowCycle":
			return "Cycle through windows"
		case "builtin:closeWindow":
			return "Close the window"
		case "builtin:windowMenu":
			return "Open the window menu"
		case "builtin:imeToggle":
			return "Switch the input mode"
		case "builtin:windowScreenshot":
			return "Screenshot of the active window"
		case "builtin:securityScreen":
			return "Security options"
		}
		return "a keyboard shortcut (#" + id + ")"
	},
	RecorderDisable: "Turn Off",
	RecorderCancel:  "Cancel",
	LayoutName: func(l StripLayout) string {
		switch l {
		case LayoutFull:
			return "Standard"
		case LayoutUsage:
			return "Usage only"
		case LayoutTab:
			return "Tucked (handle only)"
		}
		return "Hidden (show with the shortcut or the tray icon)"
	},
	TrayStyleName: func(t TrayStyle) string {
		switch t {
		case TrayGauge:
			return "Weekly gauge"
		case TrayPercent:
			return "Weekly usage as a number"
		}
		return "Icon only"
	},
	BasicNoUsage:   "Usage is not known yet. Turn on Get Usage from the API, or set up statusline.sh as described in the README.",
	MenuUsageAPI:   "Get Usage from the API…",
	UsageAPIFailed: func(s string) string { return "Could not get usage from the API (" + s + ")" },
	ConsentTitle:   "Get usage from Anthropic's server?",
	ConsentWhat:    "If you turn this on, Hachibu uses the login that Claude Code saved on this PC (the \".credentials.json\" file in the .claude folder, or the one inside WSL if you use Claude Code there) to ask Anthropic's usage endpoint every 5 minutes. The 5-hour, weekly, and per-model weekly usage then stay up to date even when you are not using the terminal.",
	ConsentRisk:    "Anthropic's terms for Claude Code say that its OAuth login is designed to support \"ordinary use of Claude Code and other native Anthropic applications\", and that developers \"may not collect, store, or intermediate Claude.ai credentials or session tokens\". Hachibu is not an Anthropic application, so using this may be treated as a violation of those terms. Anthropic says it may enforce them without prior notice, and any measures would apply to your account. The usage endpoint is also not publicly documented and may stop working.",
	ConsentPrivacy: "The login is used only for this request. It is not saved or logged, and nothing else is sent.",
	ConsentWithout: "If you leave it off, Hachibu shows only the 5-hour and weekly usage from Claude Code's statusLine, which updates while you use Claude Code in the terminal. You can change this later from the tray icon or the strip's right-click menu.",
	ConsentTerms:   "Read the terms (Claude Code: Legal and compliance)",
	ConsentEnable:  "Turn On",
	ConsentDecline: "Not Now",
	LimitFiveHour:  "5-hour",
	LimitWeekly:    "Weekly",
	LimitLine: func(name string, percent int, resets string) string {
		head := fmt.Sprintf("%s %d%%", name, percent)
		if resets == "" {
			return head
		}
		return head + " (resets " + resets + ")"
	},
	UsageAsOf:                 func(s string) string { return "Last updated " + s },
	MenuLaunchAtLogin:         "Launch at Sign-in",
	LoginItemDisabledBySystem: "Turned off in Settings › Apps › Startup. Open Settings…",
	LoginItemFailed:           func(s string) string { return "Could not change the startup setting: " + s },
	ResetSameDay:              clock,
	ResetOtherDay: func(t time.Time) string {
		return fmt.Sprintf("%s %d/%d %s", enWeekdays[t.Weekday()], int(t.Month()), t.Day(), clock(t))
	},
	MenuLanguage:     "Language",
	MenuLegend:       "How to Read the Strip",
	LegendContext:    func(s string) string { return "Dot after the model name: " + s + " context" },
	LegendLimit:      func(letter, name string) string { return letter + ": " + name + " limit" },
	LegendModelLimit: func(letter, name string) string { return letter + ": weekly limit for " + name },
	LegendWarning:    func(p int) string { return fmt.Sprintf("%d%% or more", p) },
	LegendCritical:   func(p int) string { return fmt.Sprintf("%d%% or more", p) },
	TrayHidden:       "The tray icon is hidden by Windows. Open Taskbar Settings…",
	LanguageSystem:   func(s string) string { return "Same as Windows (" + s + ")" },

	LogShortcutCommitted: func(s string) string { return "Set the shortcut to " + s + " (confirmed that it arrives)" },
	LogShortcutDisabled:  "Turned off the shortcut",
	LogShortcutFailed:    func(s string) string { return "Could not register the shortcut " + s },
	LogUsageAPIEnabled:   "Turned on getting usage from the API",
	LogUsageAPIDeclined:  "Left getting usage from the API off",
	LogUsageAPIDisabled:  "Stopped getting usage from the API",
	LogLoginItemOn:       "Set Hachibu to launch at sign-in",
	LogLoginItemOff:      "Stopped launching at sign-in",
	LogOpened:            "Opened the action log",
	LogLanguageChanged:   func(s string) string { return "Changed the language to " + s },
	LogClaudeDirs:        func(s string) string { return "Reading Claude Code data from: " + s },
	SettingsSaveFailed: func(s string) string {
		return "Could not save the setting (" + s + "). The change applies only until Hachibu quits."
	},

	MenuUpdates:             "Updates",
	UpdateCurrentVersion:    func(s string) string { return "This version: " + s },
	MenuUpdateAutoCheck:     "Check for Updates Automatically",
	MenuUpdateCheckNow:      "Check Now",
	UpdateAvailable:         func(s string) string { return "Hachibu " + s + " is available. Open the Download Page…" },
	UpdateNotificationTitle: func(s string) string { return "Hachibu " + s + " is available" },
	UpdateNotificationBody:  "Click to open the download page.",
	UpdateUpToDate:          func(s string) string { return "Hachibu " + s + " is the latest version." },
	UpdateCheckFailed:       func(s string) string { return "Could not check for updates (" + s + ")" },
	LogUpdateAvailable:      func(s string) string { return "Found a new version, " + s },
	LogUpdateAutoCheckOn:    "Turned on checking for updates automatically",
	LogUpdateAutoCheckOff:   "Turned off checking for updates automatically",
}

var Ja = &Strings{
	Language:                Japanese,
	MenuLayout:              "表示",
	MenuOpacity:             "不透明度",
	MenuTray:                "トレイのアイコン",
	MenuThresholds:          "色の閾値",
	ThresholdWarningHeader:  "黄にする使用率",
	ThresholdCriticalHeader: "赤にする使用率",
	ThresholdChoice:         func(p int) string { return fmt.Sprintf("%d%%以上（残り%d%%以下）", p, 100-p) },
	MenuSummon:              "バーを呼び出す",
	MenuChangeShortcut:      "ショートカットを変更…",
	MenuCycleLayout:         "次の表示に切り替える",
	MenuSetCycleShortcut:    "表示切り替えのショートカット…",
	RecorderCycleTitle:      "表示を切り替えるショートカット",
	MenuShortcutOff:         "ショートカットは無効です",
	MenuResetPosition:       "位置を初期状態に戻す",
	MenuOpenLog:             "操作ログを開く",
	MenuQuit:                "Hachibuを終了",
	MenuNoUsage:             "使用率はまだ取得できていません",
	MenuHotKeyUnavailable:   func(s string) string { return "ショートカット" + s + "は他のアプリが使っています" },
	RecorderTitle:           "バーを呼び出すショートカット",
	RecorderPrompt:          "新しい組み合わせを押してください。CtrlかAltのどちらかを含めます。Escで取り消します。",
	RecorderCurrent: func(s string) string {
		if s == "" {
			s = "なし"
		}
		return "現在：" + s
	},
	RecorderRejected: "CtrlかAltと一緒に押してください",
	RecorderTaken:    func(s string) string { return s + "は他のアプリが使っているため登録できません" },
	RecorderReserved: func(s, name string) string {
		return s + "はWindowsの「" + name + "」に割り当てられているため、押してもHachibuに届きません"
	},
	RecorderVerify: func(s string) string {
		return "まだ保存していません。届くかを確かめるため、もう一度" + s + "を押してください。届いたら「保存」を押せるようになります。反応しない場合は、Windowsか他のアプリが先に受け取っています。別の組み合わせを押すか、キャンセルで元の組み合わせに戻してください。"
	},
	RecorderVerified: func(s string) string {
		return s + "が届くことを確かめました。「保存」を押すと確定します。キャンセルすると元の組み合わせに戻ります。"
	},
	RecorderSaved:    func(s string) string { return s + "を保存しました" },
	RecorderSave:     "保存",
	RecorderAppProne: "CtrlだけやAltだけの組み合わせは、アプリのショートカットと重なりやすいです。CtrlとAltを両方使うと安全です。",
	SystemShortcutName: func(id string) string {
		switch id {
		case "builtin:taskManager":
			return "タスク マネージャー"
		case "builtin:appSwitcher":
			return "アプリの切り替え"
		case "builtin:startMenu":
			return "スタートを開く"
		case "builtin:windowCycle":
			return "ウインドウを順に切り替える"
		case "builtin:closeWindow":
			return "ウインドウを閉じる"
		case "builtin:windowMenu":
			return "ウインドウのメニューを開く"
		case "builtin:imeToggle":
			return "入力モードの切り替え"
		case "builtin:windowScreenshot":
			return "作業中のウインドウのスクリーンショット"
		case "builtin:securityScreen":
			return "セキュリティ オプション"
		}
		return "キーボードショートカット（番号" + id + "）"
	},
	RecorderDisable: "無効にする",
	RecorderCancel:  "キャンセル",
	LayoutName: func(l StripLayout) string {
		switch l {
		case LayoutFull:
			return "標準"
		case LayoutUsage:
			return "使用率だけ"
		case LayoutTab:
			return "端に収納（つまみだけ）"
		}
		return "隠す（ショートカットかトレイのアイコンで出す）"
	},
	TrayStyleName: func(t TrayStyle) string {
		switch t {
		case TrayGauge:
			return "週枠のメーター"
		case TrayPercent:
			return "週枠の使用率を数字で"
		}
		return "アイコンのみ"
	},
	BasicNoUsage:   "使用率はまだ分かりません。「使用率をAPIから取得」を有効にするか、READMEの手順でstatusline.shを設定してください",
	MenuUsageAPI:   "使用率をAPIから取得…",
	UsageAPIFailed: func(s string) string { return "APIから使用率を取得できませんでした（" + s + "）" },
	ConsentTitle:   "使用率をAnthropicのサーバーから取得しますか？",
	ConsentWhat:    "有効にすると、Claude CodeがこのPCに保存したログイン情報（.claudeフォルダの「.credentials.json」。WSLでClaude Codeを使っている場合はWSL側のもの）を使い、5分ごとにAnthropicの使用率のエンドポイントへ問い合わせます。5時間枠、週枠、モデル別の週枠が、ターミナルを使っていないときも更新されます。",
	ConsentRisk:    "Claude Codeの規約は、OAuthによるログインを「ordinary use of Claude Code and other native Anthropic applications」（Claude Codeと、Anthropic純正のアプリの通常の利用）のためのものとし、開発者は「may not collect, store, or intermediate Claude.ai credentials or session tokens」（Claude.aiのログイン情報やセッショントークンを収集、保存、仲介してはならない）としています。HachibuはAnthropicのアプリではないため、この機能を使うと規約違反と判断されるおそれがあります。Anthropicは予告なく制限を執行しうるとしており、その措置はあなたのアカウントに及びます。使用率のエンドポイント自体も公開されておらず、予告なく使えなくなることがあります。",
	ConsentPrivacy: "ログイン情報はこの問い合わせにだけ使い、保存も記録もしません。ほかには何も送りません。",
	ConsentWithout: "有効にしない場合は、Claude CodeのstatusLineから5時間枠と週枠だけを表示します。値が更新されるのは、ターミナルでClaude Codeを使っている間です。あとからトレイのアイコンかバーの右クリックメニューで変えられます。",
	ConsentTerms:   "規約を読む（Claude Code: Legal and compliance）",
	ConsentEnable:  "有効にする",
	ConsentDecline: "今はしない",
	LimitFiveHour:  "5時間枠",
	LimitWeekly:    "週枠",
	LimitLine: func(name string, percent int, resets string) string {
		// 日本語と数字の間には空白を入れない（"週枠75%"）。英字の名前なら空ける（"Fable 55%"）
		sep := ""
		if name != "" && name[len(name)-1] < 0x80 {
			sep = " "
		}
		head := fmt.Sprintf("%s%s%d%%", name, sep, percent)
		if resets == "" {
			return head
		}
		return head + "（" + resets + "にリセット）"
	},
	UsageAsOf:                 func(s string) string { return s + "時点の値" },
	MenuLaunchAtLogin:         "サインイン時に起動",
	LoginItemDisabledBySystem: "Windowsの「設定」＞「アプリ」＞「スタートアップ」でオフになっています。設定を開く…",
	LoginItemFailed:           func(s string) string { return "スタートアップの設定を変更できませんでした: " + s },
	ResetSameDay:              clock,
	ResetOtherDay: func(t time.Time) string {
		return fmt.Sprintf("%d/%d(%s) %s", int(t.Month()), t.Day(), jaWeekdays[t.Weekday()], clock(t))
	},
	MenuLanguage:     "言語",
	MenuLegend:       "表示の読み方",
	LegendContext:    func(s string) string { return "モデル名の後ろの点：" + s + "コンテキスト" },
	LegendLimit:      func(letter, name string) string { return letter + "：" + name },
	LegendModelLimit: func(letter, name string) string { return letter + "：" + name + "の週枠" },
	LegendWarning:    func(p int) string { return fmt.Sprintf("%d%%以上", p) },
	LegendCritical:   func(p int) string { return fmt.Sprintf("%d%%以上", p) },
	TrayHidden:       "トレイのアイコンがWindowsに隠されています。「タスク バー」の設定を開く…",
	LanguageSystem:   func(s string) string { return "Windowsに合わせる（" + s + "）" },

	LogShortcutCommitted: func(s string) string {
		return "ショートカットを" + s + "にしました（届くことを確認済み）"
	},
	LogShortcutDisabled: "ショートカットを無効にしました",
	LogShortcutFailed:   func(s string) string { return "ショートカット" + s + "の登録に失敗しました" },
	LogUsageAPIEnabled:  "使用率APIからの取得を有効にしました",
	LogUsageAPIDeclined: "使用率APIからの取得は有効にしませんでした",
	LogUsageAPIDisabled: "使用率APIからの取得をやめました",
	LogLoginItemOn:      "サインイン時に起動するようにしました",
	LogLoginItemOff:     "サインイン時の起動をやめました",
	LogOpened:           "操作ログを開きました",
	LogLanguageChanged:  func(s string) string { return "言語を" + s + "にしました" },
	LogClaudeDirs:       func(s string) string { return "Claude Codeのデータを読む場所：" + s },
	SettingsSaveFailed: func(s string) string {
		return "設定を保存できませんでした（" + s + "）。変えた内容は、Hachibuを終了するまでしか効きません"
	},

	MenuUpdates:          "アップデート",
	UpdateCurrentVersion: func(s string) string { return "このバージョン：" + s },
	MenuUpdateAutoCheck:  "新しいバージョンを自動で確認",
	MenuUpdateCheckNow:   "今すぐ確認",
	UpdateAvailable: func(s string) string {
		return "Hachibu " + s + "が出ています。ダウンロードのページを開く…"
	},
	UpdateNotificationTitle: func(s string) string { return "Hachibu " + s + "が出ました" },
	UpdateNotificationBody:  "クリックするとダウンロードのページを開きます",
	UpdateUpToDate:          func(s string) string { return "Hachibu " + s + "は最新のバージョンです" },
	UpdateCheckFailed:       func(s string) string { return "新しいバージョンを確認できませんでした（" + s + "）" },
	LogUpdateAvailable:      func(s string) string { return "新しいバージョン" + s + "が出ていました" },
	LogUpdateAutoCheckOn:    "新しいバージョンの自動確認を有効にしました",
	LogUpdateAutoCheckOff:   "新しいバージョンの自動確認をやめました",
}

// StringsFor は言語の文言の表。
func StringsFor(l Language) *Strings {
	if l == Japanese {
		return Ja
	}
	return En
}
