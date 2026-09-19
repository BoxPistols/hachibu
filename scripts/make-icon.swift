// アプリのアイコン（斜めの切れ目の帯）を描き、Resources/AppIcon.icnsを作る。
//   swiftc -O -o /tmp/make-icon scripts/make-icon.swift && /tmp/make-icon Resources
// 決める数は帯の太さtだけ。間隔、傾き、余白はtから導く（寸法を手で決めると形が締まらない）
import AppKit
import CoreGraphics

let canvas: CGFloat = 1024
// macOSのアイコンの格子: 1024の画面に824の本体
let body = CGRect(x: 100, y: 100, width: 824, height: 824)
let radius: CGFloat = 185

let t: CGFloat = 132
let gap = t / 3
let slant = t * 0.6

let graphiteTop = CGColor(srgbRed: 0.20, green: 0.21, blue: 0.24, alpha: 1)
let graphiteBottom = CGColor(srgbRed: 0.09, green: 0.10, blue: 0.12, alpha: 1)
let gray = CGColor(srgbRed: 0.34, green: 0.35, blue: 0.38, alpha: 1)
// 帯で「注意」に使う黄と同じ色
let yellow = CGColor(srgbRed: 242 / 255, green: 201 / 255, blue: 76 / 255, alpha: 1)
let rim = CGColor(srgbRed: 1, green: 1, blue: 1, alpha: 0.10)

func draw(_ ctx: CGContext) {
    let shape = CGPath(roundedRect: body, cornerWidth: radius, cornerHeight: radius, transform: nil)
    ctx.saveGState()
    ctx.addPath(shape)
    ctx.clip()
    let gradient = CGGradient(colorsSpace: CGColorSpace(name: CGColorSpace.sRGB),
                              colors: [graphiteTop, graphiteBottom] as CFArray, locations: [0, 1])!
    ctx.drawLinearGradient(gradient, start: CGPoint(x: 0, y: body.maxY), end: CGPoint(x: 0, y: body.minY), options: [])
    ctx.restoreGState()
    // 4辺同じ細い縁
    ctx.addPath(CGPath(roundedRect: body.insetBy(dx: 3, dy: 3), cornerWidth: radius - 3, cornerHeight: radius - 3, transform: nil))
    ctx.setStrokeColor(rim)
    ctx.setLineWidth(6)
    ctx.strokePath()

    // 帯を斜めの切れ目で2つに分け、右を黄にする。左右の余白は太さ1つ分
    let x0 = body.minX + t, x1 = body.maxX - t
    let y = body.midY - t / 2
    let cut = body.midX + t * 0.4
    let left = CGMutablePath()
    left.move(to: CGPoint(x: x0, y: y))
    left.addLine(to: CGPoint(x: cut - gap / 2 - slant / 2, y: y))
    left.addLine(to: CGPoint(x: cut - gap / 2 + slant / 2, y: y + t))
    left.addLine(to: CGPoint(x: x0, y: y + t))
    left.closeSubpath()
    ctx.addPath(left)
    ctx.setFillColor(gray)
    ctx.fillPath()
    let right = CGMutablePath()
    right.move(to: CGPoint(x: cut + gap / 2 - slant / 2, y: y))
    right.addLine(to: CGPoint(x: x1, y: y))
    right.addLine(to: CGPoint(x: x1, y: y + t))
    right.addLine(to: CGPoint(x: cut + gap / 2 + slant / 2, y: y + t))
    right.closeSubpath()
    ctx.addPath(right)
    ctx.setFillColor(yellow)
    ctx.fillPath()
}

func png(size: Int) -> Data {
    let ctx = CGContext(data: nil, width: size, height: size, bitsPerComponent: 8, bytesPerRow: 0,
                        space: CGColorSpace(name: CGColorSpace.sRGB)!,
                        bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue)!
    ctx.interpolationQuality = .high
    ctx.scaleBy(x: CGFloat(size) / canvas, y: CGFloat(size) / canvas)
    draw(ctx)
    return NSBitmapImageRep(cgImage: ctx.makeImage()!).representation(using: .png, properties: [:])!
}

let outDir = URL(fileURLWithPath: CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "Resources")
let iconset = FileManager.default.temporaryDirectory.appendingPathComponent("AppIcon.iconset")
try? FileManager.default.removeItem(at: iconset)
try FileManager.default.createDirectory(at: iconset, withIntermediateDirectories: true)
for base in [16, 32, 128, 256, 512] {
    try png(size: base).write(to: iconset.appendingPathComponent("icon_\(base)x\(base).png"))
    try png(size: base * 2).write(to: iconset.appendingPathComponent("icon_\(base)x\(base)@2x.png"))
}
let p = Process()
p.executableURL = URL(fileURLWithPath: "/usr/bin/iconutil")
p.arguments = ["-c", "icns", iconset.path, "-o", outDir.appendingPathComponent("AppIcon.icns").path]
try p.run()
p.waitUntilExit()
// READMEに載せる原寸の画像
try png(size: 1024).write(to: URL(fileURLWithPath: "docs/images/app-icon.png"))
print(p.terminationStatus == 0 ? "wrote \(outDir.path)/AppIcon.icns" : "iconutil failed")
