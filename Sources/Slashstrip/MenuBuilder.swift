import AppKit
import SlashstripCore

/// メニューバーの項目と、帯の右クリックで共通に使うメニュー項目
final class MenuBuilder {
    private let source: DataSource
    private let strip: StripController
    private let hotKeys: HotKeyCenter
    private let recorder: ShortcutRecorder

    init(source: DataSource, strip: StripController, hotKeys: HotKeyCenter, recorder: ShortcutRecorder) {
        self.source = source
        self.strip = strip
        self.hotKeys = hotKeys
        self.recorder = recorder
    }

    /// 帯の右クリック。表示の切り替えをその場で行えるよう、モードは入れ子にせず並べる
    func contextMenu() -> NSMenu {
        let menu = NSMenu()
        usageItems().forEach(menu.addItem)
        menu.addItem(.separator())
        layoutItems().forEach(menu.addItem)
        menu.addItem(opacityMenuItem())
        menu.addItem(thresholdsMenuItem())
        menu.addItem(.separator())
        shortcutItems().forEach(menu.addItem)
        menu.addItem(resetPositionItem())
        return menu
    }

    /// 状態ボタンを押したときの移動先。開始順の番号で並べ、いま前面のものに印を付ける
    /// - onPick: 項目が選ばれたとき（移動の前に呼ぶ）
    func sessionMenu(onPick: @escaping () -> Void = {}) -> NSMenu {
        let menu = NSMenu()
        let (list, focusedID) = source.sessions()
        guard !list.isEmpty else {
            menu.addItem(disabled(L10n.sessionsNone))
            return menu
        }
        menu.addItem(disabled(L10n.sessionsHeader))
        for (i, session) in list.enumerated() {
            let item = ClosureMenuItem(title: session.menuTitle(number: i + 1)) { [weak self] in
                onPick()
                self?.source.focus(session)
            }
            item.image = Self.dot(for: session.state)
            item.state = session.id == focusedID ? .on : .off
            menu.addItem(item)
        }
        return menu
    }

    /// 使用率の行。黄や赤の段階にある枠には、その色の点を付ける
    func usageItems() -> [NSMenuItem] {
        let limits = source.limits
        guard !limits.isEmpty else { return [disabled(L10n.menuNoUsage)] }
        return limits.map { limit in
            let item = disabled(limit.line)
            if let color = LevelStyle.background(strip.thresholds.level(limit.percent)) {
                item.image = Self.dot(color.nsColor)
            }
            return item
        }
    }

    /// 黄と赤の閾値。赤が黄以下になる選択肢は選べなくする
    func thresholdsMenuItem() -> NSMenuItem {
        let current = strip.thresholds
        var items: [NSMenuItem] = [disabled(L10n.thresholdWarningHeader)]
        for p in UsageThresholds.warningChoices {
            let next = UsageThresholds(warning: p, critical: current.critical)
            let item = choice(L10n.thresholdChoice(p), selected: current.warning == p) { [weak self] in
                if let next { self?.strip.setThresholds(next) }
            }
            item.isEnabled = next != nil
            item.indentationLevel = 1
            items.append(item)
        }
        items.append(.separator())
        items.append(disabled(L10n.thresholdCriticalHeader))
        for p in UsageThresholds.criticalChoices {
            let next = UsageThresholds(warning: current.warning, critical: p)
            let item = choice(L10n.thresholdChoice(p), selected: current.critical == p) { [weak self] in
                if let next { self?.strip.setThresholds(next) }
            }
            item.isEnabled = next != nil
            item.indentationLevel = 1
            items.append(item)
        }
        return submenu(L10n.menuThresholds, items)
    }

    func sourceItem() -> NSMenuItem {
        disabled(L10n.menuSourcePrefix + source.sourceDescription)
    }

    func layoutItems() -> [NSMenuItem] {
        StripLayout.allCases.map { layout in
            choice(L10n.layoutName(layout), selected: strip.restLayout == layout) { [weak self] in
                self?.strip.setRestLayout(layout)
            }
        }
    }

    func layoutMenuItem() -> NSMenuItem {
        submenu(L10n.menuLayout, layoutItems())
    }

