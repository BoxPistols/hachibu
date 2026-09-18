import Foundation
import Testing
@testable import SlashstripCore

@Suite struct StatusLineTests {
    /// 公式ドキュメントのstatusLineのJSONの形。値は架空
    static let payload: [String: Any] = [
        "model": ["id": "example-model-id", "display_name": "Opus 5 (1M context)"],
        "effort": ["level": "xhigh"],
        "rate_limits": [
            "five_hour": ["used_percentage": 9, "resets_at": 1_900_000_000],
            "seven_day": ["used_percentage": 75.4, "resets_at": 1_900_100_000],
        ],
    ]

    @Test func readsModelEffortAndLimits() {
        let s = StatusLine.parse(Self.payload)
        #expect(s.model == "Opus5 1M")
        #expect(s.effort == "xhigh")
        #expect(s.limits.map(\.name) == [L10n.limitFiveHour, L10n.limitWeekly])
        #expect(s.limits.map(\.percent) == [9, 75])
        #expect(s.limits[0].resetsAt == Date(timeIntervalSince1970: 1_900_000_000))
    }

    @Test func limitsAreEmptyBeforeTheFirstResponse() {
        var p = Self.payload
        p["rate_limits"] = nil
        #expect(StatusLine.parse(p).limits.isEmpty)
        #expect(StatusLine.parse([:]) == StatusLine.Snapshot(model: nil, effort: nil, limits: []))
    }
}

@Suite struct UsageTextTests {
    @Test func compactsDisplayNames() {
        #expect(Usage.compactDisplayName("Opus 5 (1M context)") == "Opus5 1M")
        #expect(Usage.compactDisplayName("Sonnet 5") == "Sonnet5")
        #expect(Usage.compactDisplayName("Haiku 4.5") == "Haiku4.5")
        #expect(Usage.compactDisplayName("Example (200k context)") == "Example 200K")
    }

    @Test func buildsTouchBarStyleStatus() {
        let limits = [
            UsageLimit(name: L10n.limitFiveHour, percent: 9, resetsAt: nil),
            UsageLimit(name: L10n.limitWeekly, percent: 75, resetsAt: nil),
            UsageLimit(name: "Examplemodel", percent: 55, resetsAt: nil),
        ]
        #expect(Usage.statusText(model: "Opus5 1M", effort: "xhigh", limits: limits) == "Opus5 1M xhigh · S9 W75 E55")
        #expect(Usage.statusText(model: nil, effort: nil, limits: limits) == "S9 W75 E55")
        #expect(Usage.statusText(model: "Opus5", effort: nil, limits: []) == "Opus5")
        #expect(Usage.statusText(model: nil, effort: nil, limits: []) == L10n.statusPlaceholder)
    }

    @Test func parsesFractionalSecondsOfAnyLength() {
        let a = ResetFormatter.parseISO("2030-01-02T03:04:05.865631+00:00")
        let b = ResetFormatter.parseISO("2030-01-02T03:04:05.8+00:00")
        let c = ResetFormatter.parseISO("2030-01-02T03:04:05Z")
        #expect(a != nil)
        #expect(a == b)
        #expect(b == c)
        #expect(ResetFormatter.parseISO("not a date") == nil)
    }

    @Test func formatsResetTimeRelativeToToday() {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "Asia/Tokyo")!
        let now = cal.date(from: DateComponents(year: 2030, month: 1, day: 2, hour: 9))!
        let sameDay = cal.date(from: DateComponents(year: 2030, month: 1, day: 2, hour: 23, minute: 5))!
        let later = cal.date(from: DateComponents(year: 2030, month: 1, day: 5, hour: 7, minute: 30))!
        #expect(ResetFormatter.text(sameDay, now: now, calendar: cal) == "23:05")
        #expect(ResetFormatter.text(later, now: now, calendar: cal) == "1/5(土) 7:30")
    }
}

@Suite struct LimitLineTests {
    @Test func noSpaceBetweenJapaneseAndDigits() {
        #expect(UsageLimit(name: L10n.limitWeekly, percent: 75, resetsAt: nil).line == "週枠75%")
        #expect(UsageLimit(name: "Examplemodel", percent: 55, resetsAt: nil).line == "Examplemodel 55%")
    }
}
