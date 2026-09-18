import AppKit
import SlashstripCore

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
    private(set) var thresholds = Prefs.thresholds

    /// メニューの項目にカーソルを乗せている間だけ使う仮の値。選ばずに閉じたら消す
    private var previewLayout: StripLayout?
    private var previewOpacity: Double?
    private var previewThresholds: UsageThresholds?
    /// メニューで選んだ直後は、カーソルが帯の上に残っていても広げない（選んだモードをすぐ見せる）。
    /// カーソルが一度帯の外に出たら解く
    private var hoverSuppressed = false

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
    /// 帯から開いたメニューと、表示を切り替える項目を持つ入れ子のメニューに付ける。
    /// 開閉を数え、項目に乗せたときに仮の表示を出す
    let menuWatcher = MenuWatcher()
    private var openMenus = 0

    // 乗せてから広げるまでと、離してから畳むまでの待ち。通りがかりで開閉しないため
    private static let expandDelay: TimeInterval = 0.15
    private static let collapseDelay: TimeInterval = 0.6
    // メニューが閉じてから、選んだ項目の処理が届くまでの余裕
    private static let previewGrace: TimeInterval = 0.5

    init() {
        hosting = FirstClickHostingView(rootView: StripView(store: store))
        hosting.sizingOptions = [.intrinsicContentSize]
        panel = StripPanel(contentView: hosting)
        placement = StripPlacement(panel: panel)
        store.thresholds = thresholds
        store.onPerform = { [weak self] slot in self?.performed(slot) }
        tracker.onChange = { [weak self] _ in self?.scheduleHoverCheck() }
        menuWatcher.onOpen = { [weak self] in self?.openMenus += 1 }
        menuWatcher.onClose = { [weak self] in
            guard let self else { return }
            self.openMenus = max(0, self.openMenus - 1)
            guard self.openMenus == 0 else { return }
            // メニューは選んだ項目を短く点滅させてから処理を送るので、閉じた通知の方が先に届く。
            // ここで仮の表示を消すと「仮の値→元の値→選んだ値」とちらつくため、選ばれなかったときだけ少し待って消す
            DispatchQueue.main.asyncAfter(deadline: .now() + Self.previewGrace) { [weak self] in
                guard let self, self.openMenus == 0 else { return }
                self.clearPreview()
                self.scheduleHoverCheck()
            }
        }
        menuWatcher.onHighlight = { [weak self] item in
            if let item = item as? PreviewMenuItem {
                item.preview()
            } else if item != nil {
                self?.clearPreview()
            }
            // 項目が無い通知（カーソルがメニューの外へ出た、選んだ項目の点滅）では消さない。点滅のたびにちらつくため
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

    // MARK: - 設定の変更（メニューから）

    func setRestLayout(_ layout: StripLayout) {
        restLayout = layout
        Prefs.restLayout = layout
        previewLayout = nil
        summoned = false
        suppressHover()
        render()
    }

    func setThresholds(_ value: UsageThresholds) {
        thresholds = value
        Prefs.thresholds = value
        previewThresholds = nil
        render()
    }

    func setOpacity(_ value: Double) {
        opacity = value
        Prefs.opacity = value
        previewOpacity = nil
        suppressHover()
        applyOpacity()
    }

    // MARK: - 仮の表示（メニューの項目に乗せている間）

    func preview(layout: StripLayout) {
        previewLayout = layout
        render()
    }

    func preview(opacity: Double) {
        previewOpacity = opacity
        applyOpacity()
    }

    func preview(thresholds: UsageThresholds) {
        previewThresholds = thresholds
        render()
    }

    private func clearPreview() {
        guard previewLayout != nil || previewOpacity != nil || previewThresholds != nil else { return }
        previewLayout = nil
        previewOpacity = nil
        previewThresholds = nil
        render()
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
    /// - picked: 項目が選ばれたかどうか。項目の処理の中で立てる
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

    // MARK: - 描画

    /// マウスを乗せたことによる展開。メニューで選んだ直後と、仮の表示の間は広げない
    private var hoverExpands: Bool {
        hovering && !hoverSuppressed && previewLayout == nil && previewOpacity == nil
    }

    private func render() {
        let rest = previewLayout ?? restLayout
        let layout = LayoutRules.effective(rest: rest, expanded: hoverExpands || summoned, slots: allSlots)
        store.thresholds = previewThresholds ?? thresholds
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
        // 触っている間と呼び出し中は読めるように不透明へ戻す。仮の表示の間はその値を見せる
        let solid = hoverExpands || summoned
        panel.alphaValue = solid ? 1.0 : CGFloat(previewOpacity ?? opacity)
    }

    // MARK: - マウスの出入り

    private func suppressHover() {
        hoverSuppressed = true
        // 小さくなって帯がカーソルの下から外れた場合、外へ出た通知が来ないことがあるので、置き直した後に確かめる
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) { [weak self] in
            guard let self else { return }
            if !self.panel.frame.contains(NSEvent.mouseLocation) {
                self.hoverSuppressed = false
                self.hovering = false
            }
        }
    }

    /// 出入りの通知そのものではなく、待ったあとの実際のカーソル位置で決める。
    /// 広げると枠が動くため、出入りの通知が実際と食い違うことがある
    private func scheduleHoverCheck() {
        hoverCheck?.cancel()
        let inside = panel.frame.contains(NSEvent.mouseLocation)
        let work = DispatchWorkItem { [weak self] in
            guard let self, self.openMenus == 0 else { return }
            let now = self.panel.isVisible && self.panel.frame.contains(NSEvent.mouseLocation)
            if !now { self.hoverSuppressed = false }
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

/// 帯から開いたメニューの開閉と、項目への乗せ替えを知らせる
final class MenuWatcher: NSObject, NSMenuDelegate {
    var onOpen: () -> Void = {}
    var onClose: () -> Void = {}
    var onHighlight: (NSMenuItem?) -> Void = { _ in }

    func menuWillOpen(_ menu: NSMenu) { onOpen() }
    func menuDidClose(_ menu: NSMenu) { onClose() }
    func menu(_ menu: NSMenu, willHighlight item: NSMenuItem?) { onHighlight(item) }
}

/// NSTrackingAreaの受け手
final class HoverTracker: NSResponder {
    var onChange: (Bool) -> Void = { _ in }

    override func mouseEntered(with event: NSEvent) { onChange(true) }
    override func mouseExited(with event: NSEvent) { onChange(false) }
}
