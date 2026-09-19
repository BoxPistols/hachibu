import AppKit
import HachibuCore

/// メニューバーの項目。アイコンの横に使用率や状態を文字で出せる。
/// ツールチップはアプリが非アクティブだと出ない場合があるので、リセット時刻はここでも読めるようにする。
final class StatusItemController: NSObject, NSMenuDelegate {
    private let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    private let menu = NSMenu()
    private let source: DataSource
    private let strip: StripController
    private let builder: MenuBuilder
    private var style = Prefs.menuBarStyle

    init(source: DataSource, strip: StripController, builder: MenuBuilder) {
        self.source = source
        self.strip = strip
        self.builder = builder
        super.init()
        if let button = item.button {
            button.imagePosition = .imageOnly
        }
        menu.delegate = self
        item.menu = menu
        refreshTitle()
    }

    /// macOSの「メニューバー」の設定で許可が切られていると、項目はどの画面にも属さない位置に置かれる。
    /// 画面の下端に数ptだけ掛かるので、枠の重なりでは判定できない
    var isHiddenBySystem: Bool {
        guard let window = item.button?.window else { return false }
        return window.screen == nil
    }

    /// 帯の中身が変わるたびに呼ぶ
    func refreshTitle() {
        guard let button = item.button else { return }
        let text = MenuBarStyle.title(style: style, statusText: strip.statusSlot?.text, limits: source.limits)
        // 帯と同じく、黄や赤の段階にある使用率の語だけ色の札で囲む。絵と文字は1枚の画像にまとめて描く
        let segments = text.map { UsageMarkup.segments($0, thresholds: strip.thresholds) } ?? []
        if style == .gauge, let reading = GaugeReading.from(source.limits) {
            button.image = MenuBarIcon.gaugeImage(label: reading.label, percent: reading.percent,
                                                  level: strip.thresholds.level(reading.percent))
            button.image?.accessibilityDescription = "\(L10n.appName) \(reading.label)"
        } else {
            button.image = MenuBarIcon.image(segments: segments)
            button.image?.accessibilityDescription = [L10n.appName, text].compactMap { $0 }.joined(separator: " ")
        }
        button.attributedTitle = NSAttributedString(string: "")
    }

    func menuNeedsUpdate(_ menu: NSMenu) {
        menu.removeAllItems()
        builder.usageItems().forEach(menu.addItem)
        builder.usageAPIItems().forEach(menu.addItem)
        menu.addItem(.separator())

        menu.addItem(builder.layoutMenuItem())
        menu.addItem(builder.opacityMenuItem())
        menu.addItem(builder.thresholdsMenuItem())
        menu.addItem(builder.submenu(L10n.current.menuMenuBar, MenuBarStyle.allCases.map { style in
            builder.choice(L10n.current.menuBarStyleName(style), selected: self.style == style) { [weak self] in
                self?.style = style
                Prefs.menuBarStyle = style
                self?.refreshTitle()
            }
        }))
        menu.addItem(.separator())

        builder.shortcutItems().forEach(menu.addItem)
        menu.addItem(builder.languageMenuItem())
        builder.loginItems().forEach(menu.addItem)
        menu.addItem(builder.resetPositionItem())
        menu.addItem(builder.openLogItem())
        menu.addItem(builder.legendMenuItem())

        menu.addItem(.separator())
        menu.addItem(NSMenuItem(title: L10n.current.menuQuit, action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
    }
}
