import AppKit
import SlashstripCore

/// 利用者の設定。UserDefaultsのキーはここにだけ書く
enum Prefs {
    private static let layoutKey = "restLayout"
    private static let opacityKey = "opacity"
    private static let menuBarKey = "menuBarStyle"
    private static let warningKey = "warningThreshold"
    private static let criticalKey = "criticalThreshold"

    static let opacityChoices: [Double] = [1.0, 0.85, 0.7, 0.5, 0.3]

    static var restLayout: StripLayout {
        get { StripLayout(rawValue: UserDefaults.standard.string(forKey: layoutKey) ?? "") ?? .full }
        set { UserDefaults.standard.set(newValue.rawValue, forKey: layoutKey) }
    }

    static var opacity: Double {
        get {
            let v = UserDefaults.standard.double(forKey: opacityKey)
            return opacityChoices.contains(v) ? v : 1.0
        }
        set { UserDefaults.standard.set(newValue, forKey: opacityKey) }
    }

    static var thresholds: UsageThresholds {
        get {
            let d = UserDefaults.standard
            return UsageThresholds(warning: d.integer(forKey: warningKey), critical: d.integer(forKey: criticalKey))
                ?? .default
        }
        set {
            UserDefaults.standard.set(newValue.warning, forKey: warningKey)
            UserDefaults.standard.set(newValue.critical, forKey: criticalKey)
        }
    }

    static var menuBarStyle: MenuBarStyle {
        get { MenuBarStyle(rawValue: UserDefaults.standard.string(forKey: menuBarKey) ?? "") ?? .icon }
        set { UserDefaults.standard.set(newValue.rawValue, forKey: menuBarKey) }
    }
}

/// 帯の表示状態を持つ。普段のモード（利用者が選ぶ）に、マウスを乗せたときとショートカットで呼んだときの一時的な展開を重ねる。
final class StripController {
    let store = StripStore()
    private let hosting: FirstClickHostingView<StripView>
    private let panel: StripPanel
    private let placement: StripPlacement
    private let tracker = HoverTracker()

    private var allSlots: [Slot] = []
    private var hovering = false
    private var hoverCheck: DispatchWorkItem?
    private(set) var summoned = false
    private(set) var restLayout = Prefs.restLayout
    private(set) var opacity = Prefs.opacity

    var onPerform: (Slot) -> Void = { _ in }
    /// 状態ボタンを押したとき（セッションの一覧を出す）
    var onStatusClick: () -> Void = {}
    /// 表示が変わったとき（メニューバーの文字を合わせるため）
    var onChange: () -> Void = {}
    /// 「使用率だけ」モードで状態の枠に出す文字
    var usageText: () -> String? = { nil }
    /// 帯の右クリックメニュー
    var contextMenu: () -> NSMenu? = { nil } {
        didSet { hosting.contextMenuProvider = { [weak self] in self?.openedMenu(self?.contextMenu()) } }
    }
    /// メニューを開いている間は、カーソルがメニューへ移っても帯を畳まない
    private var menuOpen = false
    private let menuWatcher = MenuWatcher()

    // 乗せてから広げるまでと、離してから畳むまでの待ち。通りがかりで開閉しないため
    private static let expandDelay: TimeInterval = 0.15
    private static let collapseDelay: TimeInterval = 0.6

    init() {
        hosting = FirstClickHostingView(rootView: StripView(store: store))
        hosting.sizingOptions = [.intrinsicContentSize]
        panel = StripPanel(contentView: hosting)
        placement = StripPlacement(panel: panel)
        store.onPerform = { [weak self] slot in self?.performed(slot) }
        tracker.onChange = { [weak self] _ in self?.scheduleHoverCheck() }
        menuWatcher.onChange = { [weak self] open in
            self?.menuOpen = open
            if !open { self?.scheduleHoverCheck() }
        }
        // アプリが非アクティブのままでも出入りを受け取る
        hosting.addTrackingArea(NSTrackingArea(rect: .zero,
                                               options: [.mouseEnteredAndExited, .activeAlways, .inVisibleRect],
                                               owner: tracker, userInfo: nil))
    }

    var statusSlot: Slot? { allSlots.first { $0.id == Widgets.status } }

    func update(_ slots: [Slot]) {
        allSlots = slots
        render()
    }

    func setRestLayout(_ layout: StripLayout) {
        restLayout = layout
        Prefs.restLayout = layout
        summoned = false
        render()
    }

    var thresholds: UsageThresholds { store.thresholds }

