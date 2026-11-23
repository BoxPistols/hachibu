package core

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Modifiers は修飾キー。Windowsのキーは記録しない（Win＋文字の大半はWindowsが使っている）。
type Modifiers int

const (
	ModCtrl Modifiers = 1 << iota
	ModAlt
	ModShift
)

// 仮想キーコード（Windowsの値）。テストと既定値で使うものだけ
const (
	VKBack     = 0x08
	VKTab      = 0x09
	VKReturn   = 0x0D
	VKPause    = 0x13
	VKKanji    = 0x19
	VKEscape   = 0x1B
	VKSpace    = 0x20
	VKPrior    = 0x21
	VKNext     = 0x22
	VKEnd      = 0x23
	VKHome     = 0x24
	VKLeft     = 0x25
	VKUp       = 0x26
	VKRight    = 0x27
	VKDown     = 0x28
	VKSnapshot = 0x2C
	VKInsert   = 0x2D
	VKDelete   = 0x2E
	VKF1       = 0x70
	VKF4       = 0x73
	VKF24      = 0x87
	VKNumpad0  = 0x60
	VKMultiply = 0x6A
	VKAdd      = 0x6B
	VKSubtract = 0x6D
	VKDecimal  = 0x6E
	VKDivide   = 0x6F
	// VKOem2 は「/」のキー。英語配列でも日本語配列でも同じ値
	VKOem2 = 0xBF
	// VKOem3 は英語配列の「`」のキー（Alt＋`で入力モードを切り替える）
	VKOem3 = 0xC0
)

// Shortcut は帯を呼び出すショートカット。仮想キーコードと修飾キー、表示用のキー名を持つ。
type Shortcut struct {
	VK   uint32    `json:"vk"`
	Mods Modifiers `json:"mods"`
	// Label は記録したときに押されたキーの文字（"/" や "K"）。キーボードの配列ごとの文字を引き直さずに済む
	Label string `json:"label"`
}

// DefaultSummon はCtrl+Alt+/。macOS向けの⌥⌘/と同じ「/」のキーにする
var DefaultSummon = Shortcut{VK: VKOem2, Mods: ModCtrl | ModAlt, Label: "/"}

// Display はWindowsの表記（"Ctrl+Alt+Shift+K"）。
func (s Shortcut) Display() string {
	return s.Mods.Prefix() + s.Label
}

// Prefix は修飾キーだけの表記（"Ctrl+Alt+"）。記録の窓で、キーを押す前の途中経過に使う
func (m Modifiers) Prefix() string {
	var b strings.Builder
	if m&ModCtrl != 0 {
		b.WriteString("Ctrl+")
	}
	if m&ModAlt != 0 {
		b.WriteString("Alt+")
	}
	if m&ModShift != 0 {
		b.WriteString("Shift+")
	}
	return b.String()
}

// IsAcceptable は、CtrlもAltも含まない組み合わせを受け付けない（ふだんの文字入力を奪うため。Shiftだけも同じ）。
func (s Shortcut) IsAcceptable() bool {
	return s.Mods&(ModCtrl|ModAlt) != 0 && s.Label != ""
}

// Win32Modifiers はRegisterHotKeyに渡す値（MOD_ALT=1, MOD_CONTROL=2, MOD_SHIFT=4）。
func (s Shortcut) Win32Modifiers() uint32 {
	var v uint32
	if s.Mods&ModAlt != 0 {
		v |= 1
	}
	if s.Mods&ModCtrl != 0 {
		v |= 2
	}
	if s.Mods&ModShift != 0 {
		v |= 4
	}
	return v
}

// Same は同じ組み合わせか（表示用のキー名は比べない）。
func (s Shortcut) Same(o Shortcut) bool {
	return s.VK == o.VK && s.Mods == o.Mods
}

var namedKeys = map[uint32]string{
	VKSpace: "Space", VKReturn: "Enter", VKTab: "Tab", VKBack: "Backspace", VKDelete: "Delete", VKEscape: "Esc",
	VKLeft: "←", VKUp: "↑", VKRight: "→", VKDown: "↓", VKHome: "Home", VKEnd: "End", VKPrior: "PageUp", VKNext: "PageDown",
	VKInsert: "Insert", VKSnapshot: "PrtScn", VKPause: "Pause",
	VKMultiply: "Num *", VKAdd: "Num +", VKSubtract: "Num -", VKDecimal: "Num .", VKDivide: "Num /",
}

// KeyLabel は表示用のキー名。文字を持たないキーは名前、それ以外はキーの文字（chars）を大文字にする。
func KeyLabel(vk uint32, chars string) string {
	if name, ok := namedKeys[vk]; ok {
		return name
	}
	if vk >= VKF1 && vk <= VKF24 {
		return fmt.Sprintf("F%d", vk-VKF1+1)
	}
	if vk >= VKNumpad0 && vk <= VKNumpad0+9 {
		return fmt.Sprintf("Num %d", vk-VKNumpad0)
	}
	return strings.ToUpper(strings.TrimSpace(chars))
}

// EncodeShortcut とDecodeShortcut は設定ファイルに書く形。
func EncodeShortcut(s Shortcut) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func DecodeShortcut(text string) (Shortcut, bool) {
	var s Shortcut
	if text == "" || json.Unmarshal([]byte(text), &s) != nil || s.VK == 0 {
		return Shortcut{}, false
	}
	return s, true
}
