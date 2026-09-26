import AppKit
import HachibuCore

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var source: DataSource!
    private var strip: StripController!
    private var hotKeys: HotKeyCenter!
    private var recorder: ShortcutRecorder!
    private var cycleKeys: HotKeyCenter!
    private var cycleRecorder: ShortcutRecorder!
    private var builder: MenuBuilder!
    private var statusItem: StatusItemController?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // 同じアプリの別のコピーがすでに動いていれば、そちらに帯を出させて終わる（帯を重ねて出さない）
        if SingleInstance.handOff() {
            NSApp.terminate(nil)
            return
        }
        // 文言を読む部品を作る前に、メニューで選んだ言語にしておく
        L10n.apply(saved: Prefs.language)
        // 開発用: HACHIBU_SOURCE=demoのときは架空の値で動かす（撮影用）
        let isDemo = ProcessInfo.processInfo.environment["HACHIBU_SOURCE"] == "demo"
        let source: DataSource = isDemo ? DemoSource() : BasicSource()
        let strip = StripController()
        self.source = source
        self.strip = strip

        let registrar = CarbonRegistrar(id: 1)
        let hotKeys = HotKeyCenter(registrar: registrar, settings: ShortcutSettings(defaults: Prefs.defaults))
        registrar.onPress = { [weak hotKeys] in hotKeys?.fire() }
        hotKeys.onPress = { [weak strip] in strip?.toggleSummon() }
        let logShortcut: (HotKeyCenter.Event) -> Void = { event in
            switch event {
            case .committed(let s): ActionLog.append(L10n.current.logShortcutCommitted(s.display))
            case .disabled: ActionLog.append(L10n.current.logShortcutDisabled)
            case .registrationFailed(let s): ActionLog.append(L10n.current.logShortcutFailed(s.display))
            }
        }
        hotKeys.onEvent = logShortcut
        self.hotKeys = hotKeys

        // 表示を順に切り替えるショートカット。利用者が設定するまでは何も登録しない
        let cycleRegistrar = CarbonRegistrar(id: 2)
        let cycleKeys = HotKeyCenter(registrar: cycleRegistrar, settings: .layoutCycle(defaults: Prefs.defaults))
        cycleRegistrar.onPress = { [weak cycleKeys] in cycleKeys?.fire() }
        cycleKeys.onPress = { [weak strip] in strip?.cycleLayout() }
        cycleKeys.onEvent = logShortcut
        self.cycleKeys = cycleKeys

        recorder = ShortcutRecorder(hotKeys: hotKeys, usedElsewhere: { [weak cycleKeys] in cycleKeys?.shortcut })
        cycleRecorder = ShortcutRecorder(hotKeys: cycleKeys, title: { L10n.current.recorderCycleTitle },
                                         usedElsewhere: { [weak hotKeys] in hotKeys?.shortcut })
        builder = MenuBuilder(source: source, strip: strip, hotKeys: hotKeys, recorder: recorder)
        builder.cycleKeys = cycleKeys
        builder.cycleRecorder = cycleRecorder
        // 撮影用の起動ではメニューバーの項目を作らない（macOSの「メニューバー」の設定の一覧に行を増やさないため）
        if !isDemo {
            statusItem = StatusItemController(source: source, strip: strip, builder: builder)
        }
        builder.menuBarItemIsHidden = { [weak self] in self?.statusItem?.isHiddenBySystem ?? false }

        strip.onChange = { [weak self] in self?.statusItem?.refreshTitle() }
        strip.usageText = { [weak source] in
            MenuBarStyle.title(style: .usage, statusText: nil, limits: source?.limits ?? [])
        }
        strip.contextMenu = { [weak self] in self?.builder.contextMenu() }
        SingleInstance.observe { [weak strip] in strip?.showSummoned() }

        source.onSlots = { [weak strip] slots in strip?.update(slots) }
        source.start()
    }

    /// メニューバーの項目をmacOSの設定で隠していても、Finderなどからもう一度開けば帯を呼び出せるようにする
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        strip.showSummoned()
        return false
    }
}
