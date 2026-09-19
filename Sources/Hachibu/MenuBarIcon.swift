import AppKit

/// メニューバーの項目の絵。アプリのアイコンと同じ「斜めの切れ目で2つに分かれた帯」を、テンプレート画像（単色）で描く。
/// 比率はscripts/make-icon.swiftと揃える（切れ目の幅は帯の太さの1/3、傾きは0.6、切れ目の位置は左から62%）
enum MenuBarIcon {
    static func image() -> NSImage {
        let size = NSSize(width: 20, height: 16)
        let image = NSImage(size: size, flipped: false) { rect in
            let t: CGFloat = 6
            let gap = t / 3 + 0.5   // 小さい表示でも切れ目が潰れないよう、わずかに広げる
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

            // アプリのアイコンの「左は灰、右は黄」を、単色の濃淡で写す（テンプレート画像は不透明度が生きる）
            NSColor.black.withAlphaComponent(0.45).setFill()
            left.fill()
            NSColor.black.setFill()
            right.fill()
            return true
        }
        image.isTemplate = true
        return image
    }
}
