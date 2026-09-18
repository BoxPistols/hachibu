import AppKit
import SlashstripCore

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var source: DataSource!
    private var strip: StripController!
    private var hotKeys: HotKeyCenter!
    private var recorder: ShortcutRecorder!
    private var builder: MenuBuilder!
    private var statusItem: StatusItemController!

    func applicationDidFinishLaunching(_ notification: Notification) {
        // 開発用: SLASHSTRIP_SOURCE=statuslineならスクリプトがあってもstatusLineのみ、demoなら架空の値で動かす
        let forced = ProcessInfo.processInfo.environment["SLASHSTRIP_SOURCE"]
        let source: DataSource
        switch forced {
        case "demo": source = DemoSource()
        case "statusline": source = StatusLineSource()
        default: source = ScriptBackend.detect() ?? StatusLineSource()
        }
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
        builder = MenuBuilder(source: source, strip: strip, hotKeys: hotKeys, recorder: recorder)
        statusItem = StatusItemController(source: source, strip: strip, builder: builder)

        strip.onPerform = { [weak source] slot in source?.perform(slot) }
        strip.onStatusClick = { [weak self] in
            guard let self else { return }
            var picked = false
            let menu = self.builder.sessionMenu { picked = true }
            self.strip.popUp(menu) { picked }
        }
        strip.onChange = { [weak self] in self?.statusItem.refreshTitle() }
        strip.usageText = { [weak source] in
            MenuBarStyle.title(style: .usage, statusText: nil, limits: source?.limits ?? [])
        }
        strip.contextMenu = { [weak self] in self?.builder.contextMenu() }

        source.onSlots = { [weak strip] slots in strip?.update(slots) }
        source.start()
    }
}
