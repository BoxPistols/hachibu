import Foundation
import Testing
@testable import SlashstripCore

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
