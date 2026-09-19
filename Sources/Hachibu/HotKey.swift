import Carbon
import Foundation
import HachibuCore

/// どのアプリが前面でも効くショートカット。CarbonのRegisterEventHotKeyを使うので、アクセシビリティ権限は要らない。
final class HotKey {
    private var hotKeyRef: EventHotKeyRef?
    private var handlerRef: EventHandlerRef?
    private let action: () -> Void
    private let id: UInt32

    /// 登録に失敗したらnil（他のアプリが同じ組み合わせを押さえている場合など）
    /// - id: このアプリの中での番号。押下の通知は登録したすべての受け手に届くので、番号で自分のものだけを選ぶ
    init?(_ shortcut: Shortcut, id: UInt32, action: @escaping () -> Void) {
        self.action = action
        self.id = id
        var spec = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
        let me = Unmanaged.passUnretained(self).toOpaque()
        let installed = InstallEventHandler(GetApplicationEventTarget(), { _, event, userData in
            guard let userData, let event else { return OSStatus(eventNotHandledErr) }
            let hotKey = Unmanaged<HotKey>.fromOpaque(userData).takeUnretainedValue()
            var pressed = EventHotKeyID()
            let status = GetEventParameter(event, EventParamName(kEventParamDirectObject), EventParamType(typeEventHotKeyID),
                                           nil, MemoryLayout<EventHotKeyID>.size, nil, &pressed)
            guard status == noErr, pressed.id == hotKey.id else { return OSStatus(eventNotHandledErr) }
            DispatchQueue.main.async { hotKey.action() }
            return noErr
        }, 1, &spec, me, &handlerRef)
        guard installed == noErr else { return nil }

        // 'SLST'
        let hotKeyID = EventHotKeyID(signature: OSType(0x534C_5354), id: id)
        let registered = RegisterEventHotKey(shortcut.keyCode, shortcut.carbonModifiers, hotKeyID,
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

/// HotKeyCenterから使う登録器。1つの登録器が同時に登録するのは1つだけで、操作ごとに番号の違う登録器を持つ
final class CarbonRegistrar: HotKeyRegistrar {
    var onPress: () -> Void = {}
    private var hotKey: HotKey?
    private let id: UInt32

    init(id: UInt32) {
        self.id = id
    }

    func register(_ shortcut: Shortcut) -> Bool {
        // 同じ番号の登録が残っていると二重になるので、先に外す
        hotKey = nil
        hotKey = HotKey(shortcut, id: id) { [weak self] in self?.onPress() }
        return hotKey != nil
    }

    func unregister() {
        hotKey = nil
    }
}
