import Foundation
import ServiceManagement

/// ログイン時の自動起動。macOS標準のログイン項目（SMAppService）に、このアプリ自身を登録する
enum LoginItem {
    static var isEnabled: Bool { SMAppService.mainApp.status == .enabled }

    /// 登録したが、利用者がシステム設定で許可するまで動かない
    static var needsApproval: Bool { SMAppService.mainApp.status == .requiresApproval }

    /// 切り替える。失敗したら理由を返す
    static func set(_ enabled: Bool) -> String? {
        do {
            if enabled {
                try SMAppService.mainApp.register()
            } else {
                try SMAppService.mainApp.unregister()
            }
            return nil
        } catch {
            return error.localizedDescription
        }
    }

    static func openSettings() {
        SMAppService.openSystemSettingsLoginItems()
    }
}
