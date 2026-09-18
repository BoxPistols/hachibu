import Foundation

/// Touch Bar連携スクリプトと同じ "R,G,B,A"（各0〜255）形式の色。
public struct RGBA: Equatable {
    public var r: Double
    public var g: Double
    public var b: Double
    public var a: Double

    public static let idleBackground = RGBA(r: 58, g: 58, b: 60, a: 255)
    public static let idleForeground = RGBA(r: 190, g: 190, b: 195, a: 255)
    public static let commandBackground = RGBA(r: 72, g: 72, b: 72, a: 255)
    public static let white = RGBA(r: 255, g: 255, b: 255, a: 255)
    public static let clear = RGBA(r: 0, g: 0, b: 0, a: 0)

    public init(r: Double, g: Double, b: Double, a: Double) {
        self.r = r
        self.g = g
        self.b = b
        self.a = a
    }

    public init?(csv: String?) {
        guard let csv else { return nil }
        let parts = csv.split(separator: ",").map { Double($0.trimmingCharacters(in: .whitespaces)) }
        guard parts.count == 4, !parts.contains(nil) else { return nil }
        let v = parts.compactMap { $0 }
        guard v.allSatisfy({ (0...255).contains($0) }) else { return nil }
        self.init(r: v[0], g: v[1], b: v[2], a: v[3])
    }
}

/// ストリップ上のボタン1個ぶん。idはTouch Barのウィジェット名（status / cmd-0 / perm-allow …）と同じ。
public struct Slot: Identifiable, Equatable {
    public let id: String
    public var text: String
    public var background: RGBA
    public var foreground: RGBA
    public var help: String?
    public var dimmed: Bool

    public init(id: String, text: String, background: RGBA, foreground: RGBA,
                help: String? = nil, dimmed: Bool = false) {
        self.id = id
        self.text = text
        self.background = background
        self.foreground = foreground
        self.help = help
        self.dimmed = dimmed
    }
}

/// 使用率の枠1つ（5時間枠・週枠・モデル別枠）。
/// 枠の種類は表示名ではなくkindで見分ける（表示名は言語で変わる）
public struct UsageLimit: Equatable {
    public enum Kind: Equatable {
        case fiveHour
        case weekly
        /// モデル別の週枠。値はサーバが返すモデルの表示名
        case model(String)
    }

    public let kind: Kind
    public let percent: Int
    public let resetsAt: Date?

    public init(kind: Kind, percent: Int, resetsAt: Date?) {
        self.kind = kind
        self.percent = percent
        self.resetsAt = resetsAt
    }

    public func name(_ strings: Strings = L10n.current) -> String {
        switch kind {
        case .fiveHour: return strings.limitFiveHour
        case .weekly: return strings.limitWeekly
        case .model(let name): return name
        }
    }

    /// 帯での略号（Touch Barと同じ）: S=5時間枠、W=週枠、モデル別は名前の頭文字
    public var letter: String {
        switch kind {
        case .fiveHour: return "S"
        case .weekly: return "W"
        case .model(let name): return String(name.prefix(1))
        }
    }

    public func line(_ strings: Strings = L10n.current, now: Date = Date(), calendar: Calendar = .current) -> String {
        strings.limitLine(name(strings), percent,
                          resetsAt.map { ResetFormatter.text($0, now: now, calendar: calendar, strings: strings) })
    }
}

/// 表示と操作の供給元。Touch Bar連携スクリプトがあればそれを使い、無ければ使用率APIだけで動く。
public protocol DataSource: AnyObject {
    var onSlots: (([Slot]) -> Void)? { get set }
    var sourceDescription: String { get }
    var limits: [UsageLimit] { get }
    func start()
    func perform(_ slot: Slot)
    /// 生きているClaude Codeのセッション（一覧の並び順に並べ済み）と、いま前面にあるもののid
    func sessions() -> (list: [SessionInfo], focusedID: String?)
    func focus(_ session: SessionInfo)
}

public extension DataSource {
    func sessions() -> (list: [SessionInfo], focusedID: String?) { ([], nil) }
    func focus(_ session: SessionInfo) {}
}

public enum ResetFormatter {
    public static func text(_ date: Date, now: Date = Date(), calendar: Calendar = .current,
                            strings: Strings = L10n.current) -> String {
        let f = DateFormatter()
        f.locale = Locale(identifier: strings.resetLocale)
        f.timeZone = calendar.timeZone
        f.dateFormat = calendar.isDate(date, inSameDayAs: now) ? strings.resetSameDay : strings.resetOtherDay
        return f.string(from: date)
    }

    /// サーバが返す "2026-09-19T23:59:59.865631+00:00" 形式。小数秒の桁数がまちまちなので落としてから読む。
    public static func parseISO(_ s: String?) -> Date? {
        guard let s else { return nil }
        let trimmed = s.replacingOccurrences(of: #"\.\d+"#, with: "", options: .regularExpression)
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f.date(from: trimmed)
    }
}
