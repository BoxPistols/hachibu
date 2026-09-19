import AppKit
import HachibuCore

/// メニューバーの項目の絵。アプリのアイコンと同じ「斜めの切れ目で2つに分かれた帯」で、左は文字色の淡い灰、右は黄。
/// 使用率の文字も同じ画像の中に描く。文字列の背景色では札に角丸を付けられず、帯の札と形が揃わないため。
/// 比率はscripts/make-icon.swiftと揃える（切れ目の幅は帯の太さの1/3、傾きは0.6、切れ目の位置は左から62%）
enum MenuBarIcon {
    private static let height: CGFloat = 22
    // 絵はバッテリーの表示に近い大きさにする。太くしすぎると帯ではなく2つの塊に見えるので、アプリのアイコンの比率（太さ1：長さ約4）に寄せる
    // 横幅はバッテリーの表示と同じくらいに抑える
    private static let iconWidth: CGFloat = 22
    private static let glyphThickness: CGFloat = 8
    /// 絵と文字をひとまとまりに見せる薄い面。隣のアプリの項目と見分けられるようにする
    private static let groupInset: CGFloat = 1.5
    private static let groupPadding: CGFloat = 4
    private static let groupRadius: CGFloat = 6
    private static let accent = NSColor(srgbRed: 242 / 255, green: 201 / 255, blue: 76 / 255, alpha: 1)
    // 数字の幅を揃え、値が変わっても幅が揺れないようにする
    private static let font = NSFont.monospacedDigitSystemFont(ofSize: NSFont.systemFontSize, weight: .regular)
    private static let tagPadding: CGFloat = 3
    private static let gapAfterIcon: CGFloat = 4

    private enum Piece {
        case text(String, UsageLevel)
        case dot
    }

    /// - segments: 使用率の段階を付けた文字の部品。空ならアイコンだけ
    static func image(segments: [TextSegment] = []) -> NSImage {
        var pieces: [Piece] = []
        for segment in segments {
            if segment.level == .normal, let split = ContextMark.split(segment.text) {
                pieces += [.text(split.model, .normal), .dot, .text(split.rest, .normal)]
            } else {
                pieces.append(.text(segment.text, segment.level))
            }
        }
        let widths = pieces.map(width)
        let textWidth = widths.reduce(0, +)
        let grouped = !pieces.isEmpty
        let lead = grouped ? groupPadding : 0
        let total = lead + iconWidth + (grouped ? gapAfterIcon + textWidth + groupPadding : 0)

        // 描く時点のメニューバーの明暗で色が決まるよう、描画は都度呼ばれる形にする
        let image = NSImage(size: NSSize(width: ceil(total), height: height), flipped: false) { _ in
            if grouped {
                let face = NSBezierPath(roundedRect: NSRect(x: 0.5, y: groupInset, width: ceil(total) - 1, height: height - groupInset * 2),
                                        xRadius: groupRadius, yRadius: groupRadius)
                NSColor.labelColor.withAlphaComponent(0.09).setFill()
                face.fill()
                NSColor.labelColor.withAlphaComponent(0.16).setStroke()
                face.lineWidth = 1
                face.stroke()
            }
            drawGlyph(in: NSRect(x: lead, y: 0, width: iconWidth, height: height))
            var x = lead + iconWidth + gapAfterIcon
            for (piece, w) in zip(pieces, widths) {
                draw(piece, at: x, width: w)
                x += w
            }
            return true
        }
        image.isTemplate = false
        return image
    }

    private static func width(_ piece: Piece) -> CGFloat {
        switch piece {
        case .dot:
            return ContextDot.diameter + 2
        case .text(let text, let level):
            let w = (text as NSString).size(withAttributes: [.font: font]).width
            return level == .normal ? w : w + tagPadding * 2
        }
    }

    private static func draw(_ piece: Piece, at x: CGFloat, width: CGFloat) {
        switch piece {
        case .dot:
            LevelStyle.contextDot.nsColor.setFill()
            let d = ContextDot.diameter
            NSBezierPath(ovalIn: NSRect(x: x + 1, y: height - d - 5, width: d, height: d)).fill()
        case .text(let text, let level):
            let size = (text as NSString).size(withAttributes: [.font: font])
            let y = (height - size.height) / 2
            var color = NSColor.labelColor
            var textX = x
            if let bg = LevelStyle.background(level) {
                bg.nsColor.setFill()
                NSBezierPath(roundedRect: NSRect(x: x, y: y, width: width, height: size.height),
                             xRadius: Metrics.tagRadius, yRadius: Metrics.tagRadius).fill()
                color = LevelStyle.foreground(level).nsColor
                textX += tagPadding
            }
            (text as NSString).draw(at: NSPoint(x: textX, y: y), withAttributes: [.font: font, .foregroundColor: color])
        }
    }

    private static func drawGlyph(in rect: NSRect) {
        let t = glyphThickness
        let gap = t / 3
        let slant = t * 0.6
        let x0 = rect.minX + 1, x1 = rect.maxX - 1
        let y = (rect.height - t) / 2
        let cut = x0 + (x1 - x0) * 0.62

        let left = NSBezierPath()
        left.move(to: NSPoint(x: x0, y: y))
        left.line(to: NSPoint(x: cut - gap / 2 - slant / 2, y: y))
        left.line(to: NSPoint(x: cut - gap / 2 + slant / 2, y: y + t))
        left.line(to: NSPoint(x: x0, y: y + t))
        left.close()

        let right = NSBezierPath()
        right.move(to: NSPoint(x: cut + gap / 2 - slant / 2, y: y))
        right.line(to: NSPoint(x: x1, y: y))
        right.line(to: NSPoint(x: x1, y: y + t))
        right.line(to: NSPoint(x: cut + gap / 2 + slant / 2, y: y + t))
        right.close()

        NSColor.labelColor.withAlphaComponent(0.55).setFill()
        left.fill()
        accent.setFill()
        right.fill()
    }
}
