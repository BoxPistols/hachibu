import Foundation
import Testing
@testable import HachibuCore

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
        #expect(s.limits.map(\.kind) == [.fiveHour, .weekly])
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
            UsageLimit(kind: .fiveHour, percent: 9, resetsAt: nil),
            UsageLimit(kind: .weekly, percent: 75, resetsAt: nil),
            UsageLimit(kind: .model("Examplemodel"), percent: 55, resetsAt: nil),
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
        #expect(ResetFormatter.text(sameDay, now: now, calendar: cal, strings: .ja) == "23:05")
        #expect(ResetFormatter.text(later, now: now, calendar: cal, strings: .ja) == "1/5(土) 7:30")
    }
}

@Suite struct LimitLineTests {
    @Test func noSpaceBetweenJapaneseAndDigits() {
        #expect(UsageLimit(kind: .weekly, percent: 75, resetsAt: nil).line(.ja) == "週枠75%")
        #expect(UsageLimit(kind: .model("Examplemodel"), percent: 55, resetsAt: nil).line(.ja) == "Examplemodel 55%")
    }
}

@Suite struct EnglishStringsTests {
    @Test func limitLinesInEnglish() {
        #expect(UsageLimit(kind: .weekly, percent: 75, resetsAt: nil).line(.en) == "Weekly 75%")
        #expect(UsageLimit(kind: .fiveHour, percent: 9, resetsAt: nil).line(.en) == "5-hour 9%")
    }

    @Test func resetTimeInEnglish() {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "UTC")!
        let now = cal.date(from: DateComponents(year: 2030, month: 1, day: 2, hour: 9))!
        let later = cal.date(from: DateComponents(year: 2030, month: 1, day: 5, hour: 7, minute: 30))!
        let limit = UsageLimit(kind: .weekly, percent: 80, resetsAt: later)
        #expect(limit.line(.en, now: now, calendar: cal) == "Weekly 80% (resets Sat 1/5 7:30)")
        #expect(limit.line(.ja, now: now, calendar: cal) == "週枠80%（1/5(土) 7:30にリセット）")
    }

    @Test func picksLanguageFromPreferencesAndOverride() {
        #expect(Language.detect(preferred: ["ja-JP", "en-US"], override: nil) == .ja)
        #expect(Language.detect(preferred: ["en-US", "ja-JP"], override: nil) == .en)
        #expect(Language.detect(preferred: ["fr-FR"], override: nil) == .en)
        #expect(Language.detect(preferred: ["ja-JP"], override: "en") == .en)
        #expect(Language.detect(preferred: ["en-US"], override: "xx") == .en)
    }

    @Test func readsTheContextLengthFromTheWindowSize() {
        let big: [String: Any] = ["model": ["display_name": "Examplemodel 5.1"], "context_window": ["context_window_size": 1_000_000]]
        #expect(StatusLine.parse(big).model == "Examplemodel5.1 1M")
        let normal: [String: Any] = ["model": ["display_name": "Examplemodel 5.1"], "context_window": ["context_window_size": 200_000]]
        #expect(StatusLine.parse(normal).model == "Examplemodel5.1")
        // 表示名にすでに入っていれば重ねない
        let named: [String: Any] = ["model": ["display_name": "Examplemodel 5 (1M context)"], "context_window": ["context_window_size": 1_000_000]]
        #expect(StatusLine.parse(named).model == "Examplemodel5 1M")
    }

    @Test func languageChosenInTheMenuWinsOverMacOS() {
        #expect(Language.detect(preferred: ["ja-JP"], saved: "en", override: nil) == .en)
        #expect(Language.detect(preferred: ["en-US"], saved: "ja", override: nil) == .ja)
        // 撮影用の環境変数はメニューの選択よりも優先する
        #expect(Language.detect(preferred: ["ja-JP"], saved: "ja", override: "en") == .en)
        // 読めない値が保存されていたらmacOSに合わせる
        #expect(Language.detect(preferred: ["ja-JP"], saved: "xx", override: nil) == .ja)
    }

    @Test func englishLogLinesHaveNoJapanese() {
        let s = Strings.en
        let lines = [
            s.logShortcutCommitted("⌥⌘/"), s.logShortcutDisabled, s.logShortcutFailed("⌥⌘/"),
            s.logUsageAPIEnabled, s.logUsageAPIDeclined, s.logUsageAPIDisabled,
            s.logLoginItemOn, s.logLoginItemOff, s.logOpened, s.logLanguageChanged("English"),
            s.usageAPIFailed("HTTP 500"), s.loginItemFailed("example error"),
        ]
        for line in lines {
            #expect(!line.unicodeScalars.contains { (0x3000...0x9FFF).contains($0.value) }, "\(line)")
        }
    }

    @Test func everyModeAndStyleHasAnEnglishName() {
        for layout in StripLayout.allCases {
            #expect(!Strings.en.layoutName(layout).isEmpty)
            #expect(Strings.en.layoutName(layout) != Strings.ja.layoutName(layout))
        }
        for style in MenuBarStyle.allCases {
            #expect(Strings.en.menuBarStyleName(style) != Strings.ja.menuBarStyleName(style))
        }
    }
}
