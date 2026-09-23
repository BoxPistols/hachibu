import AppKit
import HachibuCore
import UserNotifications

/// 新しい版の確認。1日に1回、GitHubの最新リリースを見る。
///
/// 送るのはUser-Agent（"Hachibu/版"）だけで、ログイン情報や使用率は送らない。
/// 新しい版が出ていれば、その版につき1回だけ通知し、メニューにダウンロードのページを開く項目を出す。
/// メニューの「新しい版を自動で確認」を切れば、自分で「今すぐ確認」を押したときだけ見に行く
final class UpdateChecker: NSObject, UNUserNotificationCenterDelegate {
    enum Failure: Error, Equatable {
        case http(Int)
        case network
        case badResponse

        var summary: String {
            switch self {
            case .http(let code): return "HTTP \(code)"
            case .network: return "network"
            case .badResponse: return "unexpected response"
            }
        }
    }

    /// 見つけた新しい版。メニューに出す
    private(set) var available: Update.Release?
    private(set) var lastFailure: Failure?

    static let currentVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
    private static let url = URL(string: "https://api.github.com/repos/BoxPistols/hachibu/releases/latest")!
    private static let checkInterval: TimeInterval = 24 * 60 * 60
    // 起動直後はほかの準備（使用率APIの同意の窓など）と重ねない
    private static let firstCheckDelay: TimeInterval = 10

    private var timer: Timer?
    private var checking = false

    /// .appとして組み立てずに動かすと（swift run）、通知の仕組みに触れた時点で落ちる
    private static let canNotify = Bundle.main.bundleIdentifier != nil

    func start() {
        if Self.canNotify { UNUserNotificationCenter.current().delegate = self }
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.firstCheckDelay) { [weak self] in self?.checkIfDue() }
        // スリープをまたいでも1日を越えたら確認する。周期そのものは短くし、実際に叩くかは前回の時刻で決める
        let t = Timer(timeInterval: 60 * 60, repeats: true) { [weak self] _ in self?.checkIfDue() }
        RunLoop.main.add(t, forMode: .common)
        timer = t
    }

    /// 「新しい版を自動で確認」を入れたときは、すぐに一度確認する
    func autoCheckChanged() {
        if Prefs.updateAutoCheck {
            Prefs.lastUpdateCheck = nil
            checkIfDue()
        }
    }

    private func checkIfDue() {
        guard Prefs.updateAutoCheck else { return }
        if let last = Prefs.lastUpdateCheck, Date().timeIntervalSince(last) < Self.checkInterval { return }
        check(manual: false)
    }

    /// manualは「今すぐ確認」。結果が最新でも失敗でも、窓で知らせる
    func check(manual: Bool) {
        guard !checking else { return }
        checking = true
        Prefs.lastUpdateCheck = Date()
        var req = URLRequest(url: Self.url, timeoutInterval: 15)
        req.setValue("application/vnd.github+json", forHTTPHeaderField: "Accept")
        req.setValue("Hachibu/\(Self.currentVersion)", forHTTPHeaderField: "User-Agent")
        URLSession.shared.dataTask(with: req) { data, response, error in
            let result: Result<Update.Release, Failure>
            if error != nil || response == nil {
                result = .failure(.network)
            } else if let http = response as? HTTPURLResponse, http.statusCode != 200 {
                result = .failure(.http(http.statusCode))
            } else if let data, let obj = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any],
                      let release = Update.parseLatest(obj) {
                result = .success(release)
            } else {
                result = .failure(.badResponse)
            }
            DispatchQueue.main.async { [weak self] in self?.finish(result, manual: manual) }
        }.resume()
    }

    private func finish(_ result: Result<Update.Release, Failure>, manual: Bool) {
        checking = false
        let s = L10n.current
        switch result {
        case .success(let release):
            lastFailure = nil
            let newer = Update.isNewer(release.version, than: Self.currentVersion)
            available = newer ? release : nil
            if Update.shouldNotify(release, current: Self.currentVersion, notified: Prefs.notifiedUpdateVersion) {
                Prefs.notifiedUpdateVersion = release.version
                ActionLog.append(s.logUpdateAvailable(release.version))
                notify(release)
            }
            if manual {
                if newer {
                    NSWorkspace.shared.open(release.url)
                } else {
                    alert(s.updateUpToDate(Self.currentVersion))
                }
            }
        case .failure(let failure):
            if lastFailure != failure { ActionLog.append(s.updateCheckFailed(failure.summary)) }
            lastFailure = failure
            if manual { alert(s.updateCheckFailed(failure.summary)) }
        }
    }

    // MARK: - 知らせ方

    private func notify(_ release: Update.Release) {
        guard Self.canNotify else { return }
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert]) { granted, _ in
            // 通知を許可されなくても、メニューの項目で気づけるので何もしない
            guard granted else { return }
            let content = UNMutableNotificationContent()
            content.title = L10n.current.updateNotificationTitle(release.version)
            content.body = L10n.current.updateNotificationBody
            content.userInfo = ["url": release.url.absoluteString]
            center.add(UNNotificationRequest(identifier: "update-\(release.version)", content: content, trigger: nil))
        }
    }

    private func alert(_ text: String) {
        let a = NSAlert()
        a.messageText = text
        a.alertStyle = .informational
        NSApp.activate(ignoringOtherApps: true)
        a.runModal()
    }

    /// 通知を押したらダウンロードのページを開く
    func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse,
                                withCompletionHandler completionHandler: @escaping () -> Void) {
        if let s = response.notification.request.content.userInfo["url"] as? String, let url = URL(string: s) {
            DispatchQueue.main.async { NSWorkspace.shared.open(url) }
        }
        completionHandler()
    }

    /// アプリが前面にあるときも通知を出す（帯とメニューバーだけのアプリで、前面かどうかを利用者は意識しない）
    func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .list])
    }
}
