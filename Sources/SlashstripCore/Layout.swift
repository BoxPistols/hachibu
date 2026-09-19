import Foundation

/// 帯の表示モード。利用者が選ぶのは「普段の状態」で、マウスを乗せるかショートカットで一時的に標準へ広げる。
public enum StripLayout: String, CaseIterable {
    /// モデル、effort、使用率
    case full
    /// 使用率（"S12 W76 F55"）だけ
    case usage
    /// 小さなつまみだけ。使用率の段階は輪の色で示す
    case tab
    /// 出さない。メニューバーとショートカットから呼ぶ
    case hidden
}

/// 今どのモードで描くかと、そのモードで何を並べるかの判断。
public enum LayoutRules {
    /// - expanded: マウスが乗っている、またはショートカットで呼び出し中
    public static func effective(rest: StripLayout, expanded: Bool) -> StripLayout {
        expanded ? .full : rest
    }

    /// - usageText: 使用率だけの文字列。usageモードで状態の枠の文字をこれに差し替える（nilなら元の文字のまま）
    public static func visibleSlots(_ slots: [Slot], layout: StripLayout, usageText: String? = nil) -> [Slot] {
        switch layout {
        case .full:
            return slots
        case .usage:
            return slots.map { slot in
                guard slot.id == Slot.statusID, let usageText else { return slot }
                var s = slot
                s.text = usageText
                return s
            }
        case .tab:
            return slots.filter { $0.id == Slot.statusID }
        case .hidden:
            return []
        }
    }
}

/// メニューバーの項目に何を出すか。
public enum MenuBarStyle: String, CaseIterable {
    case icon
    case usage
    case status

    /// - usage: "S11 W75 F55" のような使用率だけ
    /// - status: 帯の状態の枠と同じ全文
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
