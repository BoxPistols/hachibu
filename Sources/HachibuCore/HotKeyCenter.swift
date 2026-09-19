import Foundation

/// どのアプリが前面でも効くショートカットを実際に登録する部分（アプリではCarbon）。テストでは偽物に差し替える
public protocol HotKeyRegistrar: AnyObject {
    /// 登録できたらtrue。呼ぶ側が先にunregister()している
    func register(_ shortcut: Shortcut) -> Bool
    func unregister()
}

/// 設定の保存先。アプリではUserDefaults、テストではメモリ上の辞書（ディスクに書かない）
public protocol SettingsStore: AnyObject {
    func bool(forKey key: String) -> Bool
    func data(forKey key: String) -> Data?
    func set(_ value: Any?, forKey key: String)
}

extension UserDefaults: SettingsStore {}

/// 呼び出しショートカットの保存先
public final class ShortcutSettings {
    public static let shortcutKey = "summonShortcut"
    public static let disabledKey = "summonShortcutDisabled"

    private let defaults: SettingsStore

    public init(defaults: SettingsStore) {
        self.defaults = defaults
    }

    /// 無効にしていればnil。一度も設定していなければ既定（⌥⌘/）
    public func load() -> Shortcut? {
        if defaults.bool(forKey: Self.disabledKey) { return nil }
        guard let data = defaults.data(forKey: Self.shortcutKey),
              let saved = try? JSONDecoder().decode(Shortcut.self, from: data) else {
            return .defaultSummon
        }
        return saved
    }

    public func save(_ shortcut: Shortcut) {
        defaults.set(try? JSONEncoder().encode(shortcut), forKey: Self.shortcutKey)
        defaults.set(false, forKey: Self.disabledKey)
    }

    public func saveDisabled() {
        defaults.set(true, forKey: Self.disabledKey)
    }
}

/// 呼び出しショートカットの登録と保存。
///
/// 「登録」と「保存」を分けてある。記録の窓では`tryOut`で登録だけを試し、届くことを確かめてから`commit`で保存する。
/// 保存済みの値は確かめるまで変わらないので、取り消すときは`registerSaved`で登録し直すだけで窓を開く前に戻る。
public final class HotKeyCenter {
    public enum Event: Equatable {
        case committed(Shortcut)
        case disabled
        case registrationFailed(Shortcut)
    }

    /// 保存済みの組み合わせ。無効にしていればnil
    public private(set) var shortcut: Shortcut?
    /// 保存済みの組み合わせを登録できなかった（他のアプリが使用中）
    public private(set) var registrationFailed = false
    /// 押されたとき（帯の呼び出し）
    public var onPress: () -> Void = {}
    /// 記録の窓を開いている間だけ使う。trueを返したらonPressを呼ばない
    public var interceptor: (() -> Bool)?
    public var onEvent: (Event) -> Void = { _ in }

    private let registrar: HotKeyRegistrar
    private let settings: ShortcutSettings

    public init(registrar: HotKeyRegistrar, settings: ShortcutSettings) {
        self.registrar = registrar
        self.settings = settings
        shortcut = settings.load()
        registerSaved()
    }

    /// 登録したショートカットが押されたとき、登録器から呼ぶ
    public func fire() {
        if interceptor?() == true { return }
        onPress()
    }

    /// 保存はせずに登録だけ試す。成功している間は、保存済みの組み合わせの代わりにこれが効く
    public func tryOut(_ candidate: Shortcut) -> Bool {
        registrar.unregister()
        return registrar.register(candidate)
    }

    /// 届くことを確かめた組み合わせを保存する。tryOutで登録したものがそのまま効き続ける
    public func commit(_ confirmed: Shortcut) {
        shortcut = confirmed
        registrationFailed = false
        settings.save(confirmed)
        onEvent(.committed(confirmed))
    }

    public func disable() {
        registrar.unregister()
        shortcut = nil
        registrationFailed = false
        settings.saveDisabled()
        onEvent(.disabled)
    }

    /// 記録の窓を開いている間は外す（今の組み合わせを押しても呼び出しが走らず、記録に回るように）
    public func suspend() {
        registrar.unregister()
    }

    /// 保存済みの組み合わせを登録し直す。試していた組み合わせはこれで外れる
    public func registerSaved() {
        registrar.unregister()
        guard let saved = shortcut else {
            registrationFailed = false
            return
        }
        registrationFailed = !registrar.register(saved)
        if registrationFailed {
            onEvent(.registrationFailed(saved))
        }
    }
}
