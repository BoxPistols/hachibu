import Testing
@testable import HachibuCore

@Suite struct LayoutRulesTests {
    static func status(_ text: String = "Opus5 1M xhigh · S12 W76") -> Slot {
        Slot(id: Slot.statusID, text: text, background: .idleBackground, foreground: .idleForeground, help: "reset info")
    }

    @Test func expandingAlwaysShowsTheStandardStrip() {
        for rest in StripLayout.allCases {
            #expect(LayoutRules.effective(rest: rest, expanded: true) == .full)
        }
    }

    @Test func restingModeIsKeptOtherwise() {
        for rest in StripLayout.allCases {
            #expect(LayoutRules.effective(rest: rest, expanded: false) == rest)
        }
    }

    @Test func usageModeReplacesOnlyTheText() {
        let shown = LayoutRules.visibleSlots([Self.status()], layout: .usage, usageText: "S12 W76")
        #expect(shown.count == 1)
        #expect(shown[0].text == "S12 W76")
        #expect(shown[0].help == "reset info")
        #expect(shown[0].background == .idleBackground)
    }

    @Test func usageModeKeepsTheTextWhenUsageIsUnknown() {
        #expect(LayoutRules.visibleSlots([Self.status("Opus5")], layout: .usage, usageText: nil)[0].text == "Opus5")
    }

    @Test func tabAndHidden() {
        #expect(LayoutRules.visibleSlots([Self.status()], layout: .tab).map(\.id) == [Slot.statusID])
        #expect(LayoutRules.visibleSlots([Self.status()], layout: .hidden).isEmpty)
        #expect(LayoutRules.visibleSlots([Self.status()], layout: .full) == [Self.status()])
    }

    @Test func savedCompactModeFromOlderVersionsIsNotAMode() {
        // 以前の版で保存された"compact"は読めない。呼び出し側は既定（標準）に戻す
        #expect(StripLayout(rawValue: "compact") == nil)
    }
}

@Suite struct MenuBarStyleTests {
    let limits = [
        UsageLimit(kind: .fiveHour, percent: 11, resetsAt: nil),
        UsageLimit(kind: .weekly, percent: 75, resetsAt: nil),
    ]

    @Test func titles() {
        #expect(MenuBarStyle.title(style: .icon, statusText: "Opus5 xhigh · S11 W75", limits: limits) == nil)
        #expect(MenuBarStyle.title(style: .usage, statusText: "Opus5 xhigh · S11 W75", limits: limits) == "S11 W75")
        #expect(MenuBarStyle.title(style: .status, statusText: "Opus5 xhigh · S11 W75", limits: limits) == "Opus5 xhigh · S11 W75")
        #expect(MenuBarStyle.title(style: .usage, statusText: nil, limits: []) == nil)
    }

    @Test func cyclingSkipsHidden() {
        #expect(LayoutRules.next(after: .full) == .usage)
        #expect(LayoutRules.next(after: .usage) == .tab)
        #expect(LayoutRules.next(after: .tab) == .full)
        #expect(LayoutRules.next(after: .hidden) == .full)
    }

    @Test func theGaugeReadsTheWeeklyLimit() {
        let limits = [UsageLimit(kind: .fiveHour, percent: 38, resetsAt: nil),
                      UsageLimit(kind: .weekly, percent: 68, resetsAt: nil),
                      UsageLimit(kind: .model("Examplemodel"), percent: 90, resetsAt: nil)]
        #expect(GaugeReading.from(limits) == GaugeReading(label: "W68", percent: 68))
        // 週枠が無ければ最初の枠。何も無ければ出さない
        #expect(GaugeReading.from([limits[0]]) == GaugeReading(label: "S38", percent: 38))
        #expect(GaugeReading.from([]) == nil)
        // 目盛りは絵で示すので、文字は返さない
        #expect(MenuBarStyle.title(style: .gauge, statusText: "Examplemodel5 · S38 W68", limits: limits) == nil)
    }
}
