import Foundation
import Testing
@testable import HachibuCore

struct LegendTests {
    private let limits = [
        UsageLimit(kind: .fiveHour, percent: 6, resetsAt: nil),
        UsageLimit(kind: .weekly, percent: 85, resetsAt: nil),
        UsageLimit(kind: .model("Examplemodel"), percent: 57, resetsAt: nil),
    ]

    @Test func explainsEveryMarkOnTheStrip() {
        let lines = Legend.lines(statusText: "Examplemodel5 1M xhigh · S6 W85 E57", limits: limits,
                                 thresholds: UsageThresholds(warning: 70, critical: 90)!, strings: .en)
        #expect(lines.map(\.text) == [
            "Dot after the model name: 1M context",
            "S: 5-hour limit",
            "W: Weekly limit",
            "E: weekly limit for Examplemodel",
            "70% or more",
            "90% or more",
        ])
        #expect(lines.first?.mark == .contextDot)
        #expect(lines.last?.mark == .level(.critical))
    }

    @Test func leavesOutTheDotWhenThereIsNone() {
        let lines = Legend.lines(statusText: "Examplemodel5 xhigh · S6", limits: [], thresholds: UsageThresholds(warning: 50, critical: 80)!,
                                 strings: .ja)
        #expect(lines.map(\.text) == ["50%以上", "80%以上"])
    }
}
