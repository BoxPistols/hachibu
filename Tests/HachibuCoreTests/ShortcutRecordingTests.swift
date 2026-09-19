import Foundation
import Testing
@testable import HachibuCore

/// Carbonの代わりの登録器。いま登録されている組み合わせと、他のアプリが押さえている組み合わせを持つ
final class FakeRegistrar: HotKeyRegistrar {
    var takenByOtherApps: Set<String> = []
    private(set) var registered: Shortcut?

    func register(_ shortcut: Shortcut) -> Bool {
        guard !takenByOtherApps.contains(shortcut.display) else { return false }
        registered = shortcut
        return true
    }

    func unregister() {
        registered = nil
    }
}

/// テストごとに別の保存先を使う（Swift Testingは並列に走る）。ディスクには書かない
final class MemoryStore: SettingsStore {
    private var values: [String: Any] = [:]

    func bool(forKey key: String) -> Bool { values[key] as? Bool ?? false }
    func data(forKey key: String) -> Data? { values[key] as? Data }
    func set(_ value: Any?, forKey key: String) { values[key] = value }
}

func freshDefaults() -> SettingsStore {
    MemoryStore()
}

@Suite struct ShortcutRecordingTests {
    let a = Shortcut(keyCode: 44, modifiers: [.control, .option], keyLabel: "/")
    let b = Shortcut(keyCode: 38, modifiers: [.control, .option, .command], keyLabel: "J")
    let reservedC = Shortcut(keyCode: 49, modifiers: [.control], keyLabel: "Space")
    let d = Shortcut(keyCode: 40, modifiers: [.control, .option, .command], keyLabel: "K")

    /// 保存済みがAで、記録の窓を開いた直後の状態
    func opened(with saved: Shortcut?, registrar: FakeRegistrar = FakeRegistrar())
        -> (HotKeyCenter, ShortcutSettings, FakeRegistrar) {
        let settings = ShortcutSettings(defaults: freshDefaults())
        if let saved { settings.save(saved) } else { settings.saveDisabled() }
        let center = HotKeyCenter(registrar: registrar, settings: settings)
        center.suspend()
        return (center, settings, registrar)
    }

    @Test func cancelAfterAnAcceptedThenRejectedComboRestoresTheOriginal() {
        // 報告された手順: Bを登録 → macOS予約のCで弾かれる → キャンセル
        let (center, settings, registrar) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        #expect(rec.phase == .verifying(b))
        rec.press(reservedC, reservedID: "60", center: center)
        #expect(rec.phase == .rejected(reservedC, .reserved("60")))
        rec.cancel(center: center)

        #expect(center.shortcut == a)
        #expect(registrar.registered == a)
        #expect(settings.load() == a)
    }

    @Test func escOrCancelWhileVerifyingRestoresTheOriginal() {
        let (center, settings, registrar) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        #expect(registrar.registered == b)
        rec.cancel(center: center)
        #expect(registrar.registered == a)
        #expect(settings.load() == a)
    }

    @Test func cancelKeepsAShortcutThatWasOffBeforeOpening() {
        let (center, settings, registrar) = opened(with: nil)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        rec.cancel(center: center)
        #expect(center.shortcut == nil)
        #expect(registrar.registered == nil)
        #expect(settings.load() == nil)
    }

    @Test func nothingIsSavedUntilTheShortcutArrives() {
        let (center, settings, _) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        // 確かめる前にアプリが終わっても、次に読むのは元の値
        #expect(settings.load() == a)
        #expect(center.shortcut == a)

        let saved = rec.arrived(center: center)
        #expect(saved)
        #expect(rec.phase == .saved(b))
        #expect(settings.load() == b)
        #expect(center.shortcut == b)
    }

    @Test func aComboTakenByAnotherAppReleasesThePreviousTrial() {
        let registrar = FakeRegistrar()
        registrar.takenByOtherApps = [d.display]
        let (center, settings, _) = opened(with: a, registrar: registrar)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        rec.press(d, reservedID: nil, center: center)
        #expect(rec.phase == .rejected(d, .taken))
        // 試していたBは外れていて、押しても何も起きない
        #expect(registrar.registered == nil)
        #expect(settings.load() == a)
    }

    @Test func arrivalIsIgnoredUnlessVerifying() {
        let (center, settings, _) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(reservedC, reservedID: "60", center: center)
        let saved = rec.arrived(center: center)
        #expect(!saved)
        #expect(settings.load() == a)
    }

    @Test func pressesAfterSavingAreIgnored() {
        let (center, settings, _) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        _ = rec.arrived(center: center)
        rec.press(d, reservedID: nil, center: center)
        #expect(rec.phase == .saved(b))
        #expect(settings.load() == b)
    }

    @Test func turningOffSavesDisabled() {
        let (center, settings, registrar) = opened(with: a)
        var rec = ShortcutRecording()
        rec.press(b, reservedID: nil, center: center)
        rec.turnOff(center: center)
        #expect(center.shortcut == nil)
        #expect(registrar.registered == nil)
        #expect(settings.load() == nil)
    }

    @Test func missingModifierIsRejectedBeforeRegistering() {
        let (center, _, registrar) = opened(with: a)
        var rec = ShortcutRecording()
        let plain = Shortcut(keyCode: 0, modifiers: [.shift], keyLabel: "A")
        rec.press(plain, reservedID: nil, center: center)
        #expect(rec.phase == .rejected(plain, .needsModifier))
        #expect(registrar.registered == nil)
    }
}

@Suite struct HotKeyCenterTests {
    @Test func loadsDefaultWhenNothingIsSaved() {
        let registrar = FakeRegistrar()
        let center = HotKeyCenter(registrar: registrar, settings: ShortcutSettings(defaults: freshDefaults()))
        #expect(center.shortcut == .defaultSummon)
        #expect(registrar.registered == .defaultSummon)
    }

    @Test func reportsWhenTheSavedShortcutCannotBeRegistered() {
        let registrar = FakeRegistrar()
        registrar.takenByOtherApps = [Shortcut.defaultSummon.display]
        var events: [HotKeyCenter.Event] = []
        let settings = ShortcutSettings(defaults: freshDefaults())
        let center = HotKeyCenter(registrar: registrar, settings: settings)
        center.onEvent = { events.append($0) }
        center.registerSaved()
        #expect(center.registrationFailed)
        #expect(events == [.registrationFailed(.defaultSummon)])
    }

    @Test func interceptorSwallowsThePress() {
        let center = HotKeyCenter(registrar: FakeRegistrar(), settings: ShortcutSettings(defaults: freshDefaults()))
        var pressed = 0
        center.onPress = { pressed += 1 }
        center.fire()
        center.interceptor = { true }
        center.fire()
        center.interceptor = nil
        center.fire()
        #expect(pressed == 2)
    }
}
