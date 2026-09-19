import AppKit
import SlashstripCore
import SwiftUI

final class StripStore: ObservableObject {
    @Published var slots: [Slot] = []
    @Published var layout: StripLayout = .full
    @Published var thresholds = Prefs.thresholds
}

extension RGBA {
    var color: Color {
        Color(.sRGB, red: r / 255, green: g / 255, blue: b / 255, opacity: a / 255)
    }

    var nsColor: NSColor {
        NSColor(srgbRed: r / 255, green: g / 255, blue: b / 255, alpha: a / 255)
    }
}

/// 帯そのもの。つまみと状態の枠を横へ並べる。端に収納するときはつまみだけにする。
struct StripView: View {
    @ObservedObject var store: StripStore

    var body: some View {
        Group {
            if store.layout == .tab {
                DockTab(status: store.slots.first, level: worstLevel)
            } else {
                HStack(spacing: 5) {
                    DragHandle()
                        .frame(width: 14, height: Metrics.chipHeight)
                    ForEach(store.slots) { slot in
                        StatusChip(slot: slot, segments: segments(for: slot))
                    }
                }
                .padding(Metrics.inset)
            }
        }
        .background(
            ZStack {
                GlassBackground()
                // 利用者の規約: オーバーレイは80〜90%の不透明度＋背景ぼかし
                Color(white: 0.09).opacity(0.84)
            }
        )
        .clipShape(RoundedRectangle(cornerRadius: Metrics.cornerRadius, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: Metrics.cornerRadius, style: .continuous)
                .strokeBorder(Color.white.opacity(0.1), lineWidth: 1)
        )
        .fixedSize()
    }
}

extension StripView {
    /// 使用率の語に段階を付けるのは状態の枠だけ
    func segments(for slot: Slot) -> [TextSegment] {
        guard slot.id == Slot.statusID else { return [TextSegment(text: slot.text, level: .normal)] }
        return UsageMarkup.segments(slot.text, thresholds: store.thresholds)
    }

    var worstLevel: UsageLevel {
        store.slots.first.map { segments(for: $0).map(\.level).max() ?? .normal } ?? .normal
    }
}

/// 使用率の段階ごとの札の色。黄は濃い文字（10.8:1）、赤は白い文字（4.8:1）
enum LevelStyle {
    static func background(_ level: UsageLevel) -> RGBA? {
        switch level {
        case .normal: return nil
        case .warning: return RGBA(r: 242, g: 201, b: 76, a: 255)
        case .critical: return RGBA(r: 217, g: 48, b: 37, a: 255)
        }
    }

    static func foreground(_ level: UsageLevel) -> RGBA {
        level == .warning ? RGBA(r: 28, g: 28, b: 30, a: 255) : .white
    }
}

enum Metrics {
    static let chipHeight: CGFloat = 26
    static let inset: CGFloat = 5
    static let cornerRadius: CGFloat = 12
    // 12px未満は使わない
    static let fontSize: CGFloat = 13
    // つまみの幅。高さは帯と揃え、広げたときに上下へ跳ばないようにする
    static let tabWidth: CGFloat = 20
}

/// 端に収納したときの姿。状態は点の色だけで示し、全体がドラッグで動かせるつまみになる
struct DockTab: View {
    let status: Slot?
    let level: UsageLevel

    var body: some View {
        ZStack {
            DragHandle(drawsDots: false)
            Circle()
                .fill(dotColor)
                .frame(width: 8, height: 8)
                .allowsHitTesting(false)
            // 使用率が黄や赤の段階なら、点の周りに輪を付ける（点の色は状態に使う）
            if let ring = LevelStyle.background(level) {
                Circle()
                    .strokeBorder(ring.color, lineWidth: 2)
                    .frame(width: 14, height: 14)
                    .allowsHitTesting(false)
            }
        }
        .frame(width: Metrics.tabWidth, height: Metrics.chipHeight + Metrics.inset * 2)
    }

    private var dotColor: Color {
        // 平常時（背景が透明）は灰色の点にする
        guard let bg = status?.background, bg.a > 0 else { return RGBA.idleForeground.color.opacity(0.7) }
        return bg.color
    }
}

/// 状態の枠。押す対象ではなく表示だけ。移動はつまみ、設定は右クリックで行う
struct StatusChip: View {
    let slot: Slot
    let segments: [TextSegment]

