//go:build windows

package app

import (
	"strings"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// termsURL は注意書きで引用している規約の出典
const termsURL = "https://code.claude.com/docs/en/legal-and-compliance"

const (
	consentEnableID  = 101
	consentDeclineID = 102
)

var consentOpen bool

// askUsageAPIConsent は使用率APIを使うかを尋ねる。何を使い、どんなリスクがあるかを先に示し、利用者が選んだときだけ有効にする。
// 閉じるまで戻らない。Escと閉じるボタンは「今はしない」と同じ
func askUsageAPIConsent(s *core.Strings) (enabled bool, answered bool) {
	if consentOpen {
		return false, false
	}
	consentOpen = true
	defer func() { consentOpen = false }()
	pressed, err := w32.TaskDialog(w32.TaskDialogConfig{
		Title:           core.AppName,
		MainInstruction: s.ConsentTitle,
		Content:         strings.Join([]string{s.ConsentWhat, s.ConsentRisk, s.ConsentPrivacy, s.ConsentWithout}, "\n\n"),
		Footer:          `<a href="` + termsURL + `">` + s.ConsentTerms + `</a>`,
		Buttons:         []w32.TaskDialogButton{{ID: consentEnableID, Text: s.ConsentEnable}, {ID: consentDeclineID, Text: s.ConsentDecline}},
		// 既定は「今はしない」。Enterを押しただけで有効にならないように
		DefaultButton: consentDeclineID,
		OnHyperlink:   openURL,
	})
	if err != nil {
		// タスクダイアログが使えない環境（マニフェストの無い起動など）では、はい・いいえの窓で尋ねる
		text := strings.Join([]string{s.ConsentTitle, s.ConsentWhat, s.ConsentRisk, s.ConsentPrivacy, s.ConsentWithout, termsURL}, "\n\n")
		r := w32.MessageBox(0, text, core.AppName, 0x04|0x100|0x10000) // MB_YESNO | MB_DEFBUTTON2 | MB_SETFOREGROUND
		return r == 6, true                                            // IDYES
	}
	return pressed == consentEnableID, true
}
