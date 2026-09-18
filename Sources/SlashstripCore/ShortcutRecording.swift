import Foundation

/// 「ショートカットを変更…」の窓の状態。
///
/// 押された組み合わせは登録を試すだけで、保存しない。登録したショートカットとしてもう一度届いたときに初めて保存する。
/// そのため取り消し（キャンセル、Esc）は、どの段階でも窓を開く前の状態に戻る。
public struct ShortcutRecording: Equatable {
    public enum Rejection: Equatable {
        /// ⌘・⌥・⌃のどれも含まない
        case needsModifier
        /// macOSが先に受け取る。値はcom.apple.symbolichotkeysの番号
        case reserved(String)
        /// 他のアプリが登録している
        case taken
    }

    public enum Phase: Equatable {
        case waiting
        case rejected(Shortcut, Rejection)
        /// 登録した。保存はまだで、もう一度押されるのを待っている
        case verifying(Shortcut)
        /// 届いたので保存した
        case saved(Shortcut)
    }

    public private(set) var phase: Phase = .waiting

    public init() {}

    public var isSaved: Bool {
        if case .saved = phase { return true }
        return false
    }

    /// 窓が組み合わせを受け取った
    /// - reservedID: macOSが使っている組み合わせならその番号（ShortcutConflicts.conflict）
    public mutating func press(_ candidate: Shortcut, reservedID: String?, center: HotKeyCenter) {
        guard !isSaved else { return }
        let rejection: Rejection?
        if !candidate.isAcceptable {
            rejection = .needsModifier
        } else if let reservedID {
            rejection = .reserved(reservedID)
        } else if !center.tryOut(candidate) {
            rejection = .taken
        } else {
            rejection = nil
        }
        guard let rejection else {
            phase = .verifying(candidate)
            return
        }
        // 前に試して登録したままの組み合わせがあれば外す（保存済みの値は変わらない）
        center.suspend()
        phase = .rejected(candidate, rejection)
    }

    /// 登録したショートカットとして届いた。確かめている組み合わせなら保存する
    /// - Returns: 保存したらtrue（窓を閉じてよい）
    public mutating func arrived(center: HotKeyCenter) -> Bool {
        guard case .verifying(let candidate) = phase else { return false }
        center.commit(candidate)
        phase = .saved(candidate)
        return true
    }

    /// キャンセル・Esc。確かめていない組み合わせは保存していないので、保存済みの値を登録し直せば窓を開く前に戻る
    public func cancel(center: HotKeyCenter) {
        center.registerSaved()
    }

    public mutating func turnOff(center: HotKeyCenter) {
        center.disable()
        phase = .waiting
    }
}
