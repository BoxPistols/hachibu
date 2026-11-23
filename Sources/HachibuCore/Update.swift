import Foundation

/// 新しいバージョンの確認。GitHubのリリースの一覧と、動いているバージョンを比べる
public enum Update {
    /// 公開されている最新のバージョン
    public struct Release: Equatable {
        /// "0.4.0"（タグの先頭のvは落とす）
        public let version: String
        public let url: URL
        /// 添付のHachibu-macos.zip。無ければ、その場での置き換えはせずにページを開く
        public let assetURL: URL?
        /// 本文の末尾にリリースの手順が書くzipのSHA-256（小文字の16進）。無ければ突き合わせない
        public let sha256: String?

        public init(version: String, url: URL, assetURL: URL? = nil, sha256: String? = nil) {
            self.version = version
            self.url = url
            self.assetURL = assetURL
            self.sha256 = sha256
        }
    }

    /// リリースに添付するmacOS向けのzipの名前（scripts/build-app.sh --zip）
    public static let assetName = "Hachibu-macos.zip"

    /// 確認の結果。失敗を「最新」と同じに扱うと、通信が一度落ちただけで出ていた知らせが消える
    public enum Outcome: Equatable {
        case newer(Release)
        case upToDate
        case failed
    }

    /// /releasesの一覧の応答から結果を決める。
    /// 同じリポジトリにHachibu Lite（lite-v）とWindows向け（windows-v）のリリースもあるので、
    /// /releases/latestには頼らず、vで始まるタグだけから最も新しいものを選ぶ
    public static func outcome(status: Int?, data: Data?, current: String) -> Outcome {
        guard status == 200, let data,
              let list = (try? JSONSerialization.jsonObject(with: data)) as? [[String: Any]] else { return .failed }
        let newest = list.compactMap(parseRelease).max { isNewer($1.version, than: $0.version) }
        guard let newest, isNewer(newest.version, than: current) else { return .upToDate }
        return .newer(newest)
    }

    /// 1件のリリースを読む。下書き、プレリリース、vで始まらないタグ（lite-v、windows-v）は対象にしない
    public static func parseRelease(_ obj: [String: Any]) -> Release? {
        guard let base = parseLatest(obj) else { return nil }
        let assets = obj["assets"] as? [[String: Any]] ?? []
        let asset = assets.first { $0["name"] as? String == assetName }
            .flatMap { $0["browser_download_url"] as? String }.flatMap(URL.init(string:))
        return Release(version: base.version, url: base.url, assetURL: asset,
                       sha256: (obj["body"] as? String).flatMap(sha256(inBody:)))
    }

    /// 本文の「SHA-256 of Hachibu-macos.zip: `…`」から値を取る
    static func sha256(inBody body: String) -> String? {
        let marker = "SHA-256 of \(assetName):"
        guard let r = body.range(of: marker) else { return nil }
        let hex = body[r.upperBound...].drop { $0 == " " || $0 == "`" }.prefix { $0.isHexDigit }
        return hex.count == 64 ? hex.lowercased() : nil
    }

    /// GitHubのAPIの応答を読む。下書きとプレリリースは知らせない
    public static func parseLatest(_ obj: [String: Any]) -> Release? {
        guard obj["draft"] as? Bool != true, obj["prerelease"] as? Bool != true,
              let tag = obj["tag_name"] as? String, let version = normalized(tag),
              let page = (obj["html_url"] as? String).flatMap(URL.init(string:)) else { return nil }
        return Release(version: version, url: page)
    }

    /// latestがcurrentより新しいか。どちらかがバージョン番号として読めなければfalse（開発中のビルドに知らせない）
    public static func isNewer(_ latest: String, than current: String) -> Bool {
        guard let a = components(latest), let b = components(current) else { return false }
        for i in 0..<max(a.count, b.count) {
            let x = i < a.count ? a[i] : 0
            let y = i < b.count ? b[i] : 0
            if x != y { return x > y }
        }
        return false
    }

    /// 知らせるべき新しいバージョン。すでに知らせたバージョン（notified）は、メニューには出すが通知はしない
    public static func shouldNotify(_ release: Release, current: String, notified: String?) -> Bool {
        isNewer(release.version, than: current) && (notified.map { isNewer(release.version, than: $0) } ?? true)
    }

    /// "v0.4.0" → "0.4.0"。数字とドットだけのものをバージョン番号とみなす
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
