import Foundation

/// 画面の言語。macOSの優先言語が日本語なら日本語、それ以外は英語。SLASHSTRIP_LANG=en|jaで上書きできる
public enum Language: String {
    case en
    case ja

    public static func detect(preferred: [String] = Locale.preferredLanguages,
                              override: String? = ProcessInfo.processInfo.environment["SLASHSTRIP_LANG"]) -> Language {
        if let override, let lang = Language(rawValue: override) { return lang }
        return preferred.first?.hasPrefix("ja") == true ? .ja : .en
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
    public let menuSourcePrefix: String
    public let menuNoUsage: String
    public let menuHotKeyUnavailable: (String) -> String

    public let sessionsNone: String
    public let sessionsHeader: String
    public let sessionStateName: (SessionInfo.State) -> String
    /// 番号・プロジェクト名・詳細（状態とツール）から一覧の1行を作る
    public let sessionTitle: (Int, String, [String]) -> String

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

    public let sourceScripts: String
    public let sourceStatusLine: String
    public let statusLineMissing: String
    public let dragHandle: String
    public let staleNotice: String

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
        menuSourcePrefix: "Source: ",
        menuNoUsage: "Usage is not available yet",
        menuHotKeyUnavailable: { "\($0) is already used by another app" },
        sessionsNone: "No Claude Code sessions",
        sessionsHeader: "Sessions (in start order)",
        sessionStateName: { state in
            switch state {
            case .busy: return "running"
            case .waitingPermission: return "waiting for permission"
            case .waitingInput: return "waiting for input"
            case .idle: return "idle"
            }
        },
        sessionTitle: { number, project, detail in "\(number). \(project) (\(detail.joined(separator: ", ")))" },
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
            case .full: return "Full"
            case .compact: return "Compact (status only)"
            case .usage: return "Usage only"
            case .tab: return "Tucked (handle only)"
            case .hidden: return "Hidden (appears when an answer is needed)"
            }
        },
        menuBarStyleName: { style in
            switch style {
            case .icon: return "Icon only"
            case .usage: return "Usage"
            case .status: return "Full status"
            }
        },
        sourceScripts: "Touch Bar bridge scripts (~/.claude/btt)",
        sourceStatusLine: "statusLine (no command buttons)",
        statusLineMissing: "No statusLine output yet. Set up scripts/statusline.sh as described in the README.",
        dragHandle: "Drag to move",
        staleNotice: "The render daemon has stopped updating",
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
        resetOtherDay: "EEE M/d H:mm"
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
        menuSourcePrefix: "データ元：",
        menuNoUsage: "使用率はまだ取得できていません",
        menuHotKeyUnavailable: { "ショートカット\($0)は他のアプリが使っています" },
        sessionsNone: "Claude Codeのセッションはありません",
        sessionsHeader: "移動先のセッション（開始順）",
        sessionStateName: { state in
            switch state {
            case .busy: return "実行中"
            case .waitingPermission: return "許可待ち"
            case .waitingInput: return "入力待ち"
            case .idle: return "待機中"
            }
        },
        sessionTitle: { number, project, detail in "\(number). \(project)（\(detail.joined(separator: "・"))）" },
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
            case .full: return "フル"
            case .compact: return "コンパクト（状態の1枠だけ）"
            case .usage: return "使用率だけ"
            case .tab: return "端に収納（つまみだけ）"
            case .hidden: return "隠す（答えが要るときだけ出す）"
            }
        },
        menuBarStyleName: { style in
            switch style {
            case .icon: return "アイコンのみ"
            case .usage: return "使用率"
            case .status: return "状態の全文"
            }
        },
        sourceScripts: "Touch Bar連携スクリプト（~/.claude/btt）",
        sourceStatusLine: "statusLine（コマンド送信なし）",
        statusLineMissing: "statusLineの出力がまだありません。READMEの手順でscripts/statusline.shを設定してください",
        dragHandle: "ドラッグで移動",
        staleNotice: "描画デーモンの出力が止まっています",
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
        resetOtherDay: "M/d(E) H:mm"
    )

    public static func `for`(_ language: Language) -> Strings {
        language == .ja ? .ja : .en
    }
}

/// いま使う文言の表と、どの言語でも同じ文言
public enum L10n {
    public static var current: Strings = .for(.detect())

    public static let appName = "Slashstrip"
    public static let statusPlaceholder = "CC …"

    public static func opacityName(_ value: Double) -> String {
        "\(Int((value * 100).rounded()))%"
    }
}
