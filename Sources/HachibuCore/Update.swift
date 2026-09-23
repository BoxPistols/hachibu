import Foundation

/// 新しい版の確認。GitHubの最新リリース（/releases/latest）と、動いている版を比べる
public enum Update {
    /// 公開されている最新の版
    public struct Release: Equatable {
        /// "0.4.0"（タグの先頭のvは落とす）
        public let version: String
        public let url: URL

        public init(version: String, url: URL) {
            self.version = version
            self.url = url
        }
    }

    /// GitHubのAPIの応答を読む。下書きとプレリリースは知らせない
    public static func parseLatest(_ obj: [String: Any]) -> Release? {
        guard obj["draft"] as? Bool != true, obj["prerelease"] as? Bool != true,
              let tag = obj["tag_name"] as? String, let version = normalized(tag),
              let page = (obj["html_url"] as? String).flatMap(URL.init(string:)) else { return nil }
        return Release(version: version, url: page)
    }

    /// latestがcurrentより新しいか。どちらかが版の番号として読めなければfalse（開発中のビルドに知らせない）
    public static func isNewer(_ latest: String, than current: String) -> Bool {
        guard let a = components(latest), let b = components(current) else { return false }
        for i in 0..<max(a.count, b.count) {
            let x = i < a.count ? a[i] : 0
            let y = i < b.count ? b[i] : 0
            if x != y { return x > y }
        }
        return false
    }

    /// 知らせるべき新しい版。すでに知らせた版（notified）は、メニューには出すが通知はしない
    public static func shouldNotify(_ release: Release, current: String, notified: String?) -> Bool {
        isNewer(release.version, than: current) && (notified.map { isNewer(release.version, than: $0) } ?? true)
    }

    /// "v0.4.0" → "0.4.0"。数字とドットだけのものを版の番号とみなす
    static func normalized(_ tag: String) -> String? {
        var s = tag.trimmingCharacters(in: .whitespaces)
        if s.first == "v" || s.first == "V" { s.removeFirst() }
        return components(s) == nil ? nil : s
    }

    private static func components(_ s: String) -> [Int]? {
        let parts = s.split(separator: ".", omittingEmptySubsequences: false)
        guard !parts.isEmpty else { return nil }
        var out: [Int] = []
        for p in parts {
            guard !p.isEmpty, p.allSatisfy({ $0.isASCII && $0.isNumber }), let n = Int(p) else { return nil }
            out.append(n)
        }
        return out
    }
}
