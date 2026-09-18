import Carbon
import Foundation
import SlashstripCore

/// どのアプリが前面でも効くショートカット。CarbonのRegisterEventHotKeyを使うので、アクセシビリティ権限は要らない。
final class HotKey {
    private var hotKeyRef: EventHotKeyRef?
    private var handlerRef: EventHandlerRef?
    private let action: () -> Void

    /// 登録に失敗したらnil（他のアプリが同じ組み合わせを押さえている場合など）
    init?(_ shortcut: Shortcut, action: @escaping () -> Void) {
        self.action = action
        var spec = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        let me = Unmanaged.passUnretained(self).toOpaque()
        let installed = InstallEventHandler(GetApplicationEventTarget(), { _, _, userData in
            guard let userData else { return OSStatus(eventNotHandledErr) }
            let hotKey = Unmanaged<HotKey>.fromOpaque(userData).takeUnretainedValue()
            DispatchQueue.main.async { hotKey.action() }
            return noErr
        }, 1, &spec, me, &handlerRef)
        guard installed == noErr else { return nil }

        // 'SLST'
        let id = EventHotKeyID(signature: OSType(0x534C_5354), id: 1)
        let registered = RegisterEventHotKey(shortcut.keyCode, shortcut.carbonModifiers, id,
                                             GetApplicationEventTarget(), 0, &hotKeyRef)
        guard registered == noErr else {
            if let handlerRef { RemoveEventHandler(handlerRef) }
            return nil
        }
    }

    deinit {
        if let hotKeyRef { UnregisterEventHotKey(hotKeyRef) }
        if let handlerRef { RemoveEventHandler(handlerRef) }
    }
}

/// 呼び出しショートカットの保存と登録。変更に失敗したら元の組み合わせに戻す
final class HotKeyCenter {
    private static let shortcutKey = "summonShortcut"
    private static let disabledKey = "summonShortcutDisabled"

    private var hotKey: HotKey?
    private let action: () -> Void
    /// いま設定されている組み合わせ。無効にしていればnil
    private(set) var shortcut: Shortcut?
    /// 設定はあるが登録できなかった（他のアプリが使用中）
    private(set) var registrationFailed = false

    init(action: @escaping () -> Void) {
        self.action = action
        if Prefs.defaults.bool(forKey: Self.disabledKey) {
            shortcut = nil
        } else if let data = Prefs.defaults.data(forKey: Self.shortcutKey),
                  let saved = try? JSONDecoder().decode(Shortcut.self, from: data) {
            shortcut = saved
        } else {
            shortcut = .defaultSummon
        }
        registerCurrent()
    }

    /// 新しい組み合わせにする。登録できなければ元に戻してfalse
    func change(to new: Shortcut) -> Bool {
        let old = shortcut
        hotKey = nil
        guard let registered = HotKey(new, action: action) else {
            shortcut = old
            registerCurrent()
            return false
        }
        hotKey = registered
        shortcut = new
        registrationFailed = false
        Prefs.defaults.set(try? JSONEncoder().encode(new), forKey: Self.shortcutKey)
        Prefs.defaults.set(false, forKey: Self.disabledKey)
        ActionLog.append("ショートカットを\(new.display)に変更しました")
        return true
    }

    func disable() {
        hotKey = nil
        shortcut = nil
        registrationFailed = false
        Prefs.defaults.set(true, forKey: Self.disabledKey)
        ActionLog.append("ショートカットを無効にしました")
    }

    /// 記録中は外す（今の組み合わせを押しても呼び出しが走らず、記録に回るように）
    func suspend() {
        hotKey = nil
    }

    func resume() {
        registerCurrent()
    }

    private func registerCurrent() {
        guard let shortcut else {
            hotKey = nil
            return
        }
        hotKey = HotKey(shortcut, action: action)
        registrationFailed = hotKey == nil
        if registrationFailed {
            ActionLog.append("ショートカット\(shortcut.display)の登録に失敗しました")
        }
    }
}
