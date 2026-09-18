import AppKit
import SlashstripCore
import SwiftUI

/// 「ショートカットを変更…」で開く小さな窓。次に押された組み合わせをそのまま記録する。
///
/// キー入力を受けるためにこの窓だけは前面に出す。閉じたら元のアプリへ戻す。
final class ShortcutRecorder {
    private let hotKeys: HotKeyCenter
    private let model = RecorderModel()
    private var panel: RecorderPanel?
    private var monitor: Any?
    private var previousApp: NSRunningApplication?

    init(hotKeys: HotKeyCenter) {
        self.hotKeys = hotKeys
    }

    func show() {
        guard panel == nil else {
            panel?.makeKeyAndOrderFront(nil)
            return
        }
        previousApp = NSWorkspace.shared.frontmostApplication
        hotKeys.suspend()
        model.current = hotKeys.shortcut?.display
        model.message = L10n.recorderPrompt
        model.onDisable = { [weak self] in
            self?.hotKeys.disable()
            self?.close(restoreHotKey: false)
        }
        model.onCancel = { [weak self] in self?.close(restoreHotKey: true) }

        let hosting = NSHostingView(rootView: RecorderView(model: model))
        let panel = RecorderPanel(contentView: hosting)
        panel.setContentSize(hosting.fittingSize)
        panel.center()
        self.panel = panel

        monitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            self?.record(event)
            return nil
        }
        NSApp.activate()
        panel.makeKeyAndOrderFront(nil)
    }

    private func record(_ event: NSEvent) {
        let flags = event.modifierFlags.intersection(.deviceIndependentFlagsMask)
        // 修飾キーなしのEscは取り消し
        if event.keyCode == 53, flags.isDisjoint(with: [.command, .option, .control, .shift]) {
            close(restoreHotKey: true)
            return
        }
        var modifiers: Shortcut.Modifiers = []
        if flags.contains(.control) { modifiers.insert(.control) }
        if flags.contains(.option) { modifiers.insert(.option) }
        if flags.contains(.shift) { modifiers.insert(.shift) }
        if flags.contains(.command) { modifiers.insert(.command) }
        let keyCode = UInt32(event.keyCode)
        let shortcut = Shortcut(keyCode: keyCode, modifiers: modifiers,
                                keyLabel: Shortcut.label(keyCode: keyCode, characters: event.charactersIgnoringModifiers))
        guard shortcut.isAcceptable else {
            model.message = L10n.recorderRejected
            return
        }
        if hotKeys.change(to: shortcut) {
            close(restoreHotKey: false)
        } else {
            hotKeys.suspend()
            model.message = L10n.recorderTaken(shortcut.display)
        }
    }

    private func close(restoreHotKey: Bool) {
        if let monitor { NSEvent.removeMonitor(monitor) }
        monitor = nil
        if restoreHotKey { hotKeys.resume() }
        panel?.orderOut(nil)
        panel = nil
        previousApp?.activate()
        previousApp = nil
    }
}

final class RecorderModel: ObservableObject {
    @Published var current: String?
    @Published var message = ""
    var onDisable: () -> Void = {}
    var onCancel: () -> Void = {}
}

struct RecorderView: View {
    @ObservedObject var model: RecorderModel

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text(L10n.recorderTitle)
                .font(.system(size: 15, weight: .semibold))
            Text(L10n.recorderCurrent(model.current))
                .font(.system(size: 13).monospacedDigit())
                .foregroundStyle(.secondary)
            Text(model.message)
                .font(.system(size: 13))
                .fixedSize(horizontal: false, vertical: true)
            HStack {
                Spacer()
                Button(L10n.recorderDisable, action: model.onDisable)
                Button(L10n.recorderCancel, action: model.onCancel)
            }
            .font(.system(size: 13))
        }
        .padding(18)
        .frame(width: 360)
        .background(
            ZStack {
                GlassBackground()
                // 利用者の規約: オーバーレイは80〜90%の不透明度＋背景ぼかし
                Color(white: 0.1).opacity(0.86)
            }
        )
        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 14, style: .continuous)
                .strokeBorder(Color.white.opacity(0.1), lineWidth: 1)
        )
    }
}

/// キー入力を受けられる枠なしパネル
final class RecorderPanel: NSPanel {
    init(contentView: NSView) {
        super.init(contentRect: NSRect(x: 0, y: 0, width: 360, height: 160),
                   styleMask: [.borderless], backing: .buffered, defer: false)
        level = .floating
        backgroundColor = .clear
        isOpaque = false
        hasShadow = true
        appearance = NSAppearance(named: .darkAqua)
        isReleasedWhenClosed = false
        self.contentView = contentView
    }

    override var canBecomeKey: Bool { true }
}