    func setThresholds(_ value: UsageThresholds) {
        Prefs.thresholds = value
        store.thresholds = value
        onChange()
    }

    func setOpacity(_ value: Double) {
        opacity = value
        Prefs.opacity = value
        applyOpacity()
    }

    /// ショートカットとメニューから。呼び出し中ならもう一度で元のモードへ戻す
    func toggleSummon() {
        summoned.toggle()
        render()
    }

    func resetPosition() {
        placement.reset()
    }

    /// 帯の上（押した位置）にメニューを出す。
    ///
    /// このアプリが前面にないと、プログラムから出したメニューは表示されなかった（2026-09-18、
    /// popUp(positioning:at:in:)とpopUpContextMenuのどちらでも）。出している間だけ前面に出し、
    /// 何も選ばずに閉じたら元のアプリへ戻す。選んだ項目は自分で行き先のアプリを前面に出す。
    /// - Returns: 項目が選ばれたかどうかを受け取るための印。項目の処理から`picked = true`にする
    func popUp(_ menu: NSMenu, picked: @escaping () -> Bool) {
        guard let window = hosting.window else { return }
        let point = hosting.convert(window.convertPoint(fromScreen: NSEvent.mouseLocation), from: nil)
        _ = openedMenu(menu)
        let previous = NSWorkspace.shared.frontmostApplication
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            NSApp.activate()
            menu.popUp(positioning: nil, at: point, in: self.hosting)
            if !picked(), previous?.bundleIdentifier != Bundle.main.bundleIdentifier {
                previous?.activate()
            }
        }
    }

    private func openedMenu(_ menu: NSMenu?) -> NSMenu? {
        menu?.delegate = menuWatcher
        return menu
    }

    private func performed(_ slot: Slot) {
        if slot.id == Widgets.status {
            onStatusClick()
            return
        }
        onPerform(slot)
        // コマンドを送ったら呼び出しを解く。番号選択が続く場合は番号ボタンだけが残る（LayoutRules）
        if summoned, slot.id.hasPrefix("cmd-") {
            summoned = false
            render()
        }
    }

    private func render() {
        let layout = LayoutRules.effective(rest: restLayout, expanded: hovering || summoned, slots: allSlots)
        store.layout = layout
        store.slots = LayoutRules.visibleSlots(allSlots, layout: layout, usageText: usageText())
        onChange()
        guard layout != .hidden else {
            hovering = false
            panel.orderOut(nil)
            return
        }
        // SwiftUIの再レイアウトが済んでから大きさを測る
        DispatchQueue.main.async { [weak self] in
            guard let self else { return }
            self.hosting.layoutSubtreeIfNeeded()
            self.placement.fit(to: self.hosting.fittingSize)
            if !self.panel.isVisible {
                self.panel.orderFrontRegardless()
            }
            self.applyOpacity()
        }
    }

    private func applyOpacity() {
        // 触っている間と呼び出し中は読めるように不透明へ戻す
        panel.alphaValue = (hovering || summoned) ? 1.0 : CGFloat(opacity)
    }

    /// 出入りの通知そのものではなく、待ったあとの実際のカーソル位置で決める。
    /// 広げると枠が動くため、出入りの通知が実際と食い違うことがある
    private func scheduleHoverCheck() {
        hoverCheck?.cancel()
        let inside = panel.frame.contains(NSEvent.mouseLocation)
        let work = DispatchWorkItem { [weak self] in
            guard let self, !self.menuOpen else { return }
            let now = self.panel.isVisible && self.panel.frame.contains(NSEvent.mouseLocation)
            guard now != self.hovering else { return }
            self.hovering = now
            if self.restLayout == .full {
                self.applyOpacity()
            } else {
                self.render()
            }
        }
        hoverCheck = work
        DispatchQueue.main.asyncAfter(deadline: .now() + (inside ? Self.expandDelay : Self.collapseDelay), execute: work)
    }
}

/// 帯から開いたメニューの開閉を知らせる
final class MenuWatcher: NSObject, NSMenuDelegate {
    var onChange: (Bool) -> Void = { _ in }

    func menuWillOpen(_ menu: NSMenu) { onChange(true) }
    func menuDidClose(_ menu: NSMenu) { onChange(false) }
}

/// NSTrackingAreaの受け手
final class HoverTracker: NSResponder {
    var onChange: (Bool) -> Void = { _ in }

    override func mouseEntered(with event: NSEvent) { onChange(true) }
    override func mouseExited(with event: NSEvent) { onChange(false) }
}
