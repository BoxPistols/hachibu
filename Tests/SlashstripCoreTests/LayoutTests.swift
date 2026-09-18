import Testing
@testable import SlashstripCore

@Suite struct LayoutRulesTests {
    static func slot(_ id: String) -> Slot {
        Slot(id: id, text: id, background: .commandBackground, foreground: .white)
    }

    static let idle = ["status", "cmd-0", "cmd-1"].map(slot)
    static let waitingPermission = ["status", "perm-allow", "perm-reject"].map(slot)
    static let menuOpen = ["status", "menu-1", "menu-2"].map(slot)

    @Test func expandingAlwaysShowsEverything() {
        for rest in StripLayout.allCases {
            #expect(LayoutRules.effective(rest: rest, expanded: true, slots: Self.idle) == .full)
        }
    }

    @Test func restingModeIsKeptWhenNothingNeedsAnAnswer() {
        for rest in StripLayout.allCases {
            #expect(LayoutRules.effective(rest: rest, expanded: false, slots: Self.idle) == rest)
        }
    }

    @Test func tabOpensToCompactForPermissionAndMenu() {
        #expect(LayoutRules.effective(rest: .tab, expanded: false, slots: Self.waitingPermission) == .compact)
        #expect(LayoutRules.effective(rest: .tab, expanded: false, slots: Self.menuOpen) == .compact)
    }

    @Test func hiddenOpensWheneverAnAnswerIsNeeded() {
        // 3択以下の許可は応答ボタン、4択以上の質問は番号ボタンで出る。どちらでも同じように開く
        #expect(LayoutRules.effective(rest: .hidden, expanded: false, slots: Self.menuOpen) == .compact)
        #expect(LayoutRules.effective(rest: .hidden, expanded: false, slots: Self.waitingPermission) == .compact)
    }

    @Test func compactKeepsStatusAndAnswerButtonsOnly() {
        let all = ["status", "perm-allow", "menu-1", "cmd-0", "cmd-1"].map(Self.slot)
        #expect(LayoutRules.visibleSlots(all, layout: .compact).map(\.id) == ["status", "perm-allow", "menu-1"])
        #expect(LayoutRules.visibleSlots(all, layout: .tab).map(\.id) == ["status"])
        #expect(LayoutRules.visibleSlots(all, layout: .hidden).isEmpty)
        #expect(LayoutRules.visibleSlots(all, layout: .full).map(\.id) == all.map(\.id))
    }
}

@Suite struct MenuBarStyleTests {
    let limits = [
        UsageLimit(name: L10n.limitFiveHour, percent: 11, resetsAt: nil),
        UsageLimit(name: L10n.limitWeekly, percent: 75, resetsAt: nil),
    ]

    @Test func titles() {
        #expect(MenuBarStyle.title(style: .icon, statusText: "Opus5 xhigh · S11 W75", limits: limits) == nil)
        #expect(MenuBarStyle.title(style: .usage, statusText: "Opus5 xhigh · S11 W75", limits: limits) == "S11 W75")
        #expect(MenuBarStyle.title(style: .status, statusText: "Opus5 xhigh · S11 W75", limits: limits) == "Opus5 xhigh · S11 W75")
        #expect(MenuBarStyle.title(style: .usage, statusText: nil, limits: []) == nil)
    }
}

@Suite struct UsageLayoutTests {
    @Test func replacesStatusTextWithUsageAndKeepsAnswerButtons() {
        let status = Slot(id: "status", text: "Opus5 xhigh · S12 W76", background: .idleBackground, foreground: .white,
                          help: "reset info")
        let all = [status] + ["perm-allow", "cmd-0"].map(LayoutRulesTests.slot)
        let shown = LayoutRules.visibleSlots(all, layout: .usage, usageText: "S12 W76")
        #expect(shown.map(\.id) == ["status", "perm-allow"])
        #expect(shown[0].text == "S12 W76")
        #expect(shown[0].help == "reset info")
        #expect(shown[0].background == .idleBackground)
    }

    @Test func keepsOriginalTextWhenUsageIsUnknown() {
        let all = [LayoutRulesTests.slot("status")]
        #expect(LayoutRules.visibleSlots(all, layout: .usage, usageText: nil)[0].text == "status")
    }
}
