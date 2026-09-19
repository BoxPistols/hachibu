import Foundation

/// 画面の言語。利用者がメニューで選んだ言語、無ければmacOSの優先言語（日本語なら日本語、それ以外は英語）。
/// SLASHSTRIP_LANG=en|jaはどちらよりも優先する（撮影用）
public enum Language: String, CaseIterable {
    case en
    case ja

    public static func detect(preferred: [String] = Locale.preferredLanguages,
                              saved: String? = nil,
                              override: String? = ProcessInfo.processInfo.environment["SLASHSTRIP_LANG"]) -> Language {
        if let override, let lang = Language(rawValue: override) { return lang }
        if let saved, let lang = Language(rawValue: saved) { return lang }
        return preferred.first?.hasPrefix("ja") == true ? .ja : .en
    }

    /// メニューでは、どの言語の画面でもその言語自身の名前で出す
    public var nativeName: String {
        switch self {
        case .en: return "English"
        case .ja: return "日本語"
        }
    }
}

/// 画面に出る文言の表（i18nの対象はすべてここ）
public struct Strings {
    public let language: Language

    public let menuLayout: String
    public let menuOpacity: String
    public let menuMenuBar: String
    public let menuThresholds: String
    public let thresholdWarningHeader: String
    public let thresholdCriticalHeader: String
    public let thresholdChoice: (Int) -> String
    public let menuSummon: String
    public let menuChangeShortcut: String
    public let menuShortcutOff: String
    public let menuResetPosition: String
    public let menuOpenLog: String
    public let menuQuit: String
    public let menuNoUsage: String
    public let menuHotKeyUnavailable: (String) -> String


    public let recorderTitle: String
    public let recorderPrompt: String
    public let recorderCurrent: (String?) -> String
    public let recorderRejected: String
    public let recorderTaken: (String) -> String
    public let recorderReserved: (String, String) -> String
    public let recorderVerify: (String) -> String
    public let recorderVerified: (String) -> String
    public let recorderMenuProne: String
    /// macOSのショートカットの名前（com.apple.symbolichotkeysの番号から）
    public let systemShortcutName: (String) -> String
    public let recorderDisable: String
    public let recorderCancel: String

    public let layoutName: (StripLayout) -> String
    public let menuBarStyleName: (MenuBarStyle) -> String

    public let basicNoUsage: String
    public let menuUsageAPI: String
    public let usageAPIFailed: (String) -> String
    public let consentTitle: String
    public let consentWhat: String
    public let consentRisk: String
    public let consentPrivacy: String
    public let consentWithout: String
    public let consentEnable: String
    public let consentDecline: String
    public let dragHandle: String

    public let limitFiveHour: String
    public let limitWeekly: String
    public let limitLine: (String, Int, String?) -> String
    /// 使用率が古いときに添える一文（引数は時刻）
    public let usageAsOf: (String) -> String
    public let menuLaunchAtLogin: String
    public let loginItemNeedsApproval: String
    public let loginItemFailed: (String) -> String

    /// リセット時刻の書式。当日は時刻だけ、それ以外は曜日と日付を添える
    public let resetLocale: String
    public let resetSameDay: String
    public let resetOtherDay: String

    public let menuLanguage: String
    /// 引数はmacOSの言語から決まる言語の名前
    public let languageSystem: (String) -> String

    // 操作ログ（~/Library/Logs/Slashstrip/actions.log）の文言。書いた時点の画面の言語で残る
    public let logShortcutCommitted: (String) -> String
    public let logShortcutDisabled: String
    public let logShortcutFailed: (String) -> String
    public let logUsageAPIEnabled: String
    public let logUsageAPIDeclined: String
    public let logUsageAPIDisabled: String
    public let logLoginItemOn: String
    public let logLoginItemOff: String
    public let logOpened: String
    public let logLanguageChanged: (String) -> String

