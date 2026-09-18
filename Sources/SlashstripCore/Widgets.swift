import Foundation

/// ~/.claude/btt/render/<name>.jsonの解釈。ファイルの読み書きはアプリ側で行い、ここは判断だけを持つ。
public enum Widgets {
    public static let status = "status"
    private static let permKinds = ["allow", "always", "reject"]

    /// Touch Barと同じ並び: 状態 → 許可応答 → 番号 → コマンド。番号は数値順（"cmd-10" を "cmd-2" の前に置かない）
    public static func ordered(_ names: Set<String>) -> [String] {
        func numbered(_ prefix: String) -> [String] {
            names.compactMap { name -> (Int, String)? in
                guard name.hasPrefix(prefix), let n = Int(name.dropFirst(prefix.count)), n >= 0 else { return nil }
                return (n, name)
            }
            .sorted { $0.0 < $1.0 }
            .map(\.1)
        }
        let perms = permKinds.map { "perm-" + $0 }.filter(names.contains)
        return [status] + perms + numbered("menu-") + numbered("cmd-")
    }

    /// 1ファイルぶんのJSONをボタンにする。隠す指定か空文字ならnil
    public static func slot(name: String, json: [String: Any]) -> Slot? {
        let text = (json["text"] as? String) ?? ""
        if (json["hidden"] as? Bool) == true || text.isEmpty {
            return nil
        }
        return Slot(id: name, text: text,
                    background: RGBA(csv: json["background_color"] as? String) ?? .commandBackground,
                    foreground: RGBA(csv: json["font_color"] as? String) ?? .white)
    }

    /// デーモン起動直後でstatus.jsonがまだ無いときの仮表示（cc-widget.shと同じ）
    public static func placeholderStatus() -> Slot {
        Slot(id: status, text: L10n.statusPlaceholder, background: .clear, foreground: .idleForeground)
    }

    /// 押したときにBTTのウィジェットと同じスクリプトを同じ引数で呼ぶ。知らない名前には何も返さない
    public static func arguments(for id: String, base: URL) -> [String]? {
        func script(_ name: String) -> String { base.appendingPathComponent(name).path }
        if id == status {
            return [script("cc-focus.py")]
        }
        if id.hasPrefix("perm-") {
            let kind = String(id.dropFirst("perm-".count))
            return permKinds.contains(kind) ? [script("cc-send.py"), kind] : nil
        }
        if id.hasPrefix("menu-"), let n = Int(id.dropFirst("menu-".count)), n > 0 {
            return [script("cc-menu.py"), "send", String(n)]
        }
        if id.hasPrefix("cmd-"), let i = Int(id.dropFirst("cmd-".count)), i >= 0 {
            return [script("cc-menu.py"), "run", String(i)]
        }
        return nil
    }

    /// usage.json（statusLineのrate_limits。値は0〜100の整数）とusage-api.jsonのscoped（モデル別枠）を合わせる
    public static func limits(usage: [String: Any]?, usageAPI: [String: Any]?) -> [UsageLimit] {
        var out: [UsageLimit] = []
        if let u = usage {
            if let p = (u["session"] as? NSNumber)?.intValue {
                out.append(UsageLimit(kind: .fiveHour, percent: p, resetsAt: epoch(u["session_resets_at"])))
            }
            if let p = (u["week"] as? NSNumber)?.intValue {
                out.append(UsageLimit(kind: .weekly, percent: p, resetsAt: epoch(u["week_resets_at"])))
            }
        }
        for s in (usageAPI?["scoped"] as? [[String: Any]]) ?? [] {
            guard let name = s["name"] as? String, !name.isEmpty,
                  let p = (s["pct"] as? NSNumber)?.intValue else { continue }
            out.append(UsageLimit(kind: .model(name), percent: p, resetsAt: ResetFormatter.parseISO(s["resets_at"] as? String)))
        }
        return out
    }

    private static func epoch(_ any: Any?) -> Date? {
        guard let v = (any as? NSNumber)?.doubleValue, v > 0 else { return nil }
        return Date(timeIntervalSince1970: v)
    }
}
