import Foundation

/// 「表示の読み方」に並べる行。帯にいま出ている記号と色だけを説明する
public enum Legend {
    public enum Mark: Equatable {
        case contextDot
        case level(UsageLevel)
        case none
    }

    public struct Line: Equatable {
        public let mark: Mark
        public let text: String
    }

    public static func lines(statusText: String?, limits: [UsageLimit], thresholds: UsageThresholds,
                             strings s: Strings = L10n.current) -> [Line] {
        var lines: [Line] = []
        if let statusText, let split = ContextMark.split(statusText) {
            lines.append(Line(mark: .contextDot, text: s.legendContext(split.context)))
        }
        for limit in limits {
            switch limit.kind {
            case .model(let name): lines.append(Line(mark: .none, text: s.legendModelLimit(limit.letter, name)))
            default: lines.append(Line(mark: .none, text: s.legendLimit(limit.letter, limit.name(s))))
            }
        }
        lines.append(Line(mark: .level(.warning), text: s.legendWarning(thresholds.warning)))
        lines.append(Line(mark: .level(.critical), text: s.legendCritical(thresholds.critical)))
        return lines
    }
}
