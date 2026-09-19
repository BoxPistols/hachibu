import Foundation
import SlashstripCore

/// 基本表示の供給元。ほかの道具に頼らず、状態の枠1つ（"Opus5 1M xhigh · S2 W81 F55"）だけを出す。
///
/// - モデル名とeffort: statusLineのJSON（scripts/statusline.shが書く）があればそれ、無ければ最後の会話記録と~/.claude/settings.json
/// - 使用率: 利用者が有効にしていれば使用率API（5分ごと）、そうでなければstatusLineのrate_limits
final class BasicSource: DataSource {
    var onSlots: (([Slot]) -> Void)?
    private(set) var limits: [UsageLimit] = []
    private(set) var usageAsOf: Date?
    /// 直近のAPIの失敗。メニューに出す
    private(set) var apiFailure: UsageAPIClient.Failure?

    static let statusLineFile = FileManager.default.homeDirectoryForCurrentUser
        .appendingPathComponent("Library/Application Support/Slashstrip/statusline.json")
    private static let claudeDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(".claude")

    private var timer: Timer?
    private var lastSlots: [Slot] = []

    private var statusLine: StatusLine.Snapshot?
    private var statusLineModified: Date?
    private var statusLineLimits: [UsageLimit] = []
    private var statusLineLimitsAt: Date?

    private var transcriptModelID: String?
    private var transcriptModified: Date?
    private var lastTranscriptScan = Date.distantPast

    private var apiLimits: [UsageLimit] = []
    private var apiAt: Date?
    private var lastFetch = Date.distantPast
    private var fetching = false

    private static let tickInterval: TimeInterval = 2
    // 会話記録は全プロジェクトを見るので、頻繁には探さない
    private static let transcriptScanInterval: TimeInterval = 30
    // 表示のために頻繁に叩かない。失敗しても次の周期まで再試行しない
    private static let fetchInterval: TimeInterval = 300
    private static let transcriptTailBytes = 64 * 1024

    func start() {
        tick()
        let t = Timer(timeInterval: Self.tickInterval, repeats: true) { [weak self] _ in self?.tick() }
        RunLoop.main.add(t, forMode: .common)
        timer = t
    }

    /// 「使用率をAPIから取得」を切り替えたとき。有効にしたらすぐ取りに行く
    func usageAPISettingChanged() {
        lastFetch = .distantPast
        apiFailure = nil
        if Prefs.usageAPIConsent != true {
            apiLimits = []
            apiAt = nil
        }
        tick()
    }

    private func tick() {
        let now = Date()
        readStatusLine()
        if now.timeIntervalSince(lastTranscriptScan) >= Self.transcriptScanInterval {
            lastTranscriptScan = now
            readNewestTranscript()
        }
        maybeFetch(now: now)
        publish()
    }

    // MARK: - 読み取り

    private func readStatusLine() {
        let modified = (try? FileManager.default.attributesOfItem(atPath: Self.statusLineFile.path)[.modificationDate]) as? Date
        guard modified != statusLineModified else { return }
        statusLineModified = modified
        guard let obj = JSONFile.object(at: Self.statusLineFile) else { return }
        let parsed = StatusLine.parse(obj)
        statusLine = parsed
        // rate_limitsは最初の応答の前には入らない。直前の値を空で上書きしない
        if !parsed.limits.isEmpty {
            statusLineLimits = parsed.limits
            statusLineLimitsAt = modified
        }
    }

