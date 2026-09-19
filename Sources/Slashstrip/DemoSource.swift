import Foundation
import SlashstripCore

/// 撮影と見た目の確認用。架空の値だけを出す。
/// SLASHSTRIP_SOURCE=demoで起動する。SLASHSTRIP_DEMO=criticalで5時間枠が赤の場面になる
final class DemoSource: DataSource {
    var onSlots: (([Slot]) -> Void)?
    private(set) var limits: [UsageLimit] = []

    private let scene = ProcessInfo.processInfo.environment["SLASHSTRIP_DEMO"] ?? ""

    func start() {
        let now = Date()
        let fiveHour = scene == "critical" ? 93 : 42
        limits = [
            UsageLimit(kind: .fiveHour, percent: fiveHour, resetsAt: now.addingTimeInterval(2 * 3600)),
            UsageLimit(kind: .weekly, percent: 76, resetsAt: now.addingTimeInterval(3 * 86400)),
            UsageLimit(kind: .model("Fable"), percent: 55, resetsAt: now.addingTimeInterval(2 * 86400)),
        ]
        // BasicSourceと同じ色（平常の灰色）
        let slot = Slot(id: Slot.statusID,
                        text: Usage.statusText(model: "Opus5 1M", effort: "xhigh", limits: limits),
                        background: .idleBackground, foreground: .idleForeground,
                        help: limits.map { $0.line() }.joined(separator: "\n"))
        onSlots?([slot])
    }
}
