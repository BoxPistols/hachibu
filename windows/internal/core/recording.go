package core

// RejectionKind はショートカットを受け付けなかった理由。
type RejectionKind int

const (
	// NeedsModifier はCtrlもAltも含まない
	NeedsModifier RejectionKind = iota + 1
	// Reserved はWindowsが先に受け取る。ReservedIDにBuiltInReservedの番号が入る
	Reserved
	// Taken は他のアプリかこのアプリの別の操作が使っている
	Taken
)

// PhaseKind は記録の窓の段階。
type PhaseKind int

const (
	Waiting PhaseKind = iota
	Rejected
	// Verifying は登録した。保存はまだで、もう一度押されるのを待っている
	Verifying
	// Verified は届くことを確かめた。「保存」が押されるのを待っている
	Verified
	Saved
)

type Phase struct {
	Kind       PhaseKind
	Candidate  Shortcut
	Rejection  RejectionKind
	ReservedID string
}

// ShortcutRecording は「ショートカットを変更…」の窓の状態。
//
// 押された組み合わせは登録を試すだけで、保存しない。登録したショートカットとしてもう一度届いたら「保存」を押せるようになり、押されたときに保存する。
// そのため取り消し（キャンセル、Esc）は、どの段階でも窓を開く前の状態に戻る。
type ShortcutRecording struct {
	Phase Phase
}

func (r *ShortcutRecording) IsSaved() bool { return r.Phase.Kind == Saved }

// Press は窓が組み合わせを受け取ったとき。
// reservedIDはWindowsが使っている組み合わせならその番号（Conflict）、usedElsewhereはこのアプリの別の操作に割り当て済みならtrue
func (r *ShortcutRecording) Press(candidate Shortcut, reservedID string, usedElsewhere bool, center *HotKeyCenter) {
	if r.IsSaved() {
		return
	}
	var rejection RejectionKind
	switch {
	case !candidate.IsAcceptable():
		rejection = NeedsModifier
	case usedElsewhere:
		rejection = Taken
	case reservedID != "":
		rejection = Reserved
	case !center.TryOut(candidate):
		rejection = Taken
	}
	if rejection == 0 {
		r.Phase = Phase{Kind: Verifying, Candidate: candidate}
		return
	}
	// 前に試して登録したままの組み合わせがあれば外す（保存済みの値は変わらない）
	center.Suspend()
	r.Phase = Phase{Kind: Rejected, Candidate: candidate, Rejection: rejection, ReservedID: reservedID}
}

// CanSave は「保存」を押せる状態か。
func (r *ShortcutRecording) CanSave() bool { return r.Phase.Kind == Verified }

// Arrived は登録したショートカットとして届いたとき。確かめている組み合わせなら「保存」を押せる状態にする（まだ保存しない）。
func (r *ShortcutRecording) Arrived() bool {
	if r.Phase.Kind != Verifying {
		return false
	}
	r.Phase.Kind = Verified
	return true
}

// Save は届くことを確かめた組み合わせだけを保存する。保存したらtrue（窓を閉じてよい）。
func (r *ShortcutRecording) Save(center *HotKeyCenter) bool {
	if r.Phase.Kind != Verified {
		return false
	}
	center.Commit(r.Phase.Candidate)
	r.Phase.Kind = Saved
	return true
}

// Cancel はキャンセルとEsc。確かめていない組み合わせは保存していないので、保存済みの値を登録し直せば窓を開く前に戻る。
func (r *ShortcutRecording) Cancel(center *HotKeyCenter) {
	center.RegisterSaved()
}

func (r *ShortcutRecording) TurnOff(center *HotKeyCenter) {
	center.Disable()
	r.Phase = Phase{Kind: Waiting}
}
