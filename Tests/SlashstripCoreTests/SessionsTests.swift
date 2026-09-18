import Foundation
import Testing
@testable import SlashstripCore

@Suite struct SessionsTests {
    static func json(_ id: String, cwd: String = "/tmp/example-project", state: String = "idle",
                     kind: String? = nil, start: String? = nil, pid: Int = 100,
                     guid: String? = "GUID-1", ui: String = "tty", tool: String = "") -> [String: Any] {
        var d: [String: Any] = ["session_id": id, "cwd": cwd, "state": state, "pid": pid, "ui": ui, "tool": tool]
        if let kind { d["waiting_kind"] = kind }
        if let start { d["pid_start"] = start }
        if let guid { d["term_guid"] = guid }
        return d
    }

    @Test func parsesStateAndProject() {
        let s = SessionInfo.parse(Self.json("a", cwd: "/tmp/work/example-app", state: "waiting", kind: "permission"))
        #expect(s?.project == "example-app")
        #expect(s?.state == .waitingPermission)
        #expect(SessionInfo.parse(Self.json("b", state: "waiting", kind: "input"))?.state == .waitingInput)
        #expect(SessionInfo.parse(Self.json("c", state: "busy"))?.state == .busy)
        #expect(SessionInfo.parse(Self.json("d", state: "something"))?.state == .idle)
        #expect(SessionInfo.parse(["cwd": "/tmp/x"]) == nil)
    }

    @Test func desktopSessionsDoNotUseInheritedTerminalID() {
        // デスクトップアプリ駆動のterm_guidは起動元の残りで、そのタブを選んでも該当セッションは出ない
        let s = SessionInfo.parse(Self.json("p", guid: "GUID-INHERITED", ui: "peers"))
        #expect(s?.termGUID == nil)
        #expect(s?.hostBundle == "com.anthropic.claudefordesktop")
    }

    @Test func parsesProcessStartWithPaddedDay() {
        let a = SessionInfo.parseProcessStart("Fri Sep 18 21:30:37 2026")
        let b = SessionInfo.parseProcessStart("Tue Sep  8 09:05:00 2026")
        #expect(a != nil)
        #expect(b != nil)
        #expect(b! < a!)
        #expect(SessionInfo.parseProcessStart("garbage") == nil)
    }

    @Test func ordersByProcessStartNotByState() {
        // 状態が変わっても番号が入れ替わらないこと。許可待ちでも後から始まったものは後ろ
        let older = SessionInfo.parse(Self.json("old", state: "idle", start: "Fri Sep 18 20:00:00 2026", pid: 300))!
        let newer = SessionInfo.parse(Self.json("new", state: "waiting", kind: "permission",
                                                start: "Fri Sep 18 21:00:00 2026", pid: 200))!
        let unknown = SessionInfo.parse(Self.json("unk", state: "busy", pid: 50))!
        #expect(SessionInfo.ordered([newer, unknown, older]).map(\.id) == ["old", "new", "unk"])
        #expect(SessionInfo.ordered([older, newer, unknown]).map(\.id) == ["old", "new", "unk"])
    }

    @Test func menuTitleShowsToolOnlyWhileBusy() {
        let busy = SessionInfo.parse(Self.json("a", cwd: "/tmp/example-app", state: "busy", tool: "Bash"))!
        let idle = SessionInfo.parse(Self.json("b", cwd: "/tmp/example-app", state: "idle", tool: "Bash"))!
        #expect(busy.menuTitle(number: 1, strings: .ja) == "1. example-app（実行中・Bash）")
        #expect(idle.menuTitle(number: 2, strings: .ja) == "2. example-app（待機中）")
    }
}

@Suite struct ShortcutTests {
    @Test func displaysInAppleOrder() {
        let s = Shortcut(keyCode: 40, modifiers: [.command, .shift, .control, .option], keyLabel: "K")
        #expect(s.display == "⌃⌥⇧⌘K")
        #expect(Shortcut.defaultSummon.display == "⌥⌘/")
    }

    @Test func requiresCommandOptionOrControl() {
        #expect(!Shortcut(keyCode: 0, modifiers: [], keyLabel: "A").isAcceptable)
        #expect(!Shortcut(keyCode: 0, modifiers: [.shift], keyLabel: "A").isAcceptable)
        #expect(Shortcut(keyCode: 0, modifiers: [.control], keyLabel: "A").isAcceptable)
        #expect(!Shortcut(keyCode: 0, modifiers: [.command], keyLabel: "").isAcceptable)
    }

    @Test func mapsToCarbonModifierBits() {
        #expect(Shortcut.defaultSummon.carbonModifiers == 256 | 2048)
        #expect(Shortcut(keyCode: 0, modifiers: [.control, .shift], keyLabel: "A").carbonModifiers == 4096 | 512)
    }

    @Test func labelsKeysWithoutCharacters() {
        #expect(Shortcut.label(keyCode: 49, characters: " ") == "Space")
        #expect(Shortcut.label(keyCode: 122, characters: "\u{F704}") == "F1")
        #expect(Shortcut.label(keyCode: 40, characters: "k") == "K")
    }

    @Test func survivesSaveAndLoad() throws {
        let data = try JSONEncoder().encode(Shortcut.defaultSummon)
        #expect(try JSONDecoder().decode(Shortcut.self, from: data) == .defaultSummon)
    }
}
