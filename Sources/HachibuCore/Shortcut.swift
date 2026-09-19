import Foundation

/// 帯を呼び出すショートカット。キーコードと修飾キー、表示用のキー名を持つ。
public struct Shortcut: Equatable, Codable {
    public struct Modifiers: OptionSet, Equatable, Codable {
        public let rawValue: Int
        public init(rawValue: Int) { self.rawValue = rawValue }

        public static let control = Modifiers(rawValue: 1 << 0)
        public static let option = Modifiers(rawValue: 1 << 1)
        public static let shift = Modifiers(rawValue: 1 << 2)
        public static let command = Modifiers(rawValue: 1 << 3)
    }

    public let keyCode: UInt32
    public let modifiers: Modifiers
    /// 記録したときに押されたキーの文字（"/" や "K"）。キーコードから配列ごとの文字を引き直さずに済む
    public let keyLabel: String

    public init(keyCode: UInt32, modifiers: Modifiers, keyLabel: String) {
        self.keyCode = keyCode
        self.modifiers = modifiers
        self.keyLabel = keyLabel
    }

    /// ⌥⌘/（/は日本語配列でも同じキーコード44）
    public static let defaultSummon = Shortcut(keyCode: 44, modifiers: [.option, .command], keyLabel: "/")

    /// Appleの表記順（⌃⌥⇧⌘）で並べる
    public var display: String {
        var s = ""
        if modifiers.contains(.control) { s += "⌃" }
        if modifiers.contains(.option) { s += "⌥" }
        if modifiers.contains(.shift) { s += "⇧" }
        if modifiers.contains(.command) { s += "⌘" }
        return s + keyLabel
    }

    /// ⌘・⌥・⌃のどれも含まない組み合わせは、ふだんの文字入力を奪うので受け付けない（⇧だけも同じ）
    public var isAcceptable: Bool {
        !modifiers.isDisjoint(with: [.command, .option, .control]) && !keyLabel.isEmpty
    }

    /// CarbonのRegisterEventHotKeyに渡す値（cmdKey=256, shiftKey=512, optionKey=2048, controlKey=4096）
    public var carbonModifiers: UInt32 {
        var v: UInt32 = 0
        if modifiers.contains(.command) { v |= 256 }
        if modifiers.contains(.shift) { v |= 512 }
        if modifiers.contains(.option) { v |= 2048 }
        if modifiers.contains(.control) { v |= 4096 }
        return v
    }

    /// 文字を持たないキーの表示名。キーコードはmacOSの仮想キーコード
    public static func label(keyCode: UInt32, characters: String?) -> String {
        let named: [UInt32: String] = [
            49: "Space", 36: "↩", 48: "⇥", 51: "⌫", 117: "⌦", 53: "⎋",
            123: "←", 124: "→", 125: "↓", 126: "↑", 115: "↖", 119: "↘", 116: "⇞", 121: "⇟",
            122: "F1", 120: "F2", 99: "F3", 118: "F4", 96: "F5", 97: "F6",
            98: "F7", 100: "F8", 101: "F9", 109: "F10", 103: "F11", 111: "F12",
        ]
        if let name = named[keyCode] { return name }
        return (characters ?? "").trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
    }
}
