package core

import "strconv"

// StripLayout は帯の表示モード。利用者が選ぶのは「普段の状態」で、マウスを乗せるかショートカットで一時的に標準へ広げる。
type StripLayout string

const (
	// LayoutFull はモデル、effort、使用率
	LayoutFull StripLayout = "full"
	// LayoutUsage は使用率（"S12 W76 F55"）だけ
	LayoutUsage StripLayout = "usage"
	// LayoutTab は小さなつまみだけ。使用率の段階は輪の色で示す
	LayoutTab StripLayout = "tab"
	// LayoutHidden は出さない。トレイのアイコンとショートカットから呼ぶ
	LayoutHidden StripLayout = "hidden"
)

var AllLayouts = []StripLayout{LayoutFull, LayoutUsage, LayoutTab, LayoutHidden}

// ParseLayout は保存された値を読む。読めなければfalse（呼び出し側は標準に戻す）。
func ParseLayout(s string) (StripLayout, bool) {
	for _, l := range AllLayouts {
		if string(l) == s {
			return l, true
		}
	}
	return "", false
}

// NextLayout はショートカットで順に切り替える並び。「隠す」は含めない（切り替えた結果が見えなくなるため）。
func NextLayout(after StripLayout) StripLayout {
	switch after {
	case LayoutFull:
		return LayoutUsage
	case LayoutUsage:
		return LayoutTab
	default:
		return LayoutFull
	}
}

// EffectiveLayout は今どのモードで描くか。expandedはマウスが乗っている、またはショートカットで呼び出し中。
func EffectiveLayout(rest StripLayout, expanded bool) StripLayout {
	if expanded {
		return LayoutFull
	}
	return rest
}

// VisibleSlots はそのモードで何を並べるか。usageTextは使用率だけの文字列で、usageモードで状態の枠の文字をこれに差し替える（空なら元の文字のまま）。
func VisibleSlots(slots []Slot, layout StripLayout, usageText string) []Slot {
	switch layout {
	case LayoutFull:
		return slots
	case LayoutUsage:
		out := make([]Slot, len(slots))
		for i, s := range slots {
			if s.ID == StatusID && usageText != "" {
				s.Text = usageText
			}
			out[i] = s
		}
		return out
	case LayoutTab:
		var out []Slot
		for _, s := range slots {
			if s.ID == StatusID {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// GaugeReading は目盛りの表示に使う値。週枠があれば週枠、無ければ最初の枠。
type GaugeReading struct {
	Label   string
	Percent int
}

func GaugeFrom(limits []UsageLimit) (GaugeReading, bool) {
	if len(limits) == 0 {
		return GaugeReading{}, false
	}
	limit := limits[0]
	for _, l := range limits {
		if l.Kind == KindWeekly {
			limit = l
			break
		}
	}
	// 幅を取らないよう、枠の頭文字は付けずに数字だけにする（どの枠かはメニューに出る）
	return GaugeReading{Label: strconv.Itoa(limit.Percent), Percent: limit.Percent}, true
}

// TrayStyle はトレイのアイコンに何を描くか。トレイには文字を並べられないので、macOS向けのメニューバーの「使用率」「状態の全文」の代わりに数字を描く。
type TrayStyle string

const (
	TrayIcon TrayStyle = "icon"
	// TrayGauge は絵そのものを週枠の目盛りにする
	TrayGauge TrayStyle = "gauge"
	// TrayPercent は週枠の使用率を数字で描く。黄や赤の段階では札の色にする
	TrayPercent TrayStyle = "percent"
)

var AllTrayStyles = []TrayStyle{TrayIcon, TrayGauge, TrayPercent}

func ParseTrayStyle(s string) (TrayStyle, bool) {
	for _, t := range AllTrayStyles {
		if string(t) == s {
			return t, true
		}
	}
	return "", false
}
