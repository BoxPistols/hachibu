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
            button.image = NSImage(systemSymbolName: "slash.circle", accessibilityDescription: L10n.appName)
            button.image?.isTemplate = true
            button.imagePosition = .imageLeading
        }
        menu.delegate = self
        item.menu = menu
        refreshTitle()
    }

    /// 帯の中身が変わるたびに呼ぶ
    func refreshTitle() {
        guard let button = item.button else { return }
        let text = MenuBarStyle.title(style: style, statusText: strip.statusSlot?.text, limits: source.limits)
        if let text {
            // 数字の幅を揃え、値が変わってもアイコンの位置が揺れないようにする
            let font = NSFont.monospacedDigitSystemFont(ofSize: NSFont.systemFontSize, weight: .regular)
            let title = NSMutableAttributedString(string: " ", attributes: [.font: font])
            // 帯と同じく、黄や赤の段階にある使用率の語だけ色の札で囲む
            for segment in UsageMarkup.segments(text, thresholds: strip.thresholds) {
                var attrs: [NSAttributedString.Key: Any] = [.font: font]
                if let bg = LevelStyle.background(segment.level) {
                    attrs[.backgroundColor] = bg.nsColor
                    attrs[.foregroundColor] = LevelStyle.foreground(segment.level).nsColor
                }
                title.append(NSAttributedString(string: segment.text, attributes: attrs))
            }
            button.attributedTitle = title
        } else {
            button.attributedTitle = NSAttributedString(string: "")
        }
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

        menu.addItem(.separator())
        menu.addItem(NSMenuItem(title: L10n.current.menuQuit, action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
    }
}
