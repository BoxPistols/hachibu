import AppKit
import SlashstripCore

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var source: DataSource!
    private var strip: StripController!
    private var hotKeys: HotKeyCenter!
    private var recorder: ShortcutRecorder!
    private var builder: MenuBuilder!
    private let consent = UsageAPIConsent()
    private var statusItem: StatusItemController!

    func applicationDidFinishLaunching(_ notification: Notification) {
        // 開発用: SLASHSTRIP_SOURCE=demoのときは架空の値で動かす（撮影用）
        let isDemo = ProcessInfo.processInfo.environment["SLASHSTRIP_SOURCE"] == "demo"
        let source: DataSource = isDemo ? DemoSource() : BasicSource()
        let strip = StripController()
        self.source = source
        self.strip = strip

        let registrar = CarbonRegistrar()
        let hotKeys = HotKeyCenter(registrar: registrar, settings: ShortcutSettings(defaults: Prefs.defaults))
        registrar.onPress = { [weak hotKeys] in hotKeys?.fire() }
        hotKeys.onPress = { [weak strip] in strip?.toggleSummon() }
        hotKeys.onEvent = { event in
            switch event {
            case .committed(let s): ActionLog.append("ショートカットを\(s.display)にしました（届くことを確認済み）")
            case .disabled: ActionLog.append("ショートカットを無効にしました")
            case .registrationFailed(let s): ActionLog.append("ショートカット\(s.display)の登録に失敗しました")
            }
        }
        self.hotKeys = hotKeys
        recorder = ShortcutRecorder(hotKeys: hotKeys)
        builder = MenuBuilder(source: source, strip: strip, hotKeys: hotKeys, recorder: recorder, consent: consent)
        statusItem = StatusItemController(source: source, strip: strip, builder: builder)

        strip.onChange = { [weak self] in self?.statusItem.refreshTitle() }
        strip.usageText = { [weak source] in
            MenuBarStyle.title(style: .usage, statusText: nil, limits: source?.limits ?? [])
        }
        strip.contextMenu = { [weak self] in self?.builder.contextMenu() }

        source.onSlots = { [weak strip] slots in strip?.update(slots) }
        source.start()

        // 基本表示で、まだ尋ねていなければ、使用率APIを使うかを最初に一度だけ尋ねる
        if let basic = source as? BasicSource, Prefs.usageAPIConsent == nil {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.8) { [weak self] in
                self?.consent.show { enabled in
                    Prefs.usageAPIConsent = enabled
                    ActionLog.append(enabled ? "使用率APIからの取得を有効にしました" : "使用率APIからの取得は有効にしませんでした")
                    basic.usageAPISettingChanged()
                }
            }
        }
    }
}
