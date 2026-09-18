import Foundation

// 画面に出る文言はここに集める（i18nの対象）
public enum L10n {
    public static let appName = "Slashstrip"

    public static let menuLayout = "表示"
    public static let menuOpacity = "不透明度"
    public static let menuMenuBar = "メニューバー"
    public static let menuThresholds = "色の閾値"
    public static let thresholdWarningHeader = "黄にする使用率"
    public static let thresholdCriticalHeader = "赤にする使用率"
    public static func thresholdChoice(_ percent: Int) -> String {
        "\(percent)%以上（残り\(100 - percent)%以下）"
    }
    public static let menuSummon = "帯を呼び出す"
    public static let menuChangeShortcut = "ショートカットを変更…"
    public static let menuShortcutOff = "ショートカットは無効です"
    public static let menuResetPosition = "位置を初期状態に戻す"
    public static let menuOpenLog = "操作ログを開く"
    public static let menuQuit = "Slashstripを終了"
    public static let menuSourcePrefix = "データ元："
    public static let menuNoUsage = "使用率はまだ取得できていません"
    public static func menuHotKeyUnavailable(_ display: String) -> String {
        "ショートカット\(display)は他のアプリが使っています"
    }

    public static let sessionsNone = "Claude Codeのセッションはありません"
    public static let sessionsHeader = "移動先のセッション（開始順）"

    public static func sessionStateName(_ state: SessionInfo.State) -> String {
        switch state {
        case .busy: return "実行中"
        case .waitingPermission: return "許可待ち"
        case .waitingInput: return "入力待ち"
        case .idle: return "待機中"
        }
    }

    public static let recorderTitle = "帯を呼び出すショートカット"
    public static let recorderPrompt = "新しい組み合わせを押してください。⌘・⌥・⌃のどれかを含めます。"
    public static func recorderCurrent(_ display: String?) -> String {
        "現在：" + (display ?? "なし")
    }
    public static let recorderRejected = "⌘・⌥・⌃のどれかと一緒に押してください"
    public static func recorderTaken(_ display: String) -> String {
        "\(display)は他のアプリが使っているため登録できません"
    }
    public static let recorderDisable = "無効にする"
    public static let recorderCancel = "キャンセル"

    public static func layoutName(_ layout: StripLayout) -> String {
        switch layout {
        case .full: return "フル"
        case .compact: return "コンパクト（状態の1枠だけ）"
        case .usage: return "使用率だけ"
        case .tab: return "端に収納（つまみだけ）"
        case .hidden: return "隠す（答えが要るときだけ出す）"
        }
    }

    public static func menuBarStyleName(_ style: MenuBarStyle) -> String {
        switch style {
        case .icon: return "アイコンのみ"
        case .usage: return "使用率"
        case .status: return "状態の全文"
        }
    }

    public static func opacityName(_ value: Double) -> String {
        "\(Int((value * 100).rounded()))%"
    }

    public static let sourceScripts = "Touch Bar連携スクリプト（~/.claude/btt）"
    public static let sourceStatusLine = "statusLine（コマンド送信なし）"
    public static let statusLineMissing = "statusLineの出力がまだありません。READMEの手順でscripts/statusline.shを設定してください"

    public static let dragHandle = "ドラッグで移動"
    public static let statusPlaceholder = "CC …"
    public static let staleNotice = "描画デーモンの出力が止まっています"

    public static let limitFiveHour = "5時間枠"
    public static let limitWeekly = "週枠"

    public static func limitLine(name: String, percent: Int, resets: String?) -> String {
        // 日本語と数字の間には空白を入れない（"週枠75%"）。英字の名前なら空ける（"Fable 55%"）
        let sep = name.last?.isASCII == true ? " " : ""
        let head = "\(name)\(sep)\(percent)%"
        guard let resets else { return head }
        return "\(head)（\(resets)にリセット）"
    }
}
