import AppKit
import SlashstripCore

/// メニューバーの項目と、帯の右クリックで共通に使うメニュー項目
final class MenuBuilder {
    private let source: DataSource
    private let strip: StripController
    private let hotKeys: HotKeyCenter
    private let recorder: ShortcutRecorder
    private let consent: UsageAPIConsent

    init(source: DataSource, strip: StripController, hotKeys: HotKeyCenter, recorder: ShortcutRecorder,
         consent: UsageAPIConsent) {
        self.source = source
        self.strip = strip
        self.hotKeys = hotKeys
        self.recorder = recorder
        self.consent = consent
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

    /// 使用率の行。黄や赤の段階にある枠には、その色の点を付ける
    func usageItems() -> [NSMenuItem] {
        let limits = source.limits
        guard !limits.isEmpty else { return [disabled(L10n.current.menuNoUsage)] }
        var items = limits.map { limit in
            let item = disabled(limit.line())
            if let color = LevelStyle.background(strip.thresholds.level(limit.percent)) {
                item.image = Self.dot(color.nsColor)
            }
            return item
        }
        // ターミナルでClaude Codeを使っていない間は値が更新されない。いつ時点の値かを添える
        if Usage.isStale(source.usageAsOf), let asOf = source.usageAsOf {
            items.append(disabled(L10n.current.usageAsOf(ResetFormatter.text(asOf))))
        }
        return items
    }

    /// 黄と赤の閾値。赤が黄以下になる選択肢は選べなくする
    func thresholdsMenuItem() -> NSMenuItem {
        let current = strip.thresholds
        let warning = UsageThresholds.warningChoices.map { p in
            thresholdChoice(p, selected: current.warning == p,
                            next: UsageThresholds(warning: p, critical: current.critical))
        }
        let critical = UsageThresholds.criticalChoices.map { p in
            thresholdChoice(p, selected: current.critical == p,
                            next: UsageThresholds(warning: current.warning, critical: p))
        }
        let items = [disabled(L10n.current.thresholdWarningHeader)] + warning + [.separator()]
            + [disabled(L10n.current.thresholdCriticalHeader)] + critical
        return submenu(L10n.current.menuThresholds, items)
    }

    /// 撮影用の架空の値では出さない。有効にするときは、何を使いどんなリスクがあるかを先に示す
    func usageAPIItems() -> [NSMenuItem] {
        guard let basic = source as? BasicSource else { return [] }
        let item = choice(L10n.current.menuUsageAPI, selected: Prefs.usageAPIConsent == true) { [weak self] in
            if Prefs.usageAPIConsent == true {
                Prefs.usageAPIConsent = false
                ActionLog.append("使用率APIからの取得をやめました")
                basic.usageAPISettingChanged()
            } else {
                self?.consent.show { enabled in
                    Prefs.usageAPIConsent = enabled
                    ActionLog.append(enabled ? "使用率APIからの取得を有効にしました" : "使用率APIからの取得は有効にしませんでした")
                    basic.usageAPISettingChanged()
                }
            }
        }
        var items = [item]
        if Prefs.usageAPIConsent == true, let failure = basic.apiFailure {
            items.append(disabled(L10n.current.usageAPIFailed(failure.summary)))
        }
        return items
    }

    /// 項目に乗せると帯をそのモードで仮に表示し、選ぶと確定する
    func layoutItems() -> [NSMenuItem] {
        StripLayout.allCases.map { layout in
            previewChoice(L10n.current.layoutName(layout), selected: strip.restLayout == layout,
                          preview: { [weak self] in self?.strip.preview(layout: layout) },
                          commit: { [weak self] in self?.strip.setRestLayout(layout) })
        }
    }

    func layoutMenuItem() -> NSMenuItem {
        submenu(L10n.current.menuLayout, layoutItems())
    }

    func opacityMenuItem() -> NSMenuItem {
        submenu(L10n.current.menuOpacity, Prefs.opacityChoices.map { value in
            previewChoice(L10n.opacityName(value), selected: strip.opacity == value,
                          preview: { [weak self] in self?.strip.preview(opacity: value) },
                          commit: { [weak self] in self?.strip.setOpacity(value) })
        })
    }

    func shortcutItems() -> [NSMenuItem] {
        let summon = choice(L10n.current.menuSummon, selected: strip.summoned) { [weak self] in self?.strip.toggleSummon() }
        var items = [summon]
        if let shortcut = hotKeys.shortcut {
            if hotKeys.registrationFailed {
                items.append(disabled(L10n.current.menuHotKeyUnavailable(shortcut.display)))
            } else if shortcut.keyLabel.count == 1 {
                // 表示のため。実際に効くのはHotKeyの登録で、メニューのキー割り当てはメニューを開いている間しか効かない
                summon.keyEquivalent = shortcut.keyLabel.lowercased()
                summon.keyEquivalentModifierMask = Self.flags(shortcut.modifiers)
            } else {
                summon.title = "\(L10n.current.menuSummon)（\(shortcut.display)）"
            }
        } else {
            items.append(disabled(L10n.current.menuShortcutOff))
        }
        items.append(choice(L10n.current.menuChangeShortcut, selected: false) { [weak self] in self?.recorder.show() })
        return items
    }

    /// ログイン時の自動起動。失敗の理由は次にメニューを開いたときに出す
    private var loginItemError: String?

    func loginItems() -> [NSMenuItem] {
        let toggle = choice(L10n.current.menuLaunchAtLogin, selected: LoginItem.isEnabled || LoginItem.needsApproval) {
            [weak self] in
            let turnOn = !(LoginItem.isEnabled || LoginItem.needsApproval)
            self?.loginItemError = LoginItem.set(turnOn)
            if let error = self?.loginItemError {
                ActionLog.append("ログイン項目を変更できませんでした: \(error)")
            } else {
                ActionLog.append(turnOn ? "ログイン時に起動するようにしました" : "ログイン時の起動をやめました")
            }
        }
        var items = [toggle]
        if LoginItem.needsApproval {
            items.append(choice(L10n.current.loginItemNeedsApproval, selected: false) { LoginItem.openSettings() })
        }
        if let loginItemError {
            items.append(disabled(L10n.current.loginItemFailed(loginItemError)))
        }
        return items
    }

    func resetPositionItem() -> NSMenuItem {
        choice(L10n.current.menuResetPosition, selected: false) { [weak self] in self?.strip.resetPosition() }
    }

    func openLogItem() -> NSMenuItem {
        choice(L10n.current.menuOpenLog, selected: false) {
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
        // 入れ子の中の項目に乗せたときも仮の表示を出す
        sub.delegate = strip.menuWatcher
        items.forEach(sub.addItem)
        parent.submenu = sub
        return parent
    }

    func previewChoice(_ title: String, selected: Bool,
                       preview: @escaping () -> Void, commit: @escaping () -> Void) -> NSMenuItem {
        let item = PreviewMenuItem(title: title, preview: preview, handler: commit)
        item.state = selected ? .on : .off
        return item
    }

    /// 閾値の1項目。見出しの下に字下げして並べる
    private func thresholdChoice(_ percent: Int, selected: Bool, next: UsageThresholds?) -> NSMenuItem {
        let item: NSMenuItem
        if let next {
            item = previewChoice(L10n.current.thresholdChoice(percent), selected: selected,
                                 preview: { [weak self] in self?.strip.preview(thresholds: next) },
                                 commit: { [weak self] in self?.strip.setThresholds(next) })
        } else {
            item = disabled(L10n.current.thresholdChoice(percent))
            item.state = selected ? .on : .off
        }
        item.indentationLevel = 1
        return item
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

    private static func dot(_ color: NSColor) -> NSImage {
        NSImage(size: NSSize(width: 10, height: 10), flipped: false) { rect in
            color.setFill()
            NSBezierPath(ovalIn: rect.insetBy(dx: 1, dy: 1)).fill()
            return true
        }
    }
}

/// 押したときにクロージャを呼ぶメニュー項目
class ClosureMenuItem: NSMenuItem {
    private let handler: () -> Void

    init(title: String, handler: @escaping () -> Void) {
        self.handler = handler
        super.init(title: title, action: #selector(fire), keyEquivalent: "")
        target = self
    }

    required init(coder: NSCoder) { fatalError("not used") }

    @objc private func fire() { handler() }
}

/// 乗せたときに仮の表示を出し、押したときに確定するメニュー項目
final class PreviewMenuItem: ClosureMenuItem {
    let preview: () -> Void

    init(title: String, preview: @escaping () -> Void, handler: @escaping () -> Void) {
        self.preview = preview
        super.init(title: title, handler: handler)
    }

    required init(coder: NSCoder) { fatalError("not used") }
}
