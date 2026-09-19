import AppKit
import SwiftUI

/// 押してもアクティブにならないパネル。
///
/// 帯を触っても、使っているアプリからキー入力が離れないようにする（ドラッグや右クリックで前面が入れ替わらない）。
final class StripPanel: NSPanel {
    init(contentView: NSView) {
        super.init(contentRect: NSRect(x: 0, y: 0, width: 240, height: 40),
                   styleMask: [.borderless, .nonactivatingPanel],
                   backing: .buffered, defer: false)
        isFloatingPanel = true
        level = .floating
        // 全デスクトップとフルスクリーンのターミナルの上にも出す
        collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary, .ignoresCycle]
        hidesOnDeactivate = false
        becomesKeyOnlyIfNeeded = true
        // 移動はつまみ（performDrag）だけで行う。背景を掴めるとボタンを押したつもりが帯ごと動く
        isMovableByWindowBackground = false
        backgroundColor = .clear
        isOpaque = false
        hasShadow = true
        // Touch Barの配色（黒地前提で算出された色）をそのまま使うため、外観は暗色に固定する
        appearance = NSAppearance(named: .darkAqua)
        self.contentView = contentView
    }

    override var canBecomeKey: Bool { false }
    override var canBecomeMain: Bool { false }
}

/// 右クリックメニューを出す面。帯の中の部品は、ここまで遡ってメニューを受け取る
protocol ContextMenuHost: AnyObject {
    func contextMenu(for event: NSEvent) -> NSMenu?
}

extension NSView {
    /// 自分より外側にある右クリックメニューの持ち主から、メニューを受け取る
    func enclosingContextMenu(for event: NSEvent) -> NSMenu? {
        var view = superview
        while let v = view {
            if let host = v as? ContextMenuHost { return host.contextMenu(for: event) }
            view = v.superview
        }
        return nil
    }
}

/// 非アクティブなウィンドウでも1回目のクリックをボタンに届ける。
final class FirstClickHostingView<Content: View>: NSHostingView<Content>, ContextMenuHost {
    var contextMenuProvider: (() -> NSMenu?)?
    /// 中身の大きさが変わったとき。表示を更新した直後に測ると、描き直しの前の大きさを拾うことがある（実測で文字が切れた）
    var onContentSizeChange: (() -> Void)?

    override func invalidateIntrinsicContentSize() {
        super.invalidateIntrinsicContentSize()
        onContentSizeChange?()
    }

    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }
    // 枠なしウィンドウでは、この値がtrueの面を押すとボタンに届かずウィンドウのドラッグになる（実測）
    override var mouseDownCanMoveWindow: Bool { false }

    override func menu(for event: NSEvent) -> NSMenu? {
        contextMenu(for: event)
    }

    func contextMenu(for event: NSEvent) -> NSMenu? {
        contextMenuProvider?()
    }
}

/// 帯の置き場所。中身の幅が変わっても、利用者が寄せた側の端を動かさない。
///
/// 位置は「左端・中央・右端のどれを基準にするか」と、その基準点の座標で覚える。
/// 左上の座標で覚えると、フルで保存した位置にコンパクトで出したとき中央寄せの帯が左へずれる。
final class StripPlacement {
    private let panel: NSPanel
    private static let anchorKey = "stripAnchor"
    private static let edgeKey = "stripEdge"
    private static let bottomMargin: CGFloat = 12

    private enum Edge: String {
        case left, center, right
    }

    private var isResizing = false

    init(panel: NSPanel) {
        self.panel = panel
        NotificationCenter.default.addObserver(forName: NSWindow.didMoveNotification, object: panel, queue: .main) {
            [weak self] _ in self?.userMoved()
        }
    }

    /// 中身の大きさに合わせて置き直す
    func fit(to size: NSSize) {
        // 端数の切り上げに1ptの余裕を足す。ちょうどの幅だと、描くときの丸めで文字が詰められることがある
        let size = NSSize(width: ceil(size.width) + 1, height: ceil(size.height))
        guard size.width > 0, size.height > 0 else { return }
        setFrame(clamped(frame(for: size)))
    }

    func reset() {
        Prefs.defaults.removeObject(forKey: Self.anchorKey)
        Prefs.defaults.removeObject(forKey: Self.edgeKey)
        setFrame(clamped(frame(for: panel.frame.size)))
    }

    private func setFrame(_ frame: NSRect) {
        isResizing = true
        panel.setFrame(frame, display: true)
        isResizing = false
    }

    private func userMoved() {
        guard !isResizing else { return }
        let frame = panel.frame
        // 画面の左3分の1に置いたら左端、右3分の1なら右端を基準にする
        let visible = (panel.screen ?? NSScreen.main)?.visibleFrame ?? frame
        let ratio = (frame.midX - visible.minX) / max(visible.width, 1)
        let edge: Edge = ratio < 1.0 / 3 ? .left : (ratio > 2.0 / 3 ? .right : .center)
        let x: CGFloat
        switch edge {
        case .left: x = frame.minX
        case .center: x = frame.midX
        case .right: x = frame.maxX
        }
        Prefs.defaults.set(NSStringFromPoint(NSPoint(x: x, y: frame.minY)), forKey: Self.anchorKey)
        Prefs.defaults.set(edge.rawValue, forKey: Self.edgeKey)
    }

    private func frame(for size: NSSize) -> NSRect {
        if let saved = Prefs.defaults.string(forKey: Self.anchorKey),
           let edge = Edge(rawValue: Prefs.defaults.string(forKey: Self.edgeKey) ?? "") {
            let anchor = NSPointFromString(saved)
            let x: CGFloat
            switch edge {
            case .left: x = anchor.x
            case .center: x = anchor.x - size.width / 2
            case .right: x = anchor.x - size.width
            }
            let frame = NSRect(x: x, y: anchor.y, width: size.width, height: size.height)
            // 外部ディスプレイを外した後などで、保存位置がどの画面にも無ければ既定に戻す
            if NSScreen.screens.contains(where: { $0.visibleFrame.intersects(frame) }) {
                return frame
            }
        }
        let visible = (NSScreen.screens.first ?? NSScreen.main)?.visibleFrame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        return NSRect(x: visible.midX - size.width / 2, y: visible.minY + Self.bottomMargin,
                      width: size.width, height: size.height)
    }

    private func clamped(_ frame: NSRect) -> NSRect {
        let screen = NSScreen.screens.first(where: { $0.visibleFrame.intersects(frame) }) ?? NSScreen.main
        guard let visible = screen?.visibleFrame else { return frame }
        var f = frame
        f.origin.x = min(max(f.minX, visible.minX), visible.maxX - f.width)
        f.origin.y = min(max(f.minY, visible.minY), visible.maxY - f.height)
        return f
    }
}
