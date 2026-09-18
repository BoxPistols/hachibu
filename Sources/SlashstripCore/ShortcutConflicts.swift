import Foundation

/// macOSが先に受け取るショートカット。登録できても押したときにアプリへ届かないので、登録の前に弾く
public struct ReservedShortcut: Equatable {
    /// com.apple.symbolichotkeysの番号。組み込みの一覧で番号が無いものは"builtin:名前"
    public let id: String
    public let keyCode: UInt32
    public let modifiers: Shortcut.Modifiers

    public init(id: String, keyCode: UInt32, modifiers: Shortcut.Modifiers) {
        self.id = id
        self.keyCode = keyCode
        self.modifiers = modifiers
    }
}

public enum ShortcutConflicts {
    /// 設定ファイルに書かれていなくても効く既定。番号があるものは、設定ファイルにその番号があれば設定ファイルを優先する
    public static let builtIn: [ReservedShortcut] = [
        ReservedShortcut(id: "64", keyCode: 49, modifiers: [.command]),                    // Spotlight ⌘Space
        ReservedShortcut(id: "65", keyCode: 49, modifiers: [.command, .option]),           // Finderの検索 ⌥⌘Space
        ReservedShortcut(id: "60", keyCode: 49, modifiers: [.control]),                    // 前の入力ソース ⌃Space
        ReservedShortcut(id: "61", keyCode: 49, modifiers: [.control, .option]),           // 次の入力ソース ⌃⌥Space
        ReservedShortcut(id: "28", keyCode: 20, modifiers: [.command, .shift]),            // 画面を保存 ⇧⌘3
        ReservedShortcut(id: "29", keyCode: 20, modifiers: [.command, .shift, .control]),  // 画面をコピー ⌃⇧⌘3
        ReservedShortcut(id: "30", keyCode: 21, modifiers: [.command, .shift]),            // 選択部分を保存 ⇧⌘4
        ReservedShortcut(id: "31", keyCode: 21, modifiers: [.command, .shift, .control]),  // 選択部分をコピー ⌃⇧⌘4
        ReservedShortcut(id: "184", keyCode: 23, modifiers: [.command, .shift]),           // スクリーンショットのオプション ⇧⌘5
        ReservedShortcut(id: "27", keyCode: 50, modifiers: [.command]),                    // 次のウインドウ ⌘`
        ReservedShortcut(id: "builtin:appSwitcher", keyCode: 48, modifiers: [.command]),   // アプリの切り替え ⌘Tab
        ReservedShortcut(id: "builtin:forceQuit", keyCode: 53, modifiers: [.command, .option]), // 強制終了 ⌥⌘Esc
        ReservedShortcut(id: "builtin:lockScreen", keyCode: 12, modifiers: [.command, .control]), // 画面をロック ⌃⌘Q
    ]

    // com.apple.symbolichotkeysの修飾キーの値（NSEventの値と同じ）。テンキーやfnのビットは比べない
    private static let shift = 1 << 17
    private static let control = 1 << 18
    private static let option = 1 << 19
    private static let command = 1 << 20

    /// com.apple.symbolichotkeysのAppleSymbolicHotKeysを読む。parametersは[文字, キーコード, 修飾キー]
    /// - Returns: 有効なもの、無効にされている番号
    public static func parseSymbolic(_ dict: [String: Any]) -> (enabled: [ReservedShortcut], disabledIDs: Set<String>) {
        var enabled: [ReservedShortcut] = []
        var disabled: Set<String> = []
        for (id, raw) in dict {
            guard let entry = raw as? [String: Any] else { continue }
            let isOn = (entry["enabled"] as? NSNumber)?.boolValue ?? false
            guard isOn else {
                disabled.insert(id)
                continue
            }
            guard let value = entry["value"] as? [String: Any],
                  let params = value["parameters"] as? [NSNumber], params.count >= 3 else { continue }
            let keyCode = params[1].int64Value
            // 65535はキーが割り当てられていない
            guard keyCode >= 0, keyCode < 65535 else { continue }
            let flags = params[2].intValue
            var m: Shortcut.Modifiers = []
            if flags & shift != 0 { m.insert(.shift) }
            if flags & control != 0 { m.insert(.control) }
            if flags & option != 0 { m.insert(.option) }
            if flags & command != 0 { m.insert(.command) }
            enabled.append(ReservedShortcut(id: id, keyCode: UInt32(keyCode), modifiers: m))
        }
        return (enabled, disabled)
    }

    /// 重なるmacOSのショートカットの番号。重ならなければnil
    public static func conflict(for shortcut: Shortcut, symbolic: [String: Any]?) -> String? {
        let parsed: (enabled: [ReservedShortcut], disabledIDs: Set<String>) = symbolic.map(parseSymbolic) ?? ([], [])
        let listed = Set(parsed.enabled.map(\.id)).union(parsed.disabledIDs)
        // 設定ファイルにある番号は設定ファイルの割り当てを使う（利用者が変えたり無効にしたりしている）
        let defaults = builtIn.filter { !listed.contains($0.id) }
        return (parsed.enabled + defaults).first {
            $0.keyCode == shortcut.keyCode && $0.modifiers == shortcut.modifiers
        }?.id
    }

    /// ⌘（と⇧）だけの組み合わせは、アプリのメニューのショートカットと重なりやすい。
    /// 登録すると、どのアプリにいてもこちらが先に受け取るので、そのアプリの機能が使えなくなる
    public static func isAppMenuProne(_ shortcut: Shortcut) -> Bool {
        shortcut.modifiers.contains(.command) && shortcut.modifiers.isDisjoint(with: [.option, .control])
    }
}