    var body: some View {
        HStack(spacing: 0) {
            ForEach(Array(segments.enumerated()), id: \.offset) { _, segment in
                SegmentText(segment: segment, normalColor: slot.foreground)
            }
        }
        .font(.system(size: Metrics.fontSize, weight: .medium).monospacedDigit())
        .lineLimit(1)
        .padding(.horizontal, 10)
        .frame(height: Metrics.chipHeight)
        .background(
            RoundedRectangle(cornerRadius: 7, style: .continuous)
                .fill(slot.background.color)
        )
        .opacity(slot.dimmed ? 0.5 : 1)
        // 見えている文字と同じ説明は出さない。出すのはリセット時刻など、帯に載っていない情報だけ
        .help(slot.help ?? "")
    }
}

/// 文字の一部。段階が付いていれば札で囲む
struct SegmentText: View {
    let segment: TextSegment
    let normalColor: RGBA

    /// 空白だけの部品（札と札の間）は、測るときに幅0と見積もられ、描くときの幅と食い違って先頭の札が切れた（実測）。
    /// 測るときにも省かれないノーブレークスペースに置き換える
    private var text: String {
        segment.text.replacingOccurrences(of: " ", with: "\u{00A0}")
    }

    var body: some View {
        // 部品ごとに自分の幅を保つ。窓の幅が1ptに満たない差で足りないと、先頭の札が「…」に詰められた（実測）
        if let bg = LevelStyle.background(segment.level) {
            Text(verbatim: text)
                .foregroundStyle(LevelStyle.foreground(segment.level).color)
                .fixedSize(horizontal: true, vertical: false)
                .padding(.horizontal, 4)
                .padding(.vertical, 1)
                .background(RoundedRectangle(cornerRadius: 4, style: .continuous).fill(bg.color))
        } else {
            Text(verbatim: text)
                .foregroundStyle(normalColor.color)
                .fixedSize(horizontal: true, vertical: false)
        }
    }
}

/// 背景ぼかし。アプリが非アクティブのままでもぼかしを止めない
struct GlassBackground: NSViewRepresentable {
    func makeNSView(context: Context) -> NSVisualEffectView {
        let v = StillEffectView()
        v.material = .hudWindow
        v.blendingMode = .behindWindow
        v.state = .active
        return v
    }

    func updateNSView(_ nsView: NSVisualEffectView, context: Context) {}
}

/// 押してもウィンドウのドラッグを始めないぼかし面
final class StillEffectView: NSVisualEffectView {
    override var mouseDownCanMoveWindow: Bool { false }

    override func menu(for event: NSEvent) -> NSMenu? {
        enclosingContextMenu(for: event)
    }
}

/// 帯を動かすつまみ。SwiftUIの面はウィンドウのドラッグを受けないので、ここだけAppKitで持つ
struct DragHandle: NSViewRepresentable {
    var drawsDots = true

    func makeNSView(context: Context) -> DragHandleView {
        let v = DragHandleView()
        v.drawsDots = drawsDots
        return v
    }

    func updateNSView(_ nsView: DragHandleView, context: Context) {
        nsView.drawsDots = drawsDots
    }
}

final class DragHandleView: NSView {
    var drawsDots = true {
        didSet { needsDisplay = true }
    }

    override init(frame: NSRect) {
        super.init(frame: frame)
        setAccessibilityRole(.handle)
        setAccessibilityLabel(L10n.current.dragHandle)
    }

    required init?(coder: NSCoder) { nil }

    override func acceptsFirstMouse(for event: NSEvent?) -> Bool { true }

    override func mouseDown(with event: NSEvent) {
        window?.performDrag(with: event)
    }

    override func menu(for event: NSEvent) -> NSMenu? {
        enclosingContextMenu(for: event)
    }

    override func resetCursorRects() {
        addCursorRect(bounds, cursor: .openHand)
    }

    override func draw(_ dirtyRect: NSRect) {
        guard drawsDots else { return }
        NSColor.white.withAlphaComponent(0.32).setFill()
        let d: CGFloat = 2.5
        let gap: CGFloat = 3.5
        let cols = 2
        let rows = 3
        let w = CGFloat(cols) * d + CGFloat(cols - 1) * gap
        let h = CGFloat(rows) * d + CGFloat(rows - 1) * gap
        let x0 = bounds.midX - w / 2
        let y0 = bounds.midY - h / 2
        for c in 0..<cols {
            for r in 0..<rows {
                let rect = NSRect(x: x0 + CGFloat(c) * (d + gap), y: y0 + CGFloat(r) * (d + gap), width: d, height: d)
                NSBezierPath(ovalIn: rect).fill()
            }
        }
    }
}
