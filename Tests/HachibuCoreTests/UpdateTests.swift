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

    @Test func notifiesEachVersionOnce() {
        let r = Update.Release(version: "0.4.0", url: URL(string: "https://example.com")!)
        #expect(Update.shouldNotify(r, current: "0.3.2", notified: nil))
        #expect(Update.shouldNotify(r, current: "0.3.2", notified: "0.3.3"))
        #expect(!Update.shouldNotify(r, current: "0.3.2", notified: "0.4.0"))
        #expect(!Update.shouldNotify(r, current: "0.4.0", notified: nil))
    }
}
