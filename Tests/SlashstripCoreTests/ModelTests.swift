import Foundation
import Testing
@testable import SlashstripCore

@Suite struct RGBATests {
    @Test func rejectsOutOfRangeColorComponents() {
        #expect(RGBA(csv: "256,0,0,255") == nil)
        #expect(RGBA(csv: "-1,0,0,255") == nil)
        #expect(RGBA(csv: "0, 0, 0, 0") == .clear)
    }

    @Test func readsValidComponents() {
        #expect(RGBA(csv: "104,72,160,255") == RGBA(r: 104, g: 72, b: 160, a: 255))
        #expect(RGBA(csv: "1,2,3") == nil)
        #expect(RGBA(csv: nil) == nil)
    }
}

@Suite struct StalenessTests {
    @Test func unknownTimeIsNotStale() {
        #expect(!Usage.isStale(nil))
    }

    @Test func olderThanThirtyMinutesIsStale() {
        let now = Date(timeIntervalSince1970: 2_000_000_000)
        #expect(Usage.isStale(now.addingTimeInterval(-31 * 60), now: now))
        #expect(!Usage.isStale(now.addingTimeInterval(-29 * 60), now: now))
    }
}
