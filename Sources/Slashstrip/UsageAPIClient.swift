import Foundation
import SlashstripCore

/// 使用率APIの呼び出し。利用者が「使用率をAPIから取得」を有効にしたときだけ使う。
///
/// Claude Codeがキーチェーンに保存したOAuthトークンを/usr/bin/securityで読み、Anthropicの使用率のエンドポイントへGETする。
/// トークンはこの関数の中でしか使わず、保存もログへの出力もしない。User-AgentはClaude Codeを名乗らない。
enum UsageAPIClient {
    enum Failure: Error, Equatable {
        case noCredentials
        case http(Int)
        case network
        case badResponse

        /// メニューに出す短い理由
        var summary: String {
            switch self {
            case .noCredentials: return "no Claude Code login in the keychain"
            case .http(let code): return "HTTP \(code)"
            case .network: return "network"
            case .badResponse: return "unexpected response"
            }
        }
    }

    private static let url = URL(string: "https://api.anthropic.com/api/oauth/usage")!
    private static let keychainService = "Claude Code-credentials"

    /// 呼び出し側でメインスレッド以外から呼ぶ（キーチェーンの確認ダイアログを待つことがある）
    static func fetch() -> Result<[UsageLimit], Failure> {
        guard let token = accessToken() else { return .failure(.noCredentials) }
        var req = URLRequest(url: url, timeoutInterval: 10)
        req.setValue("Bearer " + token, forHTTPHeaderField: "Authorization")
        req.setValue("oauth-2025-04-20", forHTTPHeaderField: "anthropic-beta")
        let version = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
        req.setValue("Slashstrip/\(version)", forHTTPHeaderField: "User-Agent")

        let done = DispatchSemaphore(value: 0)
        var result: Result<[UsageLimit], Failure> = .failure(.network)
        URLSession.shared.dataTask(with: req) { data, response, error in
            defer { done.signal() }
            guard error == nil, let http = response as? HTTPURLResponse else { return }
            guard http.statusCode == 200 else {
                result = .failure(.http(http.statusCode))
                return
            }
            guard let data, let obj = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any] else {
                result = .failure(.badResponse)
                return
            }
            let limits = UsageAPI.parse(obj)
            result = limits.isEmpty ? .failure(.badResponse) : .success(limits)
        }.resume()
        done.wait()
        return result
    }

    /// 確認ダイアログが出た場合に備え、外から殺さない（殺すと応答先の無いダイアログが画面に残る）
    private static func accessToken() -> String? {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/security")
        p.arguments = ["find-generic-password", "-s", keychainService, "-w"]
        let out = Pipe()
        p.standardOutput = out
        p.standardError = FileHandle.nullDevice
        p.standardInput = FileHandle.nullDevice
        do { try p.run() } catch { return nil }
        let data = out.fileHandleForReading.readDataToEndOfFile()
        p.waitUntilExit()
        guard p.terminationStatus == 0,
              let obj = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any],
              let oauth = obj["claudeAiOauth"] as? [String: Any],
              let token = oauth["accessToken"] as? String, !token.isEmpty else {
            return nil
        }
        return token
    }
}
