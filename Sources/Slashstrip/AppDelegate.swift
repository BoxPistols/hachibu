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
        // 文言を読む部品を作る前に、メニューで選んだ言語にしておく
        L10n.apply(saved: Prefs.language)
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
            case .committed(let s): ActionLog.append(L10n.current.logShortcutCommitted(s.display))
            case .disabled: ActionLog.append(L10n.current.logShortcutDisabled)
            case .registrationFailed(let s): ActionLog.append(L10n.current.logShortcutFailed(s.display))
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
                    ActionLog.append(enabled ? L10n.current.logUsageAPIEnabled : L10n.current.logUsageAPIDeclined)
                    basic.usageAPISettingChanged()
                }
            }
        }
    }

    /// メニューバーの項目をmacOSの設定で隠していても、Finderなどからもう一度開けば帯を呼び出せるようにする
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        strip.showSummoned()
        return false
    }
}
