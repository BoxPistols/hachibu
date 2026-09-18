import Foundation
import Testing
@testable import SlashstripCore

@Suite struct WidgetsOrderTests {
    @Test func placesStatusPermissionsMenusThenCommands() {
        let names: Set<String> = ["cmd-1", "menu-2", "perm-reject", "cmd-0", "status", "perm-allow", "menu-1"]
        #expect(Widgets.ordered(names) == ["status", "perm-allow", "perm-reject", "menu-1", "menu-2", "cmd-0", "cmd-1"])
    }

    @Test func sortsNumberedWidgetsNumericallyNotLexically() {
        let names: Set<String> = ["status", "cmd-10", "cmd-2", "cmd-1"]
        #expect(Widgets.ordered(names) == ["status", "cmd-1", "cmd-2", "cmd-10"])
    }

    @Test func ignoresUnknownAndMalformedNames() {
        let names: Set<String> = ["status", "daemon", "cmd-x", "perm-maybe", "menu--1", "cmd-3"]
        #expect(Widgets.ordered(names) == ["status", "cmd-3"])
    }

    @Test func alwaysStartsWithStatusEvenIfFileIsMissing() {
        // status.jsonが無いときはアプリ側が仮表示を出すので、並びには必ず含める
        #expect(Widgets.ordered(["cmd-0"]) == ["status", "cmd-0"])
    }
}

@Suite struct WidgetsSlotTests {
    @Test func hiddenOrEmptyWidgetsAreNotShown() {
        #expect(Widgets.slot(name: "cmd-0", json: ["text": "review", "hidden": true]) == nil)
        #expect(Widgets.slot(name: "cmd-0", json: ["text": "", "hidden": false]) == nil)
        #expect(Widgets.slot(name: "cmd-0", json: [:]) == nil)
    }

    @Test func readsTextAndColors() {
        let slot = Widgets.slot(name: "cmd-1", json: [
            "text": "model", "background_color": "104,72,160,255", "font_color": "20,20,20,255", "hidden": false,
        ])
        #expect(slot?.text == "model")
        #expect(slot?.background == RGBA(r: 104, g: 72, b: 160, a: 255))
        #expect(slot?.foreground == RGBA(r: 20, g: 20, b: 20, a: 255))
    }

    @Test func fallsBackToDefaultColorsWhenMalformed() {
        let slot = Widgets.slot(name: "cmd-0", json: ["text": "usage", "background_color": "1,2,3", "font_color": "a,b,c,d"])
        #expect(slot?.background == .commandBackground)
        #expect(slot?.foreground == .white)
    }

    @Test func rejectsOutOfRangeColorComponents() {
        #expect(RGBA(csv: "256,0,0,255") == nil)
        #expect(RGBA(csv: "-1,0,0,255") == nil)
        #expect(RGBA(csv: "0, 0, 0, 0") == .clear)
    }
}

@Suite struct WidgetsArgumentsTests {
    let base = URL(fileURLWithPath: "/tmp/example-home/.claude/btt")

    @Test func mapsEachWidgetToTheSameScriptAsTheTouchBar() {
        #expect(Widgets.arguments(for: "status", base: base) == ["/tmp/example-home/.claude/btt/cc-focus.py"])
        #expect(Widgets.arguments(for: "perm-always", base: base) == ["/tmp/example-home/.claude/btt/cc-send.py", "always"])
        #expect(Widgets.arguments(for: "menu-3", base: base) == ["/tmp/example-home/.claude/btt/cc-menu.py", "send", "3"])
        #expect(Widgets.arguments(for: "cmd-0", base: base) == ["/tmp/example-home/.claude/btt/cc-menu.py", "run", "0"])
    }

    @Test func refusesUnknownOrMalformedIDs() {
        // 知らないidでスクリプトを呼ぶと、意図しない引数がセッションに送られうる
        for id in ["perm-maybe", "menu-0", "menu-x", "cmd--1", "cmd-", "status2", "", "cmd-1;rm"] {
            #expect(Widgets.arguments(for: id, base: base) == nil, "id=\(id)")
        }
    }
}

@Suite struct WidgetsLimitsTests {
    @Test func readsStatusLineRatesAsPercentIntegers() {
        // statusLine由来の値は0〜100。1を比率と見て100%にしてはいけない
        let limits = Widgets.limits(usage: ["session": 1, "week": 75, "session_resets_at": 1_800_000_000], usageAPI: nil)
        #expect(limits.map(\.percent) == [1, 75])
        #expect(limits.map(\.kind) == [.fiveHour, .weekly])
        #expect(limits[0].resetsAt == Date(timeIntervalSince1970: 1_800_000_000))
        #expect(limits[1].resetsAt == nil)
    }

    @Test func appendsModelScopedLimits() {
        let limits = Widgets.limits(usage: ["session": 9, "week": 75],
                                    usageAPI: ["scoped": [["name": "Examplemodel", "pct": 55, "resets_at": "2030-01-02T03:04:05.123456+00:00"]]])
        #expect(limits.map(\.kind) == [.fiveHour, .weekly, .model("Examplemodel")])
        #expect(limits[2].resetsAt == ResetFormatter.parseISO("2030-01-02T03:04:05+00:00"))
    }

    @Test func toleratesMissingFiles() {
        #expect(Widgets.limits(usage: nil, usageAPI: nil).isEmpty)
    }
}

@Suite struct IdleStatusTests {
    let limits = [
        UsageLimit(kind: .fiveHour, percent: 12, resetsAt: nil),
        UsageLimit(kind: .weekly, percent: 76, resetsAt: nil),
        UsageLimit(kind: .model("Examplemodel"), percent: 55, resetsAt: nil),
    ]
    let now = Date(timeIntervalSince1970: 2_000_000_000)
    let idle = Slot(id: "status", text: Widgets.noSessionText, background: .clear, foreground: .idleForeground)

    @Test func showsLastKnownUsageWhenNoSessionIsRunning() {
        let s = Widgets.idleStatus(idle, limits: limits, asOf: now.addingTimeInterval(-60), now: now)
        #expect(s.text == "S12 W76 E55")
        #expect(!s.dimmed)
        #expect(s.background == .clear)
    }

    @Test func dimsUsageOlderThanThirtyMinutes() {
        let s = Widgets.idleStatus(idle, limits: limits, asOf: now.addingTimeInterval(-31 * 60), now: now)
        #expect(s.dimmed)
        #expect(!Widgets.idleStatus(idle, limits: limits, asOf: now.addingTimeInterval(-29 * 60), now: now).dimmed)
    }

    @Test func leavesRunningSessionsAndMissingUsageAlone() {
        let running = Slot(id: "status", text: "Opus5 1M xhigh · S12 W76", background: .idleBackground, foreground: .white)
        #expect(Widgets.idleStatus(running, limits: limits, asOf: nil, now: now) == running)
        #expect(Widgets.idleStatus(idle, limits: [], asOf: nil, now: now) == idle)
    }

    @Test func unknownTimeIsNotStale() {
        #expect(!Usage.isStale(nil))
    }
}
