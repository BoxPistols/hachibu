// App Store版（HACHIBU_APP_STORE=1）には入れない。キーチェーンの読み取りと通信はGitHub版だけで行う
#if !APP_STORE
import AppKit
import CryptoKit
import HachibuCore
import UserNotifications

/// 新しいバージョンの確認と、その場での更新。1日に1回、GitHubのリリースの一覧を見る。
///
/// 送るのはUser-Agent（"Hachibu/バージョン"）だけで、ログイン情報や使用率は送らない。
/// 新しいバージョンが出ていれば、裏で添付のzipを取って検査し、済んだらそのバージョンにつき1回だけ通知する。
/// 通知かメニューの項目を押すと、アプリを置き換えて開き直す。
/// 置き換えられない場所（ディスクイメージの中、書き込めないフォルダ）から動いているときは、従来どおりダウンロードのページを開く。
/// メニューの自動確認を切れば、自分で「今すぐ確認」を押したときだけ見に行く
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

    /// 見つけた新しいバージョン。メニューに出す
    private(set) var available: Update.Release?
    private(set) var lastFailure: Failure?
    /// zipの取得と検査の最中
    private(set) var preparing = false

    static let currentVersion = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "dev"
    private static let url = URL(string: "https://api.github.com/repos/BoxPistols/hachibu/releases?per_page=30")!
    private static let checkInterval: TimeInterval = 24 * 60 * 60
    // 起動直後はほかの準備（使用率APIの同意の窓など）と重ねない
    private static let firstCheckDelay: TimeInterval = 10

    private var timer: Timer?
    private var checking = false
    /// 取得と検査を済ませ、置き換えを待っているアプリ
    private var staged: (version: String, app: URL, dir: URL)?
    /// 準備ができたら置き換えまで進める（通知やメニューを押したが、まだ取得中だったとき）
    private var installWhenReady = false

    /// .appとして組み立てずに動かすと（swift run）、通知の仕組みに触れた時点で落ちる
    private static let canNotify = Bundle.main.bundleIdentifier != nil

    /// その場で置き換えられるか。ディスクイメージの中や、隔離属性のあるアプリを移動せずに開いたときの仮の場所
    /// （App Translocation）、書き込めないフォルダからは置き換えられないので、ダウンロードのページを開く形に戻す
    static var canInstallInPlace: Bool {
        let app = Bundle.main.bundleURL
        guard Bundle.main.bundleIdentifier != nil, app.pathExtension == "app",
              !app.path.contains("/AppTranslocation/") else { return false }
        return FileManager.default.isWritableFile(atPath: app.deletingLastPathComponent().path)
    }

    func start() {
        if Self.canNotify { UNUserNotificationCenter.current().delegate = self }
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.firstCheckDelay) { [weak self] in self?.checkIfDue() }
        // スリープをまたいでも1日を越えたら確認する。周期そのものは短くし、実際に叩くかは前回の時刻で決める
        let t = Timer(timeInterval: 60 * 60, repeats: true) { [weak self] _ in self?.checkIfDue() }
        RunLoop.main.add(t, forMode: .common)
        timer = t
    }

    /// 自動確認を入れたときは、すぐに一度確認する
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
        var req = URLRequest(url: Self.url, timeoutInterval: 15)
        req.setValue("application/vnd.github+json", forHTTPHeaderField: "Accept")
        req.setValue("Hachibu/\(Self.currentVersion)", forHTTPHeaderField: "User-Agent")
        URLSession.shared.dataTask(with: req) { [weak self] data, response, error in
            let status = error == nil ? (response as? HTTPURLResponse)?.statusCode : nil
            let outcome = Update.outcome(status: status, data: data, current: Self.currentVersion)
            let failure: Failure? = switch outcome {
            case .failed: status.map { $0 == 200 ? .badResponse : .http($0) } ?? .network
            default: nil
            }
            DispatchQueue.main.async { self?.finish(outcome, failure: failure, manual: manual) }
        }.resume()
    }

    private func finish(_ outcome: Update.Outcome, failure: Failure?, manual: Bool) {
        checking = false
        let s = L10n.current
        switch outcome {
        case .newer(let release):
            // 前回の時刻は成功したときだけ進める。失敗で進めると、通信できなかった日の確認が丸1日飛ぶ
            Prefs.lastUpdateCheck = Date()
            lastFailure = nil
            available = release
            if Update.shouldNotify(release, current: Self.currentVersion, notified: Prefs.notifiedUpdateVersion) {
                ActionLog.append(s.logUpdateAvailable(release.version))
            }
            if Self.canInstallInPlace, release.assetURL != nil {
                if manual { ask(release) }
                prepare(release)
            } else {
                notifyOnce(release, ready: false)
                if manual { NSWorkspace.shared.open(release.url) }
            }
        case .upToDate:
            Prefs.lastUpdateCheck = Date()
            lastFailure = nil
            available = nil
            discardStaged()
            if manual { alert(s.updateUpToDate(Self.currentVersion)) }
        case .failed:
            // 既に知っている新しいバージョンの知らせは残す
            let failure = failure ?? .badResponse
            if lastFailure != failure { ActionLog.append(s.updateCheckFailed(failure.summary)) }
            lastFailure = failure
            if manual { alert(s.updateCheckFailed(failure.summary)) }
        }
    }

    // MARK: - その場での更新

    /// メニューの項目、通知、手で確認したときの窓から呼ぶ。準備が済んでいれば置き換え、まだなら済んだときに置き換える
    func install() {
        guard let release = available else { return }
        guard Self.canInstallInPlace, release.assetURL != nil else {
            NSWorkspace.shared.open(release.url)
            return
        }
        if let staged, staged.version == release.version {
            replaceAndRelaunch(staged.app, version: release.version)
        } else {
            installWhenReady = true
            prepare(release)
        }
    }

    /// 裏でzipを取り、展開して検査し、置き換えを待つ場所に置く
    private func prepare(_ release: Update.Release) {
        if let staged, staged.version == release.version, FileManager.default.fileExists(atPath: staged.app.path) {
            finishPreparing(release, result: .success(staged.app))
            return
        }
        guard !preparing, let asset = release.assetURL, let bundleID = Bundle.main.bundleIdentifier else { return }
        preparing = true
        discardStaged()
        let target = Bundle.main.bundleURL
        DispatchQueue.global(qos: .utility).async { [weak self] in
            let result = Result { try Self.download(asset, release: release, bundleID: bundleID, near: target) }
            DispatchQueue.main.async {
                self?.preparing = false
                self?.finishPreparing(release, result: result)
            }
        }
    }

    private func finishPreparing(_ release: Update.Release, result: Result<URL, Error>) {
        let s = L10n.current
        switch result {
        case .success(let app):
            if staged?.app != app { staged = (release.version, app, app.deletingLastPathComponent()) }
            ActionLog.append(s.logUpdateReady(release.version))
            if Self.isQuarantined(app) { ActionLog.append(s.logUpdateQuarantined) }
            if installWhenReady {
                installWhenReady = false
                replaceAndRelaunch(app, version: release.version)
            } else {
                notifyOnce(release, ready: true)
            }
        case .failure(let error):
            let reason = (error as? InstallError)?.summary ?? error.localizedDescription
            ActionLog.append(s.updateInstallFailed(reason))
            if installWhenReady {
                installWhenReady = false
                failed(reason, release: release)
            } else {
                // 裏の準備に失敗しても、知らせは出す。押したときにもう一度取得を試みる
                notifyOnce(release, ready: false)
            }
        }
    }

    struct InstallError: Error {
        let summary: String
        init(_ summary: String) { self.summary = summary }
    }

    /// zipを取り、SHA-256、バンドルID、バージョン、署名を確かめる。
    /// 添付の名前にバージョンが入っていないので、取り違えたzipで上書きしないよう中身のInfo.plistで確かめる
    private static func download(_ asset: URL, release: Update.Release, bundleID: String, near target: URL) throws -> URL {
        let fm = FileManager.default
        var req = URLRequest(url: asset, timeoutInterval: 120)
        req.setValue("Hachibu/\(currentVersion)", forHTTPHeaderField: "User-Agent")
        let zip = try syncDownload(req)
        defer { try? fm.removeItem(at: zip) }

        if let expected = release.sha256 {
            guard let data = try? Data(contentsOf: zip, options: .mappedIfSafe) else { throw InstallError("could not read the zip") }
            let actual = SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
            guard actual == expected else { throw InstallError("SHA-256 does not match") }
        }

        // 置き換え先と同じボリュームの一時フォルダに置く。別のボリュームだと置き換えがコピーになり、途中で止まると壊れる
        let dir = try fm.url(for: .itemReplacementDirectory, in: .userDomainMask, appropriateFor: target, create: true)
        do {
            guard run("/usr/bin/ditto", ["-x", "-k", zip.path, dir.path]) == 0 else { throw InstallError("could not unzip") }
            let app = dir.appendingPathComponent("Hachibu.app")
            guard let info = NSDictionary(contentsOf: app.appendingPathComponent("Contents/Info.plist")) else {
                throw InstallError("Hachibu.app is not in the zip")
            }
            guard info["CFBundleIdentifier"] as? String == bundleID else { throw InstallError("the zip is not Hachibu") }
            guard info["CFBundleShortVersionString"] as? String == release.version else {
                throw InstallError("the app in the zip is not \(release.version)")
            }
            guard run("/usr/bin/codesign", ["--verify", "--deep", app.path]) == 0 else {
                throw InstallError("the signature of the app in the zip is broken")
            }
            return app
        } catch {
            try? fm.removeItem(at: dir)
            throw error
        }
    }

    private static func syncDownload(_ req: URLRequest) throws -> URL {
        let done = DispatchSemaphore(value: 0)
        var out: Result<URL, Error> = .failure(InstallError("download failed"))
        URLSession.shared.downloadTask(with: req) { file, response, error in
            defer { done.signal() }
            guard error == nil, let file else { return }
            guard (response as? HTTPURLResponse)?.statusCode == 200 else {
                out = .failure(InstallError("download failed (HTTP \((response as? HTTPURLResponse)?.statusCode ?? 0))"))
                return
            }
            // 渡されたファイルはこの処理を抜けると消えるので、移しておく
            let keep = FileManager.default.temporaryDirectory.appendingPathComponent("hachibu-update-\(UUID().uuidString).zip")
            do {
                try FileManager.default.moveItem(at: file, to: keep)
                out = .success(keep)
            } catch {
                out = .failure(error)
            }
        }.resume()
        done.wait()
        return try out.get()
    }

    private func replaceAndRelaunch(_ app: URL, version: String) {
        let target = Bundle.main.bundleURL
        // 置き換える直前にもう一度確かめる。待っている間に一時フォルダが消されたり書き換えられたりしうる
        guard FileManager.default.fileExists(atPath: app.path),
              Self.run("/usr/bin/codesign", ["--verify", "--deep", app.path]) == 0 else {
            discardStaged()
            installWhenReady = true
            if let available { prepare(available) }
            return
        }
        do {
            _ = try FileManager.default.replaceItemAt(target, withItemAt: app)
        } catch {
            discardStaged()
            failed("could not replace \(target.path): \(error.localizedDescription)", release: available)
            return
        }
        discardStaged()
        ActionLog.append(L10n.current.logUpdateInstalling(version))
        Self.relaunch(target)
    }

    /// 自分が終わってから開き直す。先に開くと、二重起動の防止（SingleInstance）で新しいほうが閉じる。
    /// 決まった秒数を待つ形だと、終了がそれより遅れたときに同じことが起きるので、PIDが消えるのを待つ。
    /// 60秒たっても終わらなければ開かない（開いても閉じられるだけなので）
    private static func relaunch(_ app: URL) {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/bin/sh")
        // パスは$0、PIDは$1で渡す。文字列に埋め込むと、空白や引用符を含むパスで壊れる
        p.arguments = ["-c", relaunchScript, app.path, String(ProcessInfo.processInfo.processIdentifier)]
        try? p.run()
        NSApp.terminate(nil)
    }

    static let relaunchScript = """
    i=0
    while kill -0 "$1" 2>/dev/null; do
      i=$((i+1)); [ "$i" -gt 300 ] && exit 1
      sleep 0.2
    done
    /usr/bin/open "$0"
    """

    private func discardStaged() {
        if let staged { try? FileManager.default.removeItem(at: staged.dir) }
        staged = nil
    }

    private static func isQuarantined(_ url: URL) -> Bool {
        getxattr(url.path, "com.apple.quarantine", nil, 0, 0, 0) >= 0
    }

    private static func run(_ path: String, _ args: [String]) -> Int32 {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: path)
        p.arguments = args
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        do { try p.run() } catch { return -1 }
        p.waitUntilExit()
        return p.terminationStatus
    }

    // MARK: - 知らせ方

    /// 手で確認して新しいバージョンがあったときに、更新するかを聞く
    private func ask(_ release: Update.Release) {
        let s = L10n.current
        let a = NSAlert()
        a.messageText = s.updateNotificationTitle(release.version)
        a.informativeText = s.updateAskBody(Self.currentVersion)
        a.addButton(withTitle: s.updateRestartButton)
        a.addButton(withTitle: s.updateLaterButton)
        NSApp.activate(ignoringOtherApps: true)
        if a.runModal() == .alertFirstButtonReturn { install() }
    }

    private func failed(_ reason: String, release: Update.Release?) {
        let s = L10n.current
        ActionLog.append(s.updateInstallFailed(reason))
        let a = NSAlert()
        a.messageText = s.updateInstallFailed(reason)
        a.informativeText = s.updateInstallFailedBody
        a.alertStyle = .warning
        a.addButton(withTitle: s.updateOpenPageButton)
        a.addButton(withTitle: s.updateCloseButton)
        NSApp.activate(ignoringOtherApps: true)
        if a.runModal() == .alertFirstButtonReturn, let release { NSWorkspace.shared.open(release.url) }
    }

    /// 同じバージョンでは1回だけ通知する。readyなら押すと置き換え、そうでなければダウンロードのページを開く
    private func notifyOnce(_ release: Update.Release, ready: Bool) {
        guard Update.shouldNotify(release, current: Self.currentVersion, notified: Prefs.notifiedUpdateVersion) else { return }
        Prefs.notifiedUpdateVersion = release.version
        guard Self.canNotify else { return }
        let s = L10n.current
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert]) { granted, _ in
            // 通知を許可されなくても、メニューの項目で気づけるので何もしない
            guard granted else { return }
            let content = UNMutableNotificationContent()
            content.title = ready ? s.updateReadyTitle(release.version) : s.updateNotificationTitle(release.version)
            content.body = ready ? s.updateReadyBody : s.updateNotificationBody
            content.userInfo = ["install": ready, "url": release.url.absoluteString]
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

    /// 通知を押したら、準備ができていれば置き換えて開き直し、そうでなければダウンロードのページを開く
    func userNotificationCenter(_ center: UNUserNotificationCenter, didReceive response: UNNotificationResponse,
                                withCompletionHandler completionHandler: @escaping () -> Void) {
        let info = response.notification.request.content.userInfo
        let install = info["install"] as? Bool == true
        let url = (info["url"] as? String).flatMap(URL.init(string:))
        DispatchQueue.main.async { [weak self] in
            if install, let self, self.available != nil {
                self.install()
            } else if let url {
                NSWorkspace.shared.open(url)
            }
        }
        completionHandler()
    }

    /// アプリが前面にあるときも通知を出す（帯とメニューバーだけのアプリで、前面かどうかを利用者は意識しない）
    func userNotificationCenter(_ center: UNUserNotificationCenter, willPresent notification: UNNotification,
                                withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void) {
        completionHandler([.banner, .list])
    }
}
#endif
