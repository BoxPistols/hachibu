import Foundation
import SlashstripCore

/// 撮影と見た目の確認用。架空の値だけを出し、押しても何も送らない。
/// SLASHSTRIP_SOURCE=demoで起動する。SLASHSTRIP_DEMO=permissionで許可待ち、criticalで使用率が赤の場面になる
final class DemoSource: DataSource {
    var onSlots: (([Slot]) -> Void)?
    let sourceDescription = "Demo"
    private(set) var limits: [UsageLimit] = []

    private let scene = ProcessInfo.processInfo.environment["SLASHSTRIP_DEMO"] ?? ""

    private static let busy = RGBA(r: 38, g: 102, b: 168, a: 255)
    private static let waiting = RGBA(r: 214, g: 138, b: 30, a: 255)
    private static let button = RGBA(r: 72, g: 72, b: 72, a: 255)

    func start() {
        let now = Date()
        let fiveHour = scene == "critical" ? 93 : 42
        limits = [
            UsageLimit(kind: .fiveHour, percent: fiveHour, resetsAt: now.addingTimeInterval(2 * 3600)),
            UsageLimit(kind: .weekly, percent: 76, resetsAt: now.addingTimeInterval(3 * 86400)),
        ]
        let usage = Usage.statusText(model: nil, effort: nil, limits: limits)
        var slots: [Slot]
        if scene == "permission" {
            let perm = L10n.current.language == .ja ? ["許可", "常に許可", "拒否"] : ["Allow", "Always allow", "Deny"]
            slots = [Slot(id: Widgets.status, text: (L10n.current.language == .ja ? "許可待ち · " : "Permission · ") + usage,
                          background: Self.waiting, foreground: .white)]
            slots += zip(["perm-allow", "perm-always", "perm-reject"], perm).map {
                Slot(id: $0.0, text: $0.1, background: Self.button, foreground: .white)
            }
        } else {
            slots = [Slot(id: Widgets.status, text: "Opus5 1M xhigh · " + usage, background: Self.busy, foreground: .white,
                          help: limits.map { $0.line() }.joined(separator: "\n"))]
            slots += ["review", "model", "effort", "usage", "clear", "compact"].enumerated().map {
                Slot(id: "cmd-\($0.offset)", text: $0.element, background: Self.button, foreground: .white)
            }
        }
        onSlots?(slots)
    }

    func perform(_ slot: Slot) {
        ActionLog.append("demo: \(slot.id) [\(slot.text)]")
    }

    func sessions() -> (list: [SessionInfo], focusedID: String?) {
        let list = [
            SessionInfo(id: "demo-1", project: "example-api", state: .busy, tool: "Bash", termGUID: nil,
                        hostBundle: nil, pid: nil, processStart: nil),
            SessionInfo(id: "demo-2", project: "example-web", state: .waitingPermission, tool: nil, termGUID: nil,
                        hostBundle: nil, pid: nil, processStart: nil),
            SessionInfo(id: "demo-3", project: "example-docs", state: .idle, tool: nil, termGUID: nil,
                        hostBundle: nil, pid: nil, processStart: nil),
        ]
        return (list, "demo-1")
    }
}