    public static let en = Strings(
        language: .en,
        menuLayout: "Display",
        menuOpacity: "Opacity",
        menuMenuBar: "Menu Bar",
        menuThresholds: "Color Thresholds",
        thresholdWarningHeader: "Yellow when usage is",
        thresholdCriticalHeader: "Red when usage is",
        thresholdChoice: { "\($0)% or more (\(100 - $0)% left)" },
        menuSummon: "Show Strip",
        menuChangeShortcut: "Change Shortcut…",
        menuShortcutOff: "Shortcut is off",
        menuResetPosition: "Reset Position",
        menuOpenLog: "Open Action Log",
        menuQuit: "Quit Slashstrip",
        menuNoUsage: "Usage is not available yet",
        menuHotKeyUnavailable: { "\($0) is already used by another app" },
        recorderTitle: "Shortcut to show the strip",
        recorderPrompt: "Press the new shortcut. It needs Command (⌘), Option (⌥), or Control (⌃). Esc cancels.",
        recorderCurrent: { "Current: " + ($0 ?? "none") },
        recorderRejected: "Add Command (⌘), Option (⌥), or Control (⌃) to the key.",
        recorderTaken: { "\($0) is already used by another app." },
        recorderReserved: { "\($0) is assigned to \"\($1)\" in macOS, so it would never reach Slashstrip." },
        recorderVerify: { "Not saved yet. Press \($0) once more to confirm that it arrives. If nothing happens, macOS or another app takes it first. Press a different combination, or Cancel to keep the previous one." },
        recorderVerified: { "\($0) works and is saved." },
        recorderMenuProne: "Shortcuts with only ⌘ often collide with app menu shortcuts. Adding ⌥ or ⌃ is safer.",
        systemShortcutName: { id in
            switch id {
            case "64": return "Show Spotlight search"
            case "65": return "Show Finder search window"
            case "60": return "Select the previous input source"
            case "61": return "Select next source in Input menu"
            case "28", "29", "30", "31", "184": return "Screenshots"
            case "27": return "Move focus to next window"
            case "32", "34": return "Mission Control"
            case "33", "35": return "Application windows"
            case "79", "80", "81", "82": return "Move between Spaces"
            case "98": return "Show Help menu"
            case "builtin:appSwitcher": return "App switcher"
            case "builtin:forceQuit": return "Force Quit"
            case "builtin:lockScreen": return "Lock Screen"
            default: return "a keyboard shortcut (#\(id))"
            }
        },
        recorderDisable: "Turn Off",
        recorderCancel: "Cancel",
        layoutName: { layout in
            switch layout {
            case .full: return "Standard"
            case .usage: return "Usage only"
            case .tab: return "Tucked (handle only)"
            case .hidden: return "Hidden (show with the shortcut or menu bar)"
            }
        },
        menuBarStyleName: { style in
            switch style {
            case .icon: return "Icon only"
            case .usage: return "Usage"
            case .status: return "Full status"
            }
        },
        basicNoUsage: "Usage is not known yet. Turn on Get Usage from the API, or set up statusline.sh as described in the README.",
        menuUsageAPI: "Get Usage from the API…",
        usageAPIFailed: { "Could not get usage from the API (\($0))" },
        consentTitle: "Get usage from Anthropic's server?",
        consentWhat: "If you turn this on, Slashstrip uses the login that Claude Code saved on this Mac (the \"Claude Code-credentials\" item in your keychain) to ask Anthropic's usage endpoint every 5 minutes. The 5-hour, weekly, and per-model weekly usage then stay up to date even when you are not using the terminal.",
        consentRisk: "This endpoint is not publicly documented, and we found nothing showing that Anthropic allows third-party apps to use it. It may stop working without notice, or Anthropic may consider it against its terms. Any consequence would fall on your account.",
        consentPrivacy: "The login is used only for this request. It is not saved or logged, and nothing else is sent. The first time, macOS may ask whether to allow access to the keychain item.",
        consentWithout: "If you leave it off, Slashstrip shows only the 5-hour and weekly usage from Claude Code's statusLine, which updates while you use Claude Code in the terminal. You can change this later from the menu bar.",
        consentEnable: "Turn On",
        consentDecline: "Not Now",
        dragHandle: "Drag to move",
        limitFiveHour: "5-hour",
        limitWeekly: "Weekly",
        limitLine: { name, percent, resets in
            let head = "\(name) \(percent)%"
            guard let resets else { return head }
            return "\(head) (resets \(resets))"
        },
        usageAsOf: { "Last updated \($0)" },
        menuLaunchAtLogin: "Launch at Login",
        loginItemNeedsApproval: "Allow Slashstrip in System Settings › Login Items",
        loginItemFailed: { "Could not change the login item: \($0)" },
        resetLocale: "en_US",
        resetSameDay: "H:mm",
        resetOtherDay: "EEE M/d H:mm",
        menuLanguage: "Language",
        languageSystem: { "Same as macOS (\($0))" },
        logShortcutCommitted: { "Set the shortcut to \($0) (confirmed that it arrives)" },
        logShortcutDisabled: "Turned off the shortcut",
        logShortcutFailed: { "Could not register the shortcut \($0)" },
        logUsageAPIEnabled: "Turned on getting usage from the API",
        logUsageAPIDeclined: "Left getting usage from the API off",
        logUsageAPIDisabled: "Stopped getting usage from the API",
        logLoginItemOn: "Set Slashstrip to launch at login",
        logLoginItemOff: "Stopped launching at login",
        logOpened: "Opened the action log",
        logLanguageChanged: { "Changed the language to \($0)" }
    )

