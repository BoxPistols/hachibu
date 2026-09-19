import Foundation

/// 状態の文字列からコンテキスト長（"1M"など）を取り出す。
/// 帯とメニューバーでは幅を取らない小さな印にし、メニューとツールチップでは文字で書く
public enum ContextMark {
    public struct Split: Equatable {
        /// モデル名（"Opus5"）
        public let model: String
        /// コンテキスト長（"1M"）
        public let context: String
        /// 残り（" xhigh · S2 W81"。先頭の空白を含む）
        public let rest: String
    }

    /// "Opus5 1M xhigh · S2" → ("Opus5", "1M", " xhigh · S2")。コンテキスト長が無ければnil
    public static func split(_ text: String) -> Split? {
        guard let match = text.firstMatch(of: #/^(\S+) (\d+[MK])(?= |$)/#) else { return nil }
        return Split(model: String(match.1), context: String(match.2), rest: String(text[match.range.upperBound...]))
    }

    /// メニューの先頭に出す、省略しない1行。"Opus5 · 1M context · xhigh"
    public static func spelledOut(_ statusText: String) -> String? {
        let head = statusText.components(separatedBy: " · ").first ?? statusText
        guard let s = split(head) else { return nil }
        let effort = s.rest.trimmingCharacters(in: .whitespaces)
        return ([s.model, "\(s.context) context"] + (effort.isEmpty ? [] : [effort])).joined(separator: " · ")
    }
}
