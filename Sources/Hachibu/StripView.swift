import AppKit
import HachibuCore
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
                HStack(spacing: 2) {
                    DragHandle()
                        .frame(width: 12, height: Metrics.chipHeight)
                    ForEach(store.slots) { slot in
                        StatusChip(slot: slot, segments: segments(for: slot))
                    }
                }
                .padding(.leading, Metrics.inset)
                .padding(.trailing, Metrics.inset + 4)
                .padding(.vertical, Metrics.inset)
            }
        }
        .background(GlassSurface())
        .clipShape(RoundedRectangle(cornerRadius: Metrics.cornerRadius, style: .continuous))
        .overlay(GlassRim(cornerRadius: Metrics.cornerRadius))
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
    static let chipHeight: CGFloat = 22
    static let inset: CGFloat = 3
    /// 角丸は2段だけ。面（帯、窓）は8、面の中の札は3。メニューバーの札も同じ3にする
    static let cornerRadius: CGFloat = 8
    static let tagRadius: CGFloat = 3
    /// 大きい窓（同意画面、ショートカットの窓）の角丸
    static let panelRadius: CGFloat = 12
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
        .padding(.leading, 4)
        .frame(height: Metrics.chipHeight)
        // 帯の面の中にもう1つ枠を重ねない（角丸が入れ子になり、高さも取るため）
        .opacity(slot.dimmed ? 0.5 : 1)
        // 見えている文字と同じ説明は出さない。出すのはリセット時刻など、帯に載っていない情報だけ
        .help(([ContextMark.spelledOut(slot.text)].compactMap { $0 } + [slot.help ?? ""]).filter { !$0.isEmpty }.joined(separator: "\n"))
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
                .padding(.horizontal, 3)
                .padding(.vertical, 1)
                .background(RoundedRectangle(cornerRadius: Metrics.tagRadius, style: .continuous).fill(bg.color))
        } else if let split = ContextMark.split(segment.text) {
            // コンテキスト長は文字で書かず、モデル名の肩に小さな点で示す（文字はメニューとツールチップに出す）
            HStack(alignment: .top, spacing: 1.5) {
                Text(verbatim: split.model)
                ContextDot()
                    .padding(.top, 2)
                    .accessibilityLabel(Text(verbatim: "\(split.context) context"))
                Text(verbatim: split.rest.replacingOccurrences(of: " ", with: "\u{00A0}"))
                    .padding(.leading, -1.5)
            }
            .foregroundStyle(normalColor.color)
            .fixedSize(horizontal: true, vertical: false)
        } else {
            Text(verbatim: text)
                .foregroundStyle(normalColor.color)
                .fixedSize(horizontal: true, vertical: false)
        }
    }
}

/// コンテキスト長の印。モデル名の肩に付く小さな点
struct ContextDot: View {
    static let diameter: CGFloat = 4

    var body: some View {
        Circle()
            .fill(Color(.sRGB, red: 242 / 255, green: 201 / 255, blue: 76 / 255, opacity: 1))
            .frame(width: Self.diameter, height: Self.diameter)
    }
}

/// ガラスの面。背景ぼかしの上に、上が明るく下が暗い色を重ねる。
/// 利用者の規約: オーバーレイは80〜90%の不透明度＋背景ぼかし
struct GlassSurface: View {
    var body: some View {
        ZStack {
            GlassBackground()
            LinearGradient(colors: [Color(white: 0.17).opacity(0.80), Color(white: 0.07).opacity(0.88)],
                           startPoint: .top, endPoint: .bottom)
        }
    }
}

/// ガラスの縁。上の辺が光を受け、下へ向かって消える
struct GlassRim: View {
    let cornerRadius: CGFloat

    var body: some View {
        RoundedRectangle(cornerRadius: cornerRadius, style: .continuous)
            .strokeBorder(
                LinearGradient(colors: [Color.white.opacity(0.34), Color.white.opacity(0.08), Color.white.opacity(0.14)],
                               startPoint: .top, endPoint: .bottom),
                lineWidth: 1
            )
            .allowsHitTesting(false)
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
    }

    // 言語を切り替えたあとも、読み上げのたびに今の言語で返す
    override func accessibilityLabel() -> String? { L10n.current.dragHandle }

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