    /// 最後に書かれた会話記録の末尾から、最後に使われたモデルを読む
    private func readNewestTranscript() {
        let fm = FileManager.default
        let projects = Self.claudeDir.appendingPathComponent("projects")
        guard let dirs = try? fm.contentsOfDirectory(at: projects, includingPropertiesForKeys: nil) else { return }
        var newest: (URL, Date)?
        for dir in dirs {
            guard let files = try? fm.contentsOfDirectory(at: dir, includingPropertiesForKeys: [.contentModificationDateKey]) else { continue }
            for f in files where f.pathExtension == "jsonl" {
                guard let m = try? f.resourceValues(forKeys: [.contentModificationDateKey]).contentModificationDate else { continue }
                if newest == nil || m > newest!.1 { newest = (f, m) }
            }
        }
        guard let (file, modified) = newest, modified != transcriptModified else { return }
        transcriptModified = modified
        guard let handle = try? FileHandle(forReadingFrom: file) else { return }
        defer { try? handle.close() }
        let size = (try? handle.seekToEnd()) ?? 0
        try? handle.seek(toOffset: size > UInt64(Self.transcriptTailBytes) ? size - UInt64(Self.transcriptTailBytes) : 0)
        let text = String(decoding: handle.readDataToEndOfFile(), as: UTF8.self)
        if let id = ModelInfo.lastModelID(inTranscriptTail: text) {
            transcriptModelID = id
        }
    }

    private static func settings() -> (model: String?, effort: String?) {
        guard let obj = JSONFile.object(at: claudeDir.appendingPathComponent("settings.json")) else { return (nil, nil) }
        return (obj["model"] as? String, obj["effortLevel"] as? String)
    }

    // MARK: - 使用率API

    private func maybeFetch(now: Date) {
        guard Prefs.usageAPIConsent == true, !fetching, now.timeIntervalSince(lastFetch) >= Self.fetchInterval else { return }
        // 取得前に時刻を進めておく（失敗時に毎ティック叩き直さない）
        lastFetch = now
        fetching = true
        DispatchQueue.global(qos: .utility).async { [weak self] in
            let result = UsageAPIClient.fetch()
            DispatchQueue.main.async {
                guard let self else { return }
                self.fetching = false
                // 取得中に無効にされていたら、届いた値は使わずに捨てる
                guard Prefs.usageAPIConsent == true else { return }
                switch result {
                case .success(let limits):
                    self.apiLimits = limits
                    self.apiAt = Date()
                    self.apiFailure = nil
                case .failure(let failure):
                    if self.apiFailure != failure {
                        ActionLog.append(L10n.current.usageAPIFailed(failure.summary))
                    }
                    self.apiFailure = failure
                }
                self.publish()
            }
        }
    }

    // MARK: - 表示

    private func publish() {
        let settings = Self.settings()
        // statusLineの方が会話記録より新しければ、その時点のモデルとeffortを使う
        let statusLineIsNewer = statusLineModified.map { s in transcriptModified.map { s >= $0.addingTimeInterval(-60) } ?? true } ?? false
        let model = (statusLineIsNewer ? statusLine?.model : nil)
            ?? ModelInfo.displayName(transcriptModelID: transcriptModelID, settingsModel: settings.model)
        let effort = (statusLineIsNewer ? statusLine?.effort : nil) ?? settings.effort

        let useAPI = Prefs.usageAPIConsent == true && !apiLimits.isEmpty
        limits = useAPI ? apiLimits : statusLineLimits
        usageAsOf = useAPI ? apiAt : statusLineLimitsAt

        let stale = Usage.isStale(usageAsOf)
        var lines = limits.isEmpty ? [L10n.current.basicNoUsage] : limits.map { $0.line() }
        if stale, let asOf = usageAsOf { lines.append(L10n.current.usageAsOf(ResetFormatter.text(asOf))) }
        if let apiFailure, Prefs.usageAPIConsent == true {
            lines.append(L10n.current.usageAPIFailed(apiFailure.summary))
        }
        let slot = Slot(id: Slot.statusID,
                        text: Usage.statusText(model: model, effort: effort, limits: limits),
                        background: .idleBackground, foreground: .idleForeground,
                        help: lines.joined(separator: "\n"), dimmed: limits.isEmpty || stale)
        guard [slot] != lastSlots else { return }
        lastSlots = [slot]
        onSlots?([slot])
    }
}
