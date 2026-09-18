import Foundation
import SlashstripCore

/// 利用者の設定。UserDefaultsのキーはここにだけ書く
enum Prefs {
    /// 撮影用モード（SLASHSTRIP_SOURCE=demo）では、利用者の設定に触れないよう別の保存先を使う
    static let defaults: UserDefaults = ProcessInfo.processInfo.environment["SLASHSTRIP_SOURCE"] == "demo"
        ? (UserDefaults(suiteName: "dev.local.slashstrip.demo") ?? .standard)
        : .standard

    private static let layoutKey = "restLayout"
    private static let opacityKey = "opacity"
    private static let menuBarKey = "menuBarStyle"
    private static let warningKey = "warningThreshold"
    private static let criticalKey = "criticalThreshold"

    static let opacityChoices: [Double] = [1.0, 0.85, 0.7, 0.5, 0.3]

    static var restLayout: StripLayout {
        get { StripLayout(rawValue: Prefs.defaults.string(forKey: layoutKey) ?? "") ?? .full }
        set { Prefs.defaults.set(newValue.rawValue, forKey: layoutKey) }
    }

    static var opacity: Double {
        get {
            let v = Prefs.defaults.double(forKey: opacityKey)
            return opacityChoices.contains(v) ? v : 1.0
        }
        set { Prefs.defaults.set(newValue, forKey: opacityKey) }
    }

    static var thresholds: UsageThresholds {
        get {
            let d = Prefs.defaults
            return UsageThresholds(warning: d.integer(forKey: warningKey), critical: d.integer(forKey: criticalKey))
                ?? .default
        }
        set {
            Prefs.defaults.set(newValue.warning, forKey: warningKey)
            Prefs.defaults.set(newValue.critical, forKey: criticalKey)
        }
    }

    static var menuBarStyle: MenuBarStyle {
        get { MenuBarStyle(rawValue: Prefs.defaults.string(forKey: menuBarKey) ?? "") ?? .icon }
        set { Prefs.defaults.set(newValue.rawValue, forKey: menuBarKey) }
    }
}