    func opacityMenuItem() -> NSMenuItem {
        submenu(L10n.menuOpacity, Prefs.opacityChoices.map { value in
            choice(L10n.opacityName(value), selected: strip.opacity == value) { [weak self] in
                self?.strip.setOpacity(value)
            }
        })
    }

    func shortcutItems() -> [NSMenuItem] {
        let summon = choice(L10n.menuSummon, selected: strip.summoned) { [weak self] in self?.strip.toggleSummon() }
        var items = [summon]
        if let shortcut = hotKeys.shortcut {
            if hotKeys.registrationFailed {
                items.append(disabled(L10n.menuHotKeyUnavailable(shortcut.display)))
            } else if shortcut.keyLabel.count == 1 {
                // 表示のため。実際に効くのはHotKeyの登録で、メニューのキー割り当てはメニューを開いている間しか効かない
                summon.keyEquivalent = shortcut.keyLabel.lowercased()
                summon.keyEquivalentModifierMask = Self.flags(shortcut.modifiers)
            } else {
                summon.title = "\(L10n.menuSummon)（\(shortcut.display)）"
            }
        } else {
            items.append(disabled(L10n.menuShortcutOff))
        }
        items.append(choice(L10n.menuChangeShortcut, selected: false) { [weak self] in self?.recorder.show() })
        return items
    }

    func resetPositionItem() -> NSMenuItem {
        choice(L10n.menuResetPosition, selected: false) { [weak self] in self?.strip.resetPosition() }
    }

    func openLogItem() -> NSMenuItem {
        choice(L10n.menuOpenLog, selected: false) {
            ActionLog.append("操作ログを開きました")
            // 追記は非同期なので、ファイルができるのを待たずに開くと「見つからない」になりうる
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                NSWorkspace.shared.open(ActionLog.url)
            }
        }
    }

    // MARK: - 部品

    func disabled(_ title: String) -> NSMenuItem {
        let item = NSMenuItem(title: title, action: nil, keyEquivalent: "")
        item.isEnabled = false
        return item
    }

    func submenu(_ title: String, _ items: [NSMenuItem]) -> NSMenuItem {
        let parent = NSMenuItem(title: title, action: nil, keyEquivalent: "")
        let sub = NSMenu(title: title)
        items.forEach(sub.addItem)
        parent.submenu = sub
        return parent
    }

    func choice(_ title: String, selected: Bool, _ action: @escaping () -> Void) -> NSMenuItem {
        let item = ClosureMenuItem(title: title, handler: action)
        item.state = selected ? .on : .off
        return item
    }

    private static func flags(_ m: Shortcut.Modifiers) -> NSEvent.ModifierFlags {
        var f: NSEvent.ModifierFlags = []
        if m.contains(.control) { f.insert(.control) }
        if m.contains(.option) { f.insert(.option) }
        if m.contains(.shift) { f.insert(.shift) }
        if m.contains(.command) { f.insert(.command) }
        return f
    }

    /// 状態の色の点。帯の状態ボタンと同じ色分け（青=実行中、橙=許可待ち、灰=それ以外）
    private static func dot(for state: SessionInfo.State) -> NSImage {
        switch state {
        case .busy: return dot(NSColor(srgbRed: 38 / 255, green: 102 / 255, blue: 168 / 255, alpha: 1))
        case .waitingPermission: return dot(NSColor(srgbRed: 214 / 255, green: 138 / 255, blue: 30 / 255, alpha: 1))
        case .waitingInput, .idle: return dot(NSColor(srgbRed: 120 / 255, green: 120 / 255, blue: 125 / 255, alpha: 1))
        }
    }

    private static func dot(_ color: NSColor) -> NSImage {
        NSImage(size: NSSize(width: 10, height: 10), flipped: false) { rect in
            color.setFill()
            NSBezierPath(ovalIn: rect.insetBy(dx: 1, dy: 1)).fill()
            return true
        }
    }
}

/// 押したときにクロージャを呼ぶメニュー項目
final class ClosureMenuItem: NSMenuItem {
    private let handler: () -> Void

    init(title: String, handler: @escaping () -> Void) {
        self.handler = handler
        super.init(title: title, action: #selector(fire), keyEquivalent: "")
        target = self
    }

    required init(coder: NSCoder) { fatalError("not used") }

    @objc private func fire() { handler() }
}
