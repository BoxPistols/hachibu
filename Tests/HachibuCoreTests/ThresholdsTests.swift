import Testing
@testable import HachibuCore

@Suite struct ThresholdsTests {
    @Test func defaultIsSeventyAndNinety() {
        #expect(UsageThresholds.default.warning == 70)
        #expect(UsageThresholds.default.critical == 90)
    }

    @Test func levelsAtBoundaries() {
        let t = UsageThresholds.default
        #expect(t.level(69) == .normal)
        #expect(t.level(70) == .warning)
        #expect(t.level(89) == .warning)
        #expect(t.level(90) == .critical)
        #expect(t.level(100) == .critical)
    }

    @Test func rejectsRedBelowOrEqualYellow() {
        #expect(UsageThresholds(warning: 80, critical: 80) == nil)
        #expect(UsageThresholds(warning: 85, critical: 80) == nil)
        #expect(UsageThresholds(warning: 0, critical: 80) == nil)
        #expect(UsageThresholds(warning: 50, critical: 101) == nil)
        #expect(UsageThresholds(warning: 50, critical: 80) != nil)
    }

    @Test func everyDefaultChoiceCombinationWithRedAboveYellowIsValid() {
        for w in UsageThresholds.warningChoices {
            for c in UsageThresholds.criticalChoices where c > w {
                #expect(UsageThresholds(warning: w, critical: c) != nil)
            }
        }
    }

    @Test func worstOfLimits() {
        let limits = [
            UsageLimit(kind: .fiveHour, percent: 12, resetsAt: nil),
            UsageLimit(kind: .weekly, percent: 76, resetsAt: nil),
        ]
        #expect(UsageThresholds.default.worst(limits) == .warning)
        #expect(UsageThresholds.default.worst([]) == .normal)
    }
}

@Suite struct UsageMarkupTests {
    let t = UsageThresholds.default

    @Test func marksOnlyUsageTokensOverThreshold() {
        let s = UsageMarkup.segments("Opus5 1M xhigh · S12 W76 F95", thresholds: t)
        #expect(s == [
            TextSegment(text: "Opus5 1M xhigh · S12 ", level: .normal),
            TextSegment(text: "W76", level: .warning),
            TextSegment(text: " ", level: .normal),
            TextSegment(text: "F95", level: .critical),
        ])
    }

    @Test func keepsTextIntactWhenJoinedBack() {
        let text = "Opus5 1M xhigh · S72 W91 F55 ctx91%"
        #expect(UsageMarkup.segments(text, thresholds: t).map(\.text).joined() == text)
    }

    @Test func leavesNonUsageTokensAlone() {
        // モデル名の数字、コンテキスト使用率、番号ボタンのラベルは使用率ではない
        for text in ["Opus5 1M xhigh", "ctx95%", "1 Def", "CC …", "許可待ち"] {
            #expect(UsageMarkup.segments(text, thresholds: t) == [TextSegment(text: text, level: .normal)], "\(text)")
        }
    }

    @Test func usageOnlyText() {
        let s = UsageMarkup.segments("S95 W50", thresholds: t)
        #expect(s == [TextSegment(text: "S95", level: .critical), TextSegment(text: " W50", level: .normal)])
    }
}
