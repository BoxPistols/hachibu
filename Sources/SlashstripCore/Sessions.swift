import Foundation

/// hooksが書いたセッション状態ファイル（~/.claude/btt/sessions/<id>.json）の1件。
public struct SessionInfo: Equatable {
    public enum State: Equatable {
        case busy
        case waitingPermission
        case waitingInput
        case idle
    }

    public let id: String
    public let project: String
    public let state: State
    public let tool: String?
    /// iTerm2のセッションGUID。デスクトップアプリ駆動のセッションには無い
    public let termGUID: String?
    /// ターミナル以外（デスクトップアプリ）で動いているとき、そのアプリのbundle id
    public let hostBundle: String?
    public let pid: Int32?
    public let processStart: Date?

    public init(id: String, project: String, state: State, tool: String?, termGUID: String?,
                hostBundle: String?, pid: Int32?, processStart: Date?) {
        self.id = id
        self.project = project
        self.state = state
        self.tool = tool
        self.termGUID = termGUID
        self.hostBundle = hostBundle
        self.pid = pid
        self.processStart = processStart
    }

    /// 状態ファイル1件を読む。session_idもcwdも無いものは捨てる
    public static func parse(_ json: [String: Any]) -> SessionInfo? {
        guard let id = json["session_id"] as? String, !id.isEmpty else { return nil }
        let cwd = (json["cwd"] as? String) ?? ""
        let project = cwd.isEmpty ? id.prefix(8).description : URL(fileURLWithPath: cwd).lastPathComponent

        let state: State
        switch json["state"] as? String {
        case "busy": state = .busy
        case "waiting": state = (json["waiting_kind"] as? String) == "permission" ? .waitingPermission : .waitingInput
        default: state = .idle
        }

        // デスクトップアプリ駆動（peers）のterm_guidは起動元から受け継いだ値で、実際の表示場所ではない
        let isPeers = (json["ui"] as? String) == "peers"
        let guid = (json["term_guid"] as? String).flatMap { $0.isEmpty ? nil : $0 }
        let tool = (json["tool"] as? String).flatMap { $0.isEmpty ? nil : $0 }
        return SessionInfo(id: id, project: project, state: state, tool: tool,
                           termGUID: isPeers ? nil : guid,
                           hostBundle: isPeers ? "com.anthropic.claudefordesktop" : (json["host_bundle"] as? String),
                           pid: (json["pid"] as? NSNumber)?.int32Value,
                           processStart: parseProcessStart(json["pid_start"] as? String))
    }

    /// `ps -o lstart=` の出力（例 "Fri Sep 18 21:30:37 2026"、日が1桁だと空白が2つ）
    public static func parseProcessStart(_ s: String?) -> Date? {
        guard let s else { return nil }
        let normalized = s.split(separator: " ").joined(separator: " ")
        let f = DateFormatter()
        f.locale = Locale(identifier: "en_US_POSIX")
        f.dateFormat = "EEE MMM d HH:mm:ss yyyy"
        return f.date(from: normalized)
    }

    /// 一覧の並び。プロセスの開始が古い順で固定し、状態が変わっても番号が入れ替わらないようにする。
    /// （状態の緊急度で並べると、押すたびに順番が変わって行き先が読めない）
    public static func ordered(_ sessions: [SessionInfo]) -> [SessionInfo] {
        sessions.sorted { a, b in
            switch (a.processStart, b.processStart) {
            case let (x?, y?) where x != y: return x < y
            case (_?, nil): return true
            case (nil, _?): return false
            default: break
            }
            if a.pid != b.pid { return (a.pid ?? .max) < (b.pid ?? .max) }
            return a.id < b.id
        }
    }

    /// 一覧の1行: "1. example-project（実行中・Bash）" / "1. example-project (running, Bash)"
    public func menuTitle(number: Int, strings: Strings = L10n.current) -> String {
        let detail = [strings.sessionStateName(state), state == .busy ? tool : nil].compactMap { $0 }
        return strings.sessionTitle(number, project, detail)
    }
}
