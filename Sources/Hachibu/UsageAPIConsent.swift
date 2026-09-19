import AppKit
import HachibuCore
import SwiftUI

/// 使用率APIを使うかを尋ねる窓。何を使い、どんなリスクがあるかを先に示し、利用者が選んだときだけ有効にする
final class UsageAPIConsent {
    private var panel: ConsentPanel?
    private var previousApp: NSRunningApplication?

    func show(completion: @escaping (Bool) -> Void) {
        guard panel == nil else {
            NSApp.activate(ignoringOtherApps: true)
            panel?.makeKeyAndOrderFront(nil)
            return
        }
        previousApp = NSWorkspace.shared.frontmostApplication
        let view = ConsentView(
            onEnable: { [weak self] in self?.finish(true, completion) },
            onDecline: { [weak self] in self?.finish(false, completion) }
        )
        let hosting = NSHostingView(rootView: view)
        let panel = ConsentPanel(contentView: hosting)
        panel.setContentSize(hosting.fittingSize)
        panel.center()
        self.panel = panel
        NSApp.activate(ignoringOtherApps: true)
        panel.makeKeyAndOrderFront(nil)
    }

    private func finish(_ enabled: Bool, _ completion: (Bool) -> Void) {
        panel?.orderOut(nil)
        panel = nil
        previousApp?.activate()
        previousApp = nil
        completion(enabled)
    }
}

struct ConsentView: View {
    /// 注意書きで引用している規約の出典
    static let termsURL = URL(string: "https://code.claude.com/docs/en/legal-and-compliance")!

    let onEnable: () -> Void
    let onDecline: () -> Void

    var body: some View {
        let s = L10n.current
        VStack(alignment: .leading, spacing: 12) {
            Text(s.consentTitle)
                .font(.system(size: 15, weight: .semibold))
            ForEach([s.consentWhat, s.consentRisk, s.consentPrivacy, s.consentWithout], id: \.self) { paragraph in
                Text(paragraph)
                    .font(.system(size: 13))
                    .fixedSize(horizontal: false, vertical: true)
            }
            Link(s.consentTerms, destination: Self.termsURL)
                .font(.system(size: 13))
            HStack {
                Spacer()
                Button(s.consentDecline, action: onDecline)
                    .keyboardShortcut(.cancelAction)
                Button(s.consentEnable, action: onEnable)
            }
            .font(.system(size: 13))
        }
        .padding(20)
        .frame(width: 460)
        .background(GlassSurface())
        .clipShape(RoundedRectangle(cornerRadius: Metrics.panelRadius, style: .continuous))
        .overlay(GlassRim(cornerRadius: Metrics.panelRadius))
    }
}

/// キー入力を受けられる枠なしパネル
final class ConsentPanel: NSPanel {
    init(contentView: NSView) {
        super.init(contentRect: NSRect(x: 0, y: 0, width: 460, height: 360),
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
