import Foundation

/// ステータス文字列の組み立て。
public enum Usage {
    /// Touch Barのステータスと同じ形: "Opus5 1M xhigh · S9 W75 F55"
    /// - model: 表示用に縮めたモデル名（compactDisplayNameの結果）
    public static func statusText(model: String?, effort: String?, limits: [UsageLimit]) -> String {
        let head = [model, effort].compactMap { $0 }.filter { !$0.isEmpty }
        let tail = limits.map { limit -> String in
            switch limit.name {
            case L10n.limitFiveHour: return "S\(limit.percent)"
            case L10n.limitWeekly: return "W\(limit.percent)"
            default: return "\(limit.name.prefix(1))\(limit.percent)"
            }
        }
        let text = [head.joined(separator: " "), tail.joined(separator: " ")]
            .filter { !$0.isEmpty }
            .joined(separator: " · ")
        return text.isEmpty ? L10n.statusPlaceholder : text
    }

    /// statusLineのmodel.display_nameを帯の幅に合わせて縮める。"Opus 5 (1M context)" → "Opus5 1M"
    public static func compactDisplayName(_ name: String) -> String {
        var s = name.trimmingCharacters(in: .whitespaces)
        var suffix = ""
        if let r = s.range(of: #"\s*\(\s*\d+(\.\d+)?\s*[KkMm]\s+context\s*\)"#, options: .regularExpression) {
            let size = s[r].filter { $0.isNumber || $0 == "." || "KkMm".contains($0) }
            suffix = " " + size.uppercased()
            s.removeSubrange(r)
        }
        // 名前と版番号の間の空白を詰める（"Opus 5" → "Opus5"）
        s = s.replacingOccurrences(of: #"([A-Za-z]) (\d)"#, with: "$1$2", options: .regularExpression)
        return s + suffix
    }
}

/// Claude CodeのstatusLineが標準入力に渡すJSON（公式ドキュメントの拡張点）の解釈。
public enum StatusLine {
    public struct Snapshot: Equatable {
        public let model: String?
        public let effort: String?
        public let limits: [UsageLimit]

        public init(model: String?, effort: String?, limits: [UsageLimit]) {
            self.model = model
            self.effort = effort
            self.limits = limits
        }
    }

    /// rate_limitsはPro/Maxで、そのセッションの最初の応答の後にだけ入る。無ければ使用率は空になる
    public static func parse(_ obj: [String: Any]) -> Snapshot {
        let model = ((obj["model"] as? [String: Any])?["display_name"] as? String).map(Usage.compactDisplayName)
        let effort = (obj["effort"] as? [String: Any])?["level"] as? String
        var limits: [UsageLimit] = []
        let rates = obj["rate_limits"] as? [String: Any]
        for (key, name) in [("five_hour", L10n.limitFiveHour), ("seven_day", L10n.limitWeekly)] {
            guard let item = rates?[key] as? [String: Any],
                  let pct = (item["used_percentage"] as? NSNumber)?.doubleValue else { continue }
            let resets = (item["resets_at"] as? NSNumber).map { Date(timeIntervalSince1970: $0.doubleValue) }
            limits.append(UsageLimit(name: name, percent: Int(pct.rounded()), resetsAt: resets))
        }
        return Snapshot(model: model, effort: effort, limits: limits)
    }
}
