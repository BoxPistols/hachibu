import Foundation

/// 使用率の段階。帯では数字を札で囲み、黄＝注意、赤＝危険として示す
public enum UsageLevel: Int, Comparable {
    case normal
    case warning
    case critical

    public static func < (a: UsageLevel, b: UsageLevel) -> Bool { a.rawValue < b.rawValue }
}

/// 黄と赤に切り替える使用率（%）。利用者が選べる。赤は黄より大きい値にしか置けない
public struct UsageThresholds: Equatable {
    public let warning: Int
    public let critical: Int

    public static let `default` = UsageThresholds(warning: 70, critical: 90)!
    public static let warningChoices = [50, 60, 70, 80]
    public static let criticalChoices = [80, 85, 90, 95]

    /// 赤が黄以下になる組み合わせは作らない
    public init?(warning: Int, critical: Int) {
        guard (1...100).contains(warning), (1...100).contains(critical), warning < critical else { return nil }
        self.warning = warning
        self.critical = critical
    }

    public func level(_ percent: Int) -> UsageLevel {
        if percent >= critical { return .critical }
        if percent >= warning { return .warning }
        return .normal
    }

    public func worst(_ limits: [UsageLimit]) -> UsageLevel {
        limits.map { level($0.percent) }.max() ?? .normal
    }
}

/// 帯の文字の一部を札で囲むための分割
public struct TextSegment: Equatable {
    public let text: String
    public let level: UsageLevel

    public init(text: String, level: UsageLevel) {
        self.text = text
        self.level = level
    }
}

public enum UsageMarkup {
    /// "S12" "W76" "F55" のような使用率の語だけに段階を付ける。それ以外（モデル名、ctx91%など）は普通の文字のまま
    public static func segments(_ text: String, thresholds: UsageThresholds) -> [TextSegment] {
        var out: [TextSegment] = []
        var plain = ""
        let tokens = text.split(separator: " ", omittingEmptySubsequences: false)
        for (i, token) in tokens.enumerated() {
            let sep = i == 0 ? "" : " "
            if let percent = usagePercent(token), thresholds.level(percent) != .normal {
                plain += sep
                if !plain.isEmpty { out.append(TextSegment(text: plain, level: .normal)) }
                plain = ""
                out.append(TextSegment(text: String(token), level: thresholds.level(percent)))
            } else {
                plain += sep + token
            }
        }
        if !plain.isEmpty { out.append(TextSegment(text: plain, level: .normal)) }
        return out
    }

    /// 大文字1字＋数字1〜3桁（S12、W100）
    static func usagePercent(_ token: Substring) -> Int? {
        guard token.count >= 2, token.count <= 4, let first = token.first, first.isUppercase, first.isASCII else { return nil }
        let digits = token.dropFirst()
        guard digits.allSatisfy({ $0.isASCII && $0.isNumber }) else { return nil }
        return Int(digits)
    }
}
