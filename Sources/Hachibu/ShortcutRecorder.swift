import AppKit
import HachibuCore
import SwiftUI

/// 「ショートカットを変更…」で開く小さな窓。押している修飾キーをその場で表示し、キーを押した瞬間に登録を試す。
///
/// 登録できても押したときに届くとは限らない（macOSのショートカットはシステムが先に受け取る）。そのため
/// macOSに割り当て済みの組み合わせは登録の前に弾き、登録後はもう一度押してもらって、届いたときに初めて保存する。
/// 状態の判断はShortcutRecordingが持つ。この窓はキー入力を渡し、結果を表示するだけにする。
/// キー入力を受けるためにこの窓だけは前面に出す。閉じたら元のアプリへ戻す。
final class ShortcutRecorder {
    private let hotKeys: HotKeyCenter
    private let model = RecorderModel()
    private var recording = ShortcutRecording()
    private var panel: RecorderPanel?
    private var monitor: Any?
    private var previousApp: NSRunningApplication?

    // 保存できたことを読めるだけ表示してから閉じる
    private static let closeDelay: TimeInterval = 0.9

    init(hotKeys: HotKeyCenter) {
        self.hotKeys = hotKeys
    }

    func show() {
        guard panel == nil else {
            activate()
            return
        }
        previousApp = NSWorkspace.shared.frontmostApplication
        recording = ShortcutRecording()
        hotKeys.suspend()
        // 窓を開いている間に届いたショートカットは、帯の呼び出しではなく確認に回す
        hotKeys.interceptor = { [weak self] in
            self?.arrived()
            return true
        }
        model.live = ""
        model.onDisable = { [weak self] in self?.turnOff() }
        model.onCancel = { [weak self] in self?.cancel() }
        refresh()

        let hosting = NSHostingView(rootView: RecorderView(model: model))
        let panel = RecorderPanel(contentView: hosting)
        panel.setContentSize(hosting.fittingSize)
        panel.center()
        self.panel = panel

        monitor = NSEvent.addLocalMonitorForEvents(matching: [.keyDown, .flagsChanged]) { [weak self] event in
            guard let self else { return event }
            if event.type == .flagsChanged {
                self.showHeld(event.modifierFlags)
                return event
            }
            self.keyDown(event)
            return nil
        }
        activate()
    }

    /// macOS 14以降のactivate()は要求が通らないことがあり、その場合キー入力が元のアプリへ流れる。
    /// ここは利用者がメニューで選んだ直後なので、強制して前面に出す
    private func activate() {
        NSApp.activate(ignoringOtherApps: true)
        panel?.makeKeyAndOrderFront(nil)
    }

    private func modifiers(_ flags: NSEvent.ModifierFlags) -> Shortcut.Modifiers {
        let flags = flags.intersection(.deviceIndependentFlagsMask)
        var m: Shortcut.Modifiers = []
        if flags.contains(.control) { m.insert(.control) }
        if flags.contains(.option) { m.insert(.option) }
        if flags.contains(.shift) { m.insert(.shift) }
        if flags.contains(.command) { m.insert(.command) }
        return m
    }

    /// 押している修飾キーだけを表示する（キーを押す前の途中経過）
    private func showHeld(_ flags: NSEvent.ModifierFlags) {
        guard !recording.isSaved else { return }
        model.live = Shortcut(keyCode: 0, modifiers: modifiers(flags), keyLabel: "").display
    }

    private func keyDown(_ event: NSEvent) {
        guard !recording.isSaved else { return }
        let held = modifiers(event.modifierFlags)
        // 修飾キーなしのEscはキャンセルと同じ
        if event.keyCode == 53, held.isEmpty {
            cancel()
            return
        }
        let keyCode = UInt32(event.keyCode)
        let candidate = Shortcut(keyCode: keyCode, modifiers: held,
                                 keyLabel: Shortcut.label(keyCode: keyCode, characters: event.charactersIgnoringModifiers))
        recording.press(candidate,
                        reservedID: ShortcutConflicts.conflict(for: candidate, symbolic: Self.systemShortcuts()),
                        center: hotKeys)
        refresh()
    }

