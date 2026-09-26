import Foundation

/// 基本表示のモデル名とeffort。statusLineが無くても出せるよう、会話記録と設定から組み立てる
public enum ModelInfo {
    /// モデルのID（"claude-opus-5"、"claude-opus-5-20260101"、"claude-opus-5[1m]"）を帯の形にする: "Opus5"、"Opus5 1M"
    public static func compactID(_ id: String) -> String? {
        var s = id.trimmingCharacters(in: .whitespaces)
        guard s.hasPrefix("claude-") else { return nil }
        s.removeFirst("claude-".count)
        var suffix = ""
        if let r = s.range(of: #"\[(\d+)m\]$"#, options: .regularExpression) {
            suffix = " " + s[r].dropFirst().dropLast().uppercased()
            s.removeSubrange(r)
        }
        // 末尾の日付（-20260101）は落とす
        s = s.replacingOccurrences(of: #"-\d{8}$"#, with: "", options: .regularExpression)
        let parts = s.split(separator: "-")
        guard let family = parts.first, !family.isEmpty else { return nil }
        // "opus-5" → "Opus5"、"haiku-4-5" → "Haiku4.5"
        let version = parts.dropFirst().joined(separator: ".")
        return family.prefix(1).uppercased() + family.dropFirst() + version + suffix
    }

    /// 設定のモデル指定（"opus[1m]"、"sonnet"）から、文脈の長さの指定だけを取り出す: "1M"
    public static func contextSuffix(ofAlias alias: String) -> String? {
        guard let r = alias.range(of: #"\[(\d+)m\]$"#, options: .regularExpression) else { return nil }
        return alias[r].dropFirst().dropLast().uppercased()
    }

    /// 会話記録（JSONL）の末尾から、最後に使われたモデルのIDを探す
    public static func lastModelID(inTranscriptTail text: String) -> String? {
        let pattern = #""model"\s*:\s*"(claude-[A-Za-z0-9.\-\[\]]+)""#
        guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
        let ns = text as NSString
        guard let match = regex.matches(in: text, range: NSRange(location: 0, length: ns.length)).last else { return nil }
        return ns.substring(with: match.range(at: 1))
    }

    /// 会話記録（JSONL）の末尾から、最後の応答のeffortを探す。
    /// セッション中に/effortで変えた値は~/.claude/settings.jsonに残らないので、こちらを設定より優先する
    public static func lastEffort(inTranscriptTail text: String) -> String? {
        let pattern = #""effort"\s*:\s*"(low|medium|high|xhigh|max)""#
        guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
        let ns = text as NSString
        guard let match = regex.matches(in: text, range: NSRange(location: 0, length: ns.length)).last else { return nil }
        return ns.substring(with: match.range(at: 1))
    }

    /// 会話記録のモデルIDと設定のモデル指定を合わせて、帯のモデル名にする。
    /// 会話記録のIDに文脈の長さが無ければ、同じ系統のモデルを指す設定の指定（"opus[1m]"の"1M"）を添える
    public static func displayName(transcriptModelID: String?, settingsModel: String?) -> String? {
        if let id = transcriptModelID, var name = compactID(id) {
            // 設定の指定が別の系統のモデル（会話はsonnet、設定は"opus[1m]"など）なら、その文脈の長さは付けない
            if !name.contains(" "), let settingsModel, let ctx = contextSuffix(ofAlias: settingsModel),
               let family = settingsModel.lowercased().split(whereSeparator: { !$0.isLetter }).first,
               id.lowercased().contains(family) {
                name += " " + ctx
            }
            return name
        }
        guard let settingsModel, !settingsModel.isEmpty else { return nil }
        return Usage.compactDisplayName(aliasToDisplay(settingsModel))
    }

    /// "opus[1m]" → "Opus (1M context)"（statusLineのdisplay_nameと同じ形に寄せて、同じ関数で縮める）
    static func aliasToDisplay(_ alias: String) -> String {
        var s = alias
        var ctx = ""
        if let c = contextSuffix(ofAlias: alias), let r = s.range(of: #"\[(\d+)m\]$"#, options: .regularExpression) {
            ctx = " (\(c) context)"
            s.removeSubrange(r)
        }
        return s.prefix(1).uppercased() + String(s.dropFirst()) + ctx
    }
}
