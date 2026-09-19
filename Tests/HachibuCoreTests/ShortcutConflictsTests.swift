import Foundation
import Testing
@testable import HachibuCore

@Suite struct ShortcutConflictsTests {
    /// com.apple.symbolichotkeysの形。修飾キーの値にはテンキー（1<<21）やfn（1<<23）のビットが混ざる
    static func entry(_ enabled: Bool, _ char: Int, _ key: Int, _ mods: Int) -> [String: Any] {
        ["enabled": NSNumber(value: enabled),
         "value": ["parameters": [NSNumber(value: char), NSNumber(value: key), NSNumber(value: mods)], "type": "standard"]]
    }

    static let control = 1 << 18
    static let option = 1 << 19
    static let command = 1 << 20
    static let shift = 1 << 17
    static let numericPad = 1 << 21

    @Test func rejectsEnabledSystemShortcutFromSettings() {
        // Spotlightを⌃Spaceへ変えている設定
        let symbolic: [String: Any] = ["64": Self.entry(true, 32, 49, Self.control)]
        let ctrlSpace = Shortcut(keyCode: 49, modifiers: [.control], keyLabel: "Space")
        #expect(ShortcutConflicts.conflict(for: ctrlSpace, symbolic: symbolic) == "64")
    }

    @Test func usesSettingsOverBuiltInDefaults() {
        // Spotlightが⌃Spaceへ移されていれば、⌘Spaceはもう予約されていない
        let symbolic: [String: Any] = ["64": Self.entry(true, 32, 49, Self.control)]
        let cmdSpace = Shortcut(keyCode: 49, modifiers: [.command], keyLabel: "Space")
        #expect(ShortcutConflicts.conflict(for: cmdSpace, symbolic: symbolic) == nil)
        // 設定が読めなければ既定の割り当てで判定する
        #expect(ShortcutConflicts.conflict(for: cmdSpace, symbolic: nil) == "64")
    }

    @Test func ignoresDisabledEntries() {
        let symbolic: [String: Any] = ["30": Self.entry(false, 52, 21, Self.command | Self.shift)]
        let shot = Shortcut(keyCode: 21, modifiers: [.command, .shift], keyLabel: "4")
        #expect(ShortcutConflicts.conflict(for: shot, symbolic: symbolic) == nil)
    }

    @Test func ignoresNumericPadAndFunctionBits() {
        // 矢印キーの割り当てにはテンキーのビットが付いている
        let symbolic: [String: Any] = ["32": Self.entry(true, 65535, 126, Self.control | Self.numericPad)]
        let ctrlUp = Shortcut(keyCode: 126, modifiers: [.control], keyLabel: "↑")
        #expect(ShortcutConflicts.conflict(for: ctrlUp, symbolic: symbolic) == "32")
    }

    @Test func skipsEntriesWithoutAKey() {
        let symbolic: [String: Any] = ["179": Self.entry(true, 65535, 65535, 0), "16": ["enabled": NSNumber(value: true)]]
        #expect(ShortcutConflicts.parseSymbolic(symbolic).enabled.isEmpty)
    }

    @Test func requiresExactModifiers() {
        let symbolic: [String: Any] = ["28": Self.entry(true, 51, 20, Self.command | Self.shift)]
        // ⌃⇧⌘3は組み込みの一覧（画面をコピー）にあるので、どちらにも無い⌥⇧⌘3で確かめる
        let withOption = Shortcut(keyCode: 20, modifiers: [.command, .shift, .option], keyLabel: "3")
        #expect(ShortcutConflicts.conflict(for: withOption, symbolic: symbolic) == nil)
        let exact = Shortcut(keyCode: 20, modifiers: [.command, .shift], keyLabel: "3")
        #expect(ShortcutConflicts.conflict(for: exact, symbolic: symbolic) == "28")
    }

    @Test func defaultSummonIsFreeOnAStockSystem() {
        #expect(ShortcutConflicts.conflict(for: .defaultSummon, symbolic: nil) == nil)
    }

    @Test func flagsCommandOnlyShortcutsAsMenuProne() {
        #expect(ShortcutConflicts.isAppMenuProne(Shortcut(keyCode: 44, modifiers: [.command], keyLabel: "/")))
        #expect(ShortcutConflicts.isAppMenuProne(Shortcut(keyCode: 44, modifiers: [.command, .shift], keyLabel: "/")))
        #expect(!ShortcutConflicts.isAppMenuProne(.defaultSummon))
        #expect(!ShortcutConflicts.isAppMenuProne(Shortcut(keyCode: 44, modifiers: [.control, .option], keyLabel: "/")))
    }

    @Test func everyReservedIDHasAName() {
        for r in ShortcutConflicts.builtIn {
            #expect(!Strings.en.systemShortcutName(r.id).contains("#"), "\(r.id)")
            #expect(!Strings.ja.systemShortcutName(r.id).contains("番号"), "\(r.id)")
        }
    }
}
