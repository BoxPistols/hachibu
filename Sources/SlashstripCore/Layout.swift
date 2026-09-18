import Foundation

/// 帯の表示モード。利用者が選ぶのは「普段の状態」で、マウスを乗せるかショートカットで一時的にフルへ広げる。
public enum StripLayout: String, CaseIterable {
    /// 状態＋全ボタン（Touch Barと同じ）
    case full
    /// 状態の1枠だけ。許可応答と番号ボタンは出す
    case compact
    /// 状態の枠を使用率（"S12 W76 F55"）だけに縮める。許可応答と番号ボタンは出す
    case usage
    /// 小さなつまみだけ。状態は色で示す
    case tab
    /// 普段は出さない。答えが要るとき（許可応答・番号選択）だけ1段出す。メニューバーとショートカットから呼ぶ
    case hidden
}

/// 今どのモードで描くかと、そのモードで何を並べるかの判断。
public enum LayoutRules {
    /// - expanded: マウスが乗っている、またはショートカットで呼び出し中
    public static func effective(rest: StripLayout, expanded: Bool, slots: [Slot]) -> StripLayout {
        if expanded { return .full }
        // 許可待ちや番号選択は、つまみの色だけでも、隠したままでも応えられないので1段広げる
        if rest == .tab || rest == .hidden, slots.contains(where: isTransient) {
            return .compact
        }
        return rest
    }

    /// - usageText: 使用率だけの文字列。usageモードで状態の枠の文字をこれに差し替える（nilなら元の文字のまま）
    public static func visibleSlots(_ slots: [Slot], layout: StripLayout, usageText: String? = nil) -> [Slot] {
        switch layout {
        case .full:
            return slots
        case .compact:
            return slots.filter { $0.id == Widgets.status || isTransient($0) }
        case .usage:
            return slots.filter { $0.id == Widgets.status || isTransient($0) }.map { slot in
                guard slot.id == Widgets.status, let usageText else { return slot }
                var s = slot
                s.text = usageText
                return s
            }
        case .tab:
            return slots.filter { $0.id == Widgets.status }
        case .hidden:
            return []
        }
    }

    /// 応えるまで意味が続くボタン（許可応答・番号）
    static func isTransient(_ slot: Slot) -> Bool {
        slot.id.hasPrefix("perm-") || slot.id.hasPrefix("menu-")
    }
}

/// メニューバーの項目に何を出すか。
public enum MenuBarStyle: String, CaseIterable {
    case icon
    case usage
    case status

    /// - usage: "S11 W75 F55" のような使用率だけ
    /// - status: 帯の状態ボタンと同じ全文
    public static func title(style: MenuBarStyle, statusText: String?, limits: [UsageLimit]) -> String? {
        switch style {
        case .icon:
            return nil
        case .usage:
            return limits.isEmpty ? nil : Usage.statusText(model: nil, effort: nil, limits: limits)
        case .status:
            return statusText
        }
    }
}
