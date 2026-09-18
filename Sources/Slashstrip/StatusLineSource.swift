import Foundation
import SlashstripCore

/// Touch Bar連携スクリプトが無いMac向け。scripts/statusline.shが書いたstatusLineのJSONを読み、
/// モデル・effort・使用率を1個のステータスとして出す（コマンド送信は持たない）。
///
/// statusLineはClaude Codeの公式の拡張点で、使用率（rate_limits）もそこに入る。
/// Keychainの資格情報を読んだり、公開されていないエンドポイントを呼んだりはしない。
final class StatusLineSource: DataSource {
    var onSlots: (([Slot]) -> Void)?
    let sourceDescription = L10n.current.sourceStatusLine
    private(set) var limits: [UsageLimit] = []

    static let file = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent("Library/Application Support/Slashstrip/statusline.json")

    private var timer: Timer?
    private var lastModified: Date?
    private var lastSlots: [Slot] = []
    private var snapshot: StatusLine.Snapshot?

    private static let tickInterval: TimeInterval = 2

    func start() {
        tick()
        let t = Timer(timeInterval: Self.tickInterval, repeats: true) { [weak self] _ in self?.tick() }
        RunLoop.main.add(t, forMode: .common)
        timer = t
    }

    func perform(_ slot: Slot) {
        // 送り先を確定する仕組みをまだ持たないので、何も送らない
        ActionLog.append("\(slot.id): statusLineのみのモードではコマンドを送りません")
    }

    private func tick() {
        let modified = (try? FileManager.default.attributesOfItem(atPath: Self.file.path)[.modificationDate]) as? Date
        if modified != lastModified {
            lastModified = modified
            if let data = try? Data(contentsOf: Self.file),
               let obj = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] {
                let parsed = StatusLine.parse(obj)
                snapshot = parsed
                // rate_limitsは最初の応答の前には入らない。直前の値を空で上書きしない
                if !parsed.limits.isEmpty { limits = parsed.limits }
            }
        }
        publish()
    }

    private func publish() {
        let text = snapshot.map { Usage.statusText(model: $0.model, effort: $0.effort, limits: limits) }
            ?? L10n.statusPlaceholder
        let help = snapshot == nil ? L10n.current.statusLineMissing : limits.map { $0.line() }.joined(separator: "\n")
        let slots = [Slot(id: Widgets.status, text: text, background: .idleBackground, foreground: .idleForeground,
                          help: help.isEmpty ? nil : help, dimmed: snapshot == nil)]
        guard slots != lastSlots else { return }
        lastSlots = slots
        onSlots?(slots)
    }
}
