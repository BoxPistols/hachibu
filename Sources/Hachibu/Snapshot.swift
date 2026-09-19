import AppKit
import HachibuCore
import SwiftUI

/// 開発用: HACHIBU_SNAPSHOT=出力先のフォルダで起動すると、窓を画面に出さずに各画面をPNGへ書き出して終わる。
/// 見た目の確認のたびに利用者の作業中の画面へ窓を出したり、カーソルを動かしたりしないためのもの。
/// メニューは描けないので、項目の一覧をmenus.txtに書く
enum Snapshot {
    static var outputDirectory: URL? {
        ProcessInfo.processInfo.environment["HACHIBU_SNAPSHOT"].map { URL(fileURLWithPath: $0, isDirectory: true) }
    }

    static func run(to dir: URL) {
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let limits = [
            UsageLimit(kind: .fiveHour, percent: 42, resetsAt: nil),
            UsageLimit(kind: .weekly, percent: 76, resetsAt: nil),
            UsageLimit(kind: .model("Fable"), percent: 93, resetsAt: nil),
        ]
        let slot = Slot(id: Slot.statusID, text: Usage.statusText(model: "Opus5 1M", effort: "xhigh", limits: limits),
                        background: .idleBackground, foreground: .idleForeground, help: "")
        let usageText = MenuBarStyle.title(style: .usage, statusText: nil, limits: limits)

        writeMenuBarIcon(to: dir.appendingPathComponent("menubar-icon.png"))

        for language in Language.allCases {
            L10n.current = .for(language)
            let tag = language.rawValue
            for layout in StripLayout.allCases where layout != .hidden {
                let store = StripStore()
                store.layout = layout
                store.slots = LayoutRules.visibleSlots([slot], layout: layout, usageText: usageText)
                write(StripView(store: store), to: dir.appendingPathComponent("strip-\(layout)-\(tag).png"))
            }
            write(ConsentView(onEnable: {}, onDecline: {}), to: dir.appendingPathComponent("consent-\(tag).png"))

            let candidate = Shortcut(keyCode: 40, modifiers: [.control, .option, .command], keyLabel: "K")
            let phases: [(String, RecorderModel.Result, String, Bool)] = [
                ("waiting", .waiting, L10n.current.recorderPrompt, false),
                ("verifying", .verifying, L10n.current.recorderVerify(candidate.display), false),
                ("verified", .verified, L10n.current.recorderVerified(candidate.display), true),
                ("saved", .saved, L10n.current.recorderSaved(candidate.display), false),
            ]
            for (name, result, message, canSave) in phases {
                let model = RecorderModel()
                model.current = Shortcut.defaultSummon.display
                model.live = result == .waiting ? "" : candidate.display
                model.result = result
                model.message = message
                model.canSave = canSave
                write(RecorderView(model: model), to: dir.appendingPathComponent("recorder-\(name)-\(tag).png"))
            }
        }
    }

    /// メニューバーの絵を、明るい地と暗い地の両方に2倍の解像度で描く
    private static func writeMenuBarIcon(to url: URL) {
        let segments = UsageMarkup.segments("Opus5 1M xhigh · S42 W76 F93", thresholds: UsageThresholds(warning: 70, critical: 90)!)
        let images = [MenuBarIcon.image(), MenuBarIcon.image(segments: segments)]
        let width = (images.map(\.size.width).max() ?? 0) + 24
        let rowHeight: CGFloat = 30
        let size = NSSize(width: width, height: rowHeight * 4)
        guard let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: Int(size.width * 2), pixelsHigh: Int(size.height * 2),
                                         bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
                                         colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0) else { return }
        rep.size = size
        NSGraphicsContext.saveGraphicsState()
        NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
        var row: CGFloat = 0
        for dark in [true, false] {
            let appearance = NSAppearance(named: dark ? .darkAqua : .aqua)!
            appearance.performAsCurrentDrawingAppearance {
                for image in images {
                    (dark ? NSColor(white: 0.13, alpha: 1) : NSColor(white: 0.9, alpha: 1)).setFill()
                    NSRect(x: 0, y: row * rowHeight, width: size.width, height: rowHeight).fill()
                    image.draw(in: NSRect(x: 12, y: row * rowHeight + (rowHeight - image.size.height) / 2,
                                          width: image.size.width, height: image.size.height))
                    row += 1
                }
            }
        }
        NSGraphicsContext.restoreGraphicsState()
        try? rep.representation(using: .png, properties: [:])?.write(to: url)
    }

    /// 画面に出していない窓に載せて描く（載せないとSwiftUIの一部が描かれない）。窓は画面の外に作り、表示しない
    private static func write<V: View>(_ view: V, to url: URL) {
        let hosting = NSHostingView(rootView: view.environment(\.colorScheme, .dark))
        hosting.appearance = NSAppearance(named: .darkAqua)
        let size = hosting.fittingSize
        hosting.frame = NSRect(origin: .zero, size: size)
        let window = NSWindow(contentRect: NSRect(x: -20000, y: -20000, width: size.width, height: size.height),
                              styleMask: [.borderless], backing: .buffered, defer: false)
        window.backgroundColor = NSColor(white: 0.12, alpha: 1)
        window.contentView = hosting
        hosting.layoutSubtreeIfNeeded()
        guard let rep = hosting.bitmapImageRepForCachingDisplay(in: hosting.bounds) else { return }
        hosting.cacheDisplay(in: hosting.bounds, to: rep)
        try? rep.representation(using: .png, properties: [:])?.write(to: url)
    }
}
