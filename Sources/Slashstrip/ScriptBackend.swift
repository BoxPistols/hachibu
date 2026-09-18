import Foundation
import SlashstripCore

/// ~/.claude/bttのTouch Bar連携スクリプトを、BetterTouchToolの代わりに読む供給元。
///
/// 表示内容は常駐デーモンcc-render.pyがrender/<name>.jsonに書いている。ここではそれを読むだけで、
/// どのボタンを出すか・送ってよいかの判断は一切複製しない（判断を二重に持つと、表示と送信先が食い違って別のセッションに文字が入る）。
/// タップ時もBTTのウィジェットと同じスクリプトを同じ引数で呼ぶ。
final class ScriptBackend: DataSource {
    var onSlots: (([Slot]) -> Void)?
    let sourceDescription = L10n.current.sourceScripts
    private(set) var limits: [UsageLimit] = []
    private(set) var usageAsOf: Date?

    private let base: URL
    private let render: URL
    private let python = "/usr/bin/python3"
    private var timer: Timer?
    private var lastSlots: [Slot] = []
    private var lastKeepalive = Date.distantPast
    private var lastSpawn = Date.distantPast

    private static let pollInterval: TimeInterval = 0.5
    // cc-render.pyは.seenが15秒触られないと自分で終了する。その内側で触り続ける
    private static let keepaliveInterval: TimeInterval = 2
    // デーモンは変化が無くても30ティック（約30秒）ごとに書き直す。それを超えて古ければ止まっている
    private static let staleAfter: TimeInterval = 45

    static func detect() -> ScriptBackend? {
        let base = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".claude/btt")
        guard FileManager.default.fileExists(atPath: base.appendingPathComponent("cc-render.py").path) else {
            return nil
        }
        return ScriptBackend(base: base)
    }

    private init(base: URL) {
        self.base = base
        self.render = base.appendingPathComponent("render")
    }

    func start() {
        tick()
        let t = Timer(timeInterval: Self.pollInterval, repeats: true) { [weak self] _ in self?.tick() }
        // メニューを開いている間も更新を止めない
        RunLoop.main.add(t, forMode: .common)
        timer = t
    }

    func perform(_ slot: Slot) {
        guard let args = Widgets.arguments(for: slot.id, base: base) else {
            ActionLog.append("\(slot.id): 対応するスクリプトがありません")
            return
        }
        ScriptRunner.run(python, args, label: "\(slot.id) [\(slot.text)]")
    }

    // MARK: - セッション一覧

    func sessions() -> (list: [SessionInfo], focusedID: String?) {
        let dir = base.appendingPathComponent("sessions")
        let names = (try? FileManager.default.contentsOfDirectory(atPath: dir.path)) ?? []
        let list = names.filter { $0.hasSuffix(".json") }.compactMap { name -> SessionInfo? in
            guard let json = JSONFile.object(at: dir.appendingPathComponent(name)),
                  let info = SessionInfo.parse(json) else { return nil }
            // 描画デーモンが毎秒掃除するが、その間に終わったものは自分でも弾く
            if let pid = info.pid, pid > 0, kill(pid, 0) != 0, errno != EPERM { return nil }
            return info
        }
        // フォーカス追従デーモン（iTerm2側）が書く、前面のセッションのGUID
        let focusedGUID = (try? String(contentsOf: base.appendingPathComponent("focus"), encoding: .utf8))?
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let focused = list.first { $0.termGUID != nil && $0.termGUID == focusedGUID }
        return (SessionInfo.ordered(list), focused?.id)
    }

    func focus(_ session: SessionInfo) {
        SessionFocus.focus(session)
    }

    // MARK: - 定期処理

    private func tick() {
        let now = Date()
        if now.timeIntervalSince(lastKeepalive) >= Self.keepaliveInterval {
            lastKeepalive = now
            keepDaemonAlive(now: now)
        }
        let usage = JSONFile.object(at: base.appendingPathComponent("usage.json"))
        let usageAPI = JSONFile.object(at: base.appendingPathComponent("usage-api.json"))
        limits = Widgets.limits(usage: usage, usageAPI: usageAPI)
        // 5時間枠と週枠はstatusLineが書くusage.jsonの時刻、無ければモデル別枠の取得時刻
        usageAsOf = [usage, usageAPI].lazy.compactMap { ($0?["updated_at"] as? NSNumber)?.doubleValue }
            .first.map { Date(timeIntervalSince1970: $0) }
        let slots = readSlots(now: now)
        if slots != lastSlots {
            lastSlots = slots
            onSlots?(slots)
        }
    }

    /// cc-widget.shのstatusと同じ役目。読まれている印を付け、デーモンが死んでいれば起こす。
    private func keepDaemonAlive(now: Date) {
        let seen = render.appendingPathComponent(".seen")
        let fm = FileManager.default
        try? fm.createDirectory(at: render, withIntermediateDirectories: true)
        if fm.fileExists(atPath: seen.path) {
            try? fm.setAttributes([.modificationDate: now], ofItemAtPath: seen.path)
        } else {
            fm.createFile(atPath: seen.path, contents: nil)
        }

        if daemonAlive() { return }
        // 起動直後はpidファイルがまだ無い。二重起動はデーモン側のflockが弾くが、無駄に湧かせない
        guard now.timeIntervalSince(lastSpawn) > 5 else { return }
        lastSpawn = now
        ScriptRunner.spawnDetached(python, [base.appendingPathComponent("cc-render.py").path],
                                   logTo: base.appendingPathComponent("render.log"))
    }

    private func daemonAlive() -> Bool {
        let pidFile = render.appendingPathComponent("daemon.pid")
        guard let raw = try? String(contentsOf: pidFile, encoding: .utf8),
              let pid = Int32(raw.trimmingCharacters(in: .whitespacesAndNewlines)), pid > 0 else {
            return false
        }
        // EPERMは「存在するが自分のものではない」なので生きている扱い
        return kill(pid, 0) == 0 || errno == EPERM
    }

    // MARK: - 読み取り

    private func readSlots(now: Date) -> [Slot] {
        let names = (try? FileManager.default.contentsOfDirectory(atPath: render.path)) ?? []
        let widgets = Set(names.filter { $0.hasSuffix(".json") }.map { String($0.dropLast(5)) })
        return Widgets.ordered(widgets).compactMap { readSlot($0, now: now) }
    }

    private func readSlot(_ name: String, now: Date) -> Slot? {
        let url = render.appendingPathComponent(name + ".json")
        guard let json = JSONFile.object(at: url) else {
            return name == Widgets.status ? Widgets.placeholderStatus() : nil
        }
        guard var slot = Widgets.slot(name: name, json: json) else { return nil }
        if name == Widgets.status {
            let modified = (try? FileManager.default.attributesOfItem(atPath: url.path)[.modificationDate]) as? Date
            let daemonStopped = modified.map { now.timeIntervalSince($0) > Self.staleAfter } ?? true
            slot = Widgets.idleStatus(slot, limits: limits, asOf: usageAsOf, now: now)
            slot.dimmed = slot.dimmed || daemonStopped
            var lines = (daemonStopped ? [L10n.current.staleNotice] : []) + limits.map { $0.line() }
            if Usage.isStale(usageAsOf, now: now), let asOf = usageAsOf {
                lines.append(L10n.current.usageAsOf(ResetFormatter.text(asOf, now: now)))
            }
            slot.help = lines.isEmpty ? nil : lines.joined(separator: "\n")
        }
        return slot
    }
}