    private func arrived() {
        guard recording.arrived(center: hotKeys) else { return }
        refresh()
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.closeDelay) { [weak self] in
            self?.close()
        }
    }

    private func cancel() {
        recording.cancel(center: hotKeys)
        close()
    }

    private func turnOff() {
        recording.turnOff(center: hotKeys)
        close()
    }

    /// 状態から表示を作る。「現在」は保存済みの値なので、確かめ終わるまでは窓を開く前のまま
    private func refresh() {
        let s = L10n.current
        model.current = hotKeys.shortcut?.display
        switch recording.phase {
        case .waiting:
            model.message = s.recorderPrompt
            model.result = .waiting
        case .rejected(let candidate, let rejection):
            model.live = candidate.display
            model.result = .rejected
            switch rejection {
            case .needsModifier: model.message = s.recorderRejected
            case .reserved(let id): model.message = s.recorderReserved(candidate.display, s.systemShortcutName(id))
            case .taken: model.message = s.recorderTaken(candidate.display)
            }
        case .verifying(let candidate):
            model.live = candidate.display
            model.result = .verifying
            var message = s.recorderVerify(candidate.display)
            if ShortcutConflicts.isAppMenuProne(candidate) {
                message += "\n" + s.recorderMenuProne
            }
            model.message = message
        case .saved(let candidate):
            model.live = candidate.display
            model.result = .saved
            model.message = s.recorderVerified(candidate.display)
        }
    }

    /// macOSのキーボードショートカットの設定。読めなければnil（組み込みの一覧だけで判定する）
    private static func systemShortcuts() -> [String: Any]? {
        CFPreferencesCopyAppValue("AppleSymbolicHotKeys" as CFString, "com.apple.symbolichotkeys" as CFString)
            as? [String: Any]
    }

    private func close() {
        guard panel != nil else { return }
        hotKeys.interceptor = nil
        if let monitor { NSEvent.removeMonitor(monitor) }
        monitor = nil
        panel?.orderOut(nil)
        panel = nil
        previousApp?.activate()
        previousApp = nil
    }
}

final class RecorderModel: ObservableObject {
    enum Result {
        case waiting
        case rejected
        case verifying
        case saved
    }

    /// 保存済みの組み合わせ
    @Published var current: String?
    /// いま押している組み合わせ（途中経過を含む）
    @Published var live = ""
    @Published var message = ""
    @Published var result = Result.waiting
    var onDisable: () -> Void = {}
    var onCancel: () -> Void = {}
}

struct RecorderView: View {
    @ObservedObject var model: RecorderModel

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(L10n.current.recorderTitle)
                .font(.system(size: 15, weight: .semibold))
            Text(L10n.current.recorderCurrent(model.current))
                .font(.system(size: 13).monospacedDigit())
                .foregroundStyle(.secondary)

            // 押しているキーをその場で出す枠
            Text(model.live.isEmpty ? " " : model.live)
                .font(.system(size: 24, weight: .semibold).monospacedDigit())
                .frame(maxWidth: .infinity, minHeight: 48)
                .background(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(Color.white.opacity(0.08))
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .strokeBorder(frameColor, lineWidth: 1.5)
                )

            Text(model.message)
                .font(.system(size: 13))
                .foregroundStyle(messageColor)
                .fixedSize(horizontal: false, vertical: true)
            HStack {
                Spacer()
                Button(L10n.current.recorderDisable, action: model.onDisable)
                // どの段階でも、キャンセルは窓を開く前の組み合わせに戻す（確かめるまで保存しないため）
                Button(L10n.current.recorderCancel, action: model.onCancel)
            }
            .font(.system(size: 13))
        }
        .padding(18)
        .frame(width: 380)
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

    private var frameColor: Color {
        switch model.result {
        case .waiting: return Color.white.opacity(0.18)
        case .verifying: return Color.white.opacity(0.45)
        case .rejected: return LevelStyle.background(.critical)!.color
        case .saved: return Color(.sRGB, red: 80 / 255, green: 190 / 255, blue: 120 / 255, opacity: 1)
        }
    }

    private var messageColor: Color {
        switch model.result {
        case .waiting, .verifying, .saved: return .primary
        // 赤の札と同じ色は暗い面では文字として読みにくいので、明るめの赤にする
        case .rejected: return Color(.sRGB, red: 1, green: 0.55, blue: 0.5, opacity: 1)
        }
    }
}

/// キー入力を受けられる枠なしパネル
final class RecorderPanel: NSPanel {
    init(contentView: NSView) {
        super.init(contentRect: NSRect(x: 0, y: 0, width: 380, height: 200),
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
