import Foundation

/// タップ時のスクリプト実行。結果は必ず操作ログに残す。
/// BTTではスクリプトの標準出力がどこにも出ず、「何も送らなかった理由」が見えなかった。
enum ScriptRunner {
    /// 環境変数SLASHSTRIP_DRY_RUN=1のときは実行せず、何を実行するかだけをログに書く（動作確認用）
    private static let dryRun = ProcessInfo.processInfo.environment["SLASHSTRIP_DRY_RUN"] == "1"

    static func run(_ executable: String, _ args: [String], label: String) {
        let commandLine = ([executable] + args).joined(separator: " ")
        if dryRun {
            ActionLog.append("\(label) dry-run: \(commandLine)")
            return
        }
        let p = Process()
        p.executableURL = URL(fileURLWithPath: executable)
        p.arguments = args
        p.standardInput = FileHandle.nullDevice
        let pipe = Pipe()
        p.standardOutput = pipe
        p.standardError = pipe
        do {
            try p.run()
        } catch {
            ActionLog.append("\(label) 起動失敗: \(error.localizedDescription)")
            return
        }
        // 出力を読み切ってから終了を待つ（逆順だとパイプが詰まったときに双方が待ち合う）
        DispatchQueue.global(qos: .utility).async {
            let data = pipe.fileHandleForReading.readDataToEndOfFile()
            p.waitUntilExit()
            let out = String(decoding: data, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines)
            ActionLog.append("\(label) exit=\(p.terminationStatus)\(out.isEmpty ? "" : " " + out)")
        }
    }

    /// 常駐デーモン用。待たずに切り離す（このアプリを終了しても、デーモンは読まれなくなれば自分で止まる）
    static func spawnDetached(_ executable: String, _ args: [String], logTo log: URL) {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: executable)
        p.arguments = args
        p.standardInput = FileHandle.nullDevice
        if !FileManager.default.fileExists(atPath: log.path) {
            FileManager.default.createFile(atPath: log.path, contents: nil)
        }
        if let handle = try? FileHandle(forWritingTo: log) {
            handle.seekToEndOfFile()
            p.standardOutput = handle
            p.standardError = handle
        } else {
            p.standardOutput = FileHandle.nullDevice
            p.standardError = FileHandle.nullDevice
        }
        do {
            try p.run()
            ActionLog.append("描画デーモンを起動しました（pid=\(p.processIdentifier)）")
        } catch {
            ActionLog.append("描画デーモンの起動に失敗: \(error.localizedDescription)")
        }
    }
}

/// ~/Library/Logs/Slashstrip/actions.logに1行ずつ追記する。256KBを超えたら後半だけ残す。
enum ActionLog {
    static let url: URL = {
        let dir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/Logs/Slashstrip")
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir.appendingPathComponent("actions.log")
    }()

    private static let queue = DispatchQueue(label: "slashstrip.actionlog")
    private static let maxBytes = 256 * 1024
    private static let stamp: DateFormatter = {
        let f = DateFormatter()
        f.locale = Locale(identifier: "en_US_POSIX")
        f.dateFormat = "yyyy-MM-dd HH:mm:ss"
        return f
    }()

    static func append(_ message: String) {
        let line = "\(stamp.string(from: Date())) \(message)\n"
        queue.async {
            let fm = FileManager.default
            if let size = (try? fm.attributesOfItem(atPath: url.path)[.size]) as? Int, size > maxBytes,
               let data = try? Data(contentsOf: url) {
                try? data.suffix(maxBytes / 2).write(to: url)
            }
            if !fm.fileExists(atPath: url.path) {
                fm.createFile(atPath: url.path, contents: nil)
            }
            guard let handle = try? FileHandle(forWritingTo: url) else { return }
            defer { try? handle.close() }
            handle.seekToEndOfFile()
            handle.write(Data(line.utf8))
        }
    }
}
