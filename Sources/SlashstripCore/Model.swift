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
public struct UsageLimit: Equatable {
    public let name: String
    public let percent: Int
    public let resetsAt: Date?

    public init(name: String, percent: Int, resetsAt: Date?) {
        self.name = name
        self.percent = percent
        self.resetsAt = resetsAt
    }

    public var line: String {
        L10n.limitLine(name: name, percent: percent, resets: resetsAt.map { ResetFormatter.text($0) })
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
    private static func make(_ format: String) -> DateFormatter {
        let f = DateFormatter()
        f.locale = Locale(identifier: "ja_JP")
        f.dateFormat = format
        return f
    }

    private static let sameDay = make("H:mm")
    private static let otherDay = make("M/d(E) H:mm")

    public static func text(_ date: Date, now: Date = Date(), calendar: Calendar = .current) -> String {
        let f = calendar.isDate(date, inSameDayAs: now) ? sameDay : otherDay
        f.timeZone = calendar.timeZone
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