    public static let ja = Strings(
        language: .ja,
        menuLayout: "表示",
        menuOpacity: "不透明度",
        menuMenuBar: "メニューバー",
        menuThresholds: "色の閾値",
        thresholdWarningHeader: "黄にする使用率",
        thresholdCriticalHeader: "赤にする使用率",
        thresholdChoice: { "\($0)%以上（残り\(100 - $0)%以下）" },
        menuSummon: "帯を呼び出す",
        menuChangeShortcut: "ショートカットを変更…",
        menuShortcutOff: "ショートカットは無効です",
        menuResetPosition: "位置を初期状態に戻す",
        menuOpenLog: "操作ログを開く",
        menuQuit: "Slashstripを終了",
        menuNoUsage: "使用率はまだ取得できていません",
        menuHotKeyUnavailable: { "ショートカット\($0)は他のアプリが使っています" },
        recorderTitle: "帯を呼び出すショートカット",
        recorderPrompt: "新しい組み合わせを押してください。Command（⌘）・Option（⌥）・Control（⌃）のどれかを含めます。Escで取り消します。",
        recorderCurrent: { "現在：" + ($0 ?? "なし") },
        recorderRejected: "Command（⌘）・Option（⌥）・Control（⌃）のどれかと一緒に押してください",
        recorderTaken: { "\($0)は他のアプリが使っているため登録できません" },
        recorderReserved: { "\($0)はmacOSの「\($1)」に割り当てられているため、押してもSlashstripに届きません" },
        recorderVerify: { "まだ保存していません。届くかを確かめるため、もう一度\($0)を押してください。反応しない場合は、macOSか他のアプリが先に受け取っています。別の組み合わせを押すか、キャンセルで元の組み合わせに戻してください。" },
        recorderVerified: { "\($0)が届くことを確かめて保存しました" },
        recorderMenuProne: "⌘だけの組み合わせは、アプリのメニューのショートカットと重なりやすいです。⌥か⌃を加えると安全です。",
        systemShortcutName: { id in
            switch id {
            case "64": return "Spotlight検索を表示"
            case "65": return "Finderの検索ウインドウを表示"
            case "60": return "前の入力ソースを選択"
            case "61": return "入力メニューの次のソースを選択"
            case "28", "29", "30", "31", "184": return "スクリーンショット"
            case "27": return "次のウインドウを操作対象にする"
            case "32", "34": return "Mission Control"
            case "33", "35": return "アプリケーションウインドウ"
            case "79", "80", "81", "82": return "操作スペースの移動"
            case "98": return "ヘルプメニューを表示"
            case "builtin:appSwitcher": return "アプリの切り替え"
            case "builtin:forceQuit": return "強制終了"
            case "builtin:lockScreen": return "画面をロック"
            default: return "キーボードショートカット（番号\(id)）"
            }
        },
        recorderDisable: "無効にする",
        recorderCancel: "キャンセル",
        layoutName: { layout in
            switch layout {
            case .full: return "標準"
            case .usage: return "使用率だけ"
            case .tab: return "端に収納（つまみだけ）"
            case .hidden: return "隠す（ショートカットかメニューバーで出す）"
            }
        },
        menuBarStyleName: { style in
            switch style {
            case .icon: return "アイコンのみ"
            case .usage: return "使用率"
            case .status: return "状態の全文"
            }
        },
        basicNoUsage: "使用率はまだ分かりません。「使用率をAPIから取得」を有効にするか、READMEの手順でstatusline.shを設定してください",
        menuUsageAPI: "使用率をAPIから取得…",
        usageAPIFailed: { "APIから使用率を取得できませんでした（\($0)）" },
        consentTitle: "使用率をAnthropicのサーバーから取得しますか？",
        consentWhat: "有効にすると、Claude Codeがこのマシンに保存したログイン情報（キーチェーンの「Claude Code-credentials」）を使い、5分ごとにAnthropicの使用率のエンドポイントへ問い合わせます。5時間枠、週枠、モデル別の週枠が、ターミナルを使っていないときも更新されます。",
        consentRisk: "このエンドポイントは公開されていないもので、Anthropicがサードパーティのアプリからの利用を認めているという根拠は見つかっていません。予告なく使えなくなることや、規約に反すると判断されることがありえます。その場合の影響はあなたのアカウントに及びます。",
        consentPrivacy: "ログイン情報はこの問い合わせにだけ使い、保存も記録もしません。ほかには何も送りません。初回は、キーチェーンの項目へのアクセスを許可するかをmacOSが尋ねることがあります。",
        consentWithout: "有効にしない場合は、Claude CodeのstatusLineから5時間枠と週枠だけを表示します。値が更新されるのは、ターミナルでClaude Codeを使っている間です。あとからメニューバーの項目で変えられます。",
        consentEnable: "有効にする",
        consentDecline: "今はしない",
        dragHandle: "ドラッグで移動",
        limitFiveHour: "5時間枠",
        limitWeekly: "週枠",
        limitLine: { name, percent, resets in
            // 日本語と数字の間には空白を入れない（"週枠75%"）。英字の名前なら空ける（"Fable 55%"）
            let sep = name.last?.isASCII == true ? " " : ""
            let head = "\(name)\(sep)\(percent)%"
            guard let resets else { return head }
            return "\(head)（\(resets)にリセット）"
        },
        usageAsOf: { "\($0)時点の値" },
        menuLaunchAtLogin: "ログイン時に起動",
        loginItemNeedsApproval: "システム設定の「ログイン項目」でSlashstripを許可してください",
        loginItemFailed: { "ログイン項目を変更できませんでした: \($0)" },
        resetLocale: "ja_JP",
        resetSameDay: "H:mm",
        resetOtherDay: "M/d(E) H:mm",
        menuLanguage: "言語",
        languageSystem: { "macOSに合わせる（\($0)）" },
        logShortcutCommitted: { "ショートカットを\($0)にしました（届くことを確認済み）" },
        logShortcutDisabled: "ショートカットを無効にしました",
        logShortcutFailed: { "ショートカット\($0)の登録に失敗しました" },
        logUsageAPIEnabled: "使用率APIからの取得を有効にしました",
        logUsageAPIDeclined: "使用率APIからの取得は有効にしませんでした",
        logUsageAPIDisabled: "使用率APIからの取得をやめました",
        logLoginItemOn: "ログイン時に起動するようにしました",
        logLoginItemOff: "ログイン時の起動をやめました",
        logOpened: "操作ログを開きました",
        logLanguageChanged: { "言語を\($0)にしました" }
    )

    public static func `for`(_ language: Language) -> Strings {
        language == .ja ? .ja : .en
    }
}

/// いま使う文言の表と、どの言語でも同じ文言
public enum L10n {
    public static var current: Strings = .for(.detect())

    /// メニューで選んだ言語（nilはmacOSに合わせる）に切り替える
    public static func apply(saved: String?) {
        current = .for(.detect(saved: saved))
    }

    public static let appName = "Slashstrip"
    public static let statusPlaceholder = "CC …"

    public static func opacityName(_ value: Double) -> String {
        "\(Int((value * 100).rounded()))%"
    }
}
