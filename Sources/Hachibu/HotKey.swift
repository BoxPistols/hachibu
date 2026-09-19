import Carbon
import Foundation
import HachibuCore

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

/// HotKeyCenterから使う登録器。同時に登録するのは1つだけ
final class CarbonRegistrar: HotKeyRegistrar {
    var onPress: () -> Void = {}
    private var hotKey: HotKey?

    func register(_ shortcut: Shortcut) -> Bool {
        hotKey = HotKey(shortcut) { [weak self] in self?.onPress() }
        return hotKey != nil
    }

    func unregister() {
        hotKey = nil
    }
}
