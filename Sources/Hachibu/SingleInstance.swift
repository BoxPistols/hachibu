import AppKit

/// 同じアプリを1つだけ動かす。
/// 場所の違うコピー（ダウンロード先とアプリケーションフォルダなど）を開くと、macOSはそれぞれを別に起動するため
enum SingleInstance {
    private static var summonName: Notification.Name {
        Notification.Name((Bundle.main.bundleIdentifier ?? "hachibu") + ".summon")
    }

    /// 先に起動した同じアプリがあれば、そちらに帯を出すよう知らせてtrueを返す。
    /// 撮影用や試験用の別の保存先で動かすときは、重ねて起動してよい
    static func handOff() -> Bool {
        let env = ProcessInfo.processInfo.environment
        guard env["HACHIBU_SUITE"] == nil, env["HACHIBU_SOURCE"] != "demo",
              let id = Bundle.main.bundleIdentifier else { return false }
        let me = NSRunningApplication.current
        let myLaunch = me.launchDate ?? Date()
        let earlier = NSRunningApplication.runningApplications(withBundleIdentifier: id).contains {
            $0.processIdentifier != me.processIdentifier && ($0.launchDate ?? .distantPast) <= myLaunch
        }
        guard earlier else { return false }
        DistributedNotificationCenter.default().postNotificationName(summonName, object: nil, userInfo: nil,
                                                                     deliverImmediately: true)
        return true
    }

    /// 後から開かれたコピーからの知らせを受けて、帯を出す
    static func observe(_ onSummon: @escaping () -> Void) {
        DistributedNotificationCenter.default().addObserver(forName: summonName, object: nil, queue: .main) { _ in
            onSummon()
        }
    }
}
