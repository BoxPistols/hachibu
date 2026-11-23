import Foundation
import Testing
@testable import HachibuCore

@Suite struct UpdateTests {
    static let latest: [String: Any] = [
        "tag_name": "v0.4.0",
        "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/v0.4.0",
        "draft": false,
        "prerelease": false,
    ]

    @Test func readsTheLatestRelease() {
        let r = Update.parseLatest(Self.latest)
        #expect(r?.version == "0.4.0")
        #expect(r?.url.absoluteString == "https://github.com/BoxPistols/hachibu/releases/tag/v0.4.0")
    }

    @Test func ignoresDraftsPrereleasesAndOddTags() {
        var p = Self.latest
        p["prerelease"] = true
        #expect(Update.parseLatest(p) == nil)
        p = Self.latest
        p["draft"] = true
        #expect(Update.parseLatest(p) == nil)
        p = Self.latest
        p["tag_name"] = "nightly"
        #expect(Update.parseLatest(p) == nil)
        #expect(Update.parseLatest([:]) == nil)
    }

    @Test func comparesVersionsNumerically() {
        #expect(Update.isNewer("0.4.0", than: "0.3.2"))
        #expect(Update.isNewer("0.10.0", than: "0.9.9"))
        #expect(Update.isNewer("1.0", than: "0.99.1"))
        #expect(Update.isNewer("0.3.2.1", than: "0.3.2"))
        #expect(!Update.isNewer("0.3.2", than: "0.3.2"))
        #expect(!Update.isNewer("0.3.2", than: "0.3.2.0"))
        #expect(!Update.isNewer("0.3.1", than: "0.3.2"))
        // 開発中のビルド（"dev"）や読めない版には知らせない
        #expect(!Update.isNewer("0.4.0", than: "dev"))
        #expect(!Update.isNewer("0.4.0-beta", than: "0.3.2"))
        #expect(!Update.isNewer("+1.0", than: "0.3.2"))
    }

    static func release(_ tag: String, asset: Bool = true, body: String = "") -> [String: Any] {
        var r: [String: Any] = [
            "tag_name": tag,
            "html_url": "https://example.com/releases/tag/\(tag)",
            "draft": false,
            "prerelease": false,
            "body": body,
        ]
        if asset {
            r["assets"] = [
                ["name": "statusline.sh", "browser_download_url": "https://example.com/\(tag)/statusline.sh"],
                ["name": "Hachibu-macos.zip", "browser_download_url": "https://example.com/\(tag)/Hachibu-macos.zip"],
            ]
        }
        return r
    }

    static func json(_ list: [[String: Any]]) -> Data {
        try! JSONSerialization.data(withJSONObject: list)
    }

    @Test func picksTheNewestMacReleaseAmongOtherApps() {
        // Windows向けやLiteのリリースのほうが新しくても、vのタグだけから選ぶ
        let list = [Self.release("windows-v0.9.0"), Self.release("lite-v0.8.0"), Self.release("v0.5.0"), Self.release("v0.10.0")]
        guard case .newer(let r) = Update.outcome(status: 200, data: Self.json(list), current: "0.5.0") else {
            Issue.record("expected a newer release")
            return
        }
        #expect(r.version == "0.10.0")
        #expect(r.assetURL?.absoluteString == "https://example.com/v0.10.0/Hachibu-macos.zip")
    }

    @Test func separatesUpToDateFromFailure() {
        let list = [Self.release("v0.5.0"), Self.release("windows-v0.9.0")]
        #expect(Update.outcome(status: 200, data: Self.json(list), current: "0.5.0") == .upToDate)
        // Releaseが1つも無いのは最新と同じ
        #expect(Update.outcome(status: 200, data: Self.json([]), current: "0.5.0") == .upToDate)
        // 通信の失敗、200以外、読めない本文は失敗。最新とは扱わない
        #expect(Update.outcome(status: nil, data: nil, current: "0.5.0") == .failed)
        #expect(Update.outcome(status: 403, data: Self.json(list), current: "0.5.0") == .failed)
        #expect(Update.outcome(status: 200, data: Data("{}".utf8), current: "0.5.0") == .failed)
    }

    @Test func readsTheAssetAndChecksum() {
        let sha = String(repeating: "Ab", count: 32)
        let r = Update.parseRelease(Self.release("v0.6.0", body: "Notes\n\nSHA-256 of Hachibu-macos.zip: `\(sha)`\n"))
        #expect(r?.sha256 == sha.lowercased())
        #expect(r?.assetURL != nil)
        let bare = Update.parseRelease(Self.release("v0.6.0", asset: false, body: "SHA-256 of Hachibu-macos.zip: `abc`"))
        #expect(bare?.assetURL == nil)
        #expect(bare?.sha256 == nil)
    }

    @Test func notifiesEachVersionOnce() {
        let r = Update.Release(version: "0.4.0", url: URL(string: "https://example.com")!)
        #expect(Update.shouldNotify(r, current: "0.3.2", notified: nil))
        #expect(Update.shouldNotify(r, current: "0.3.2", notified: "0.3.3"))
        #expect(!Update.shouldNotify(r, current: "0.3.2", notified: "0.4.0"))
        #expect(!Update.shouldNotify(r, current: "0.4.0", notified: nil))
    }
}
