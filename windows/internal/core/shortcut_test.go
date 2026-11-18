package core

import (
	"strings"
	"testing"
)

func TestDisplaysInWindowsOrder(t *testing.T) {
	s := Shortcut{VK: 'K', Mods: ModShift | ModAlt | ModCtrl, Label: "K"}
	if s.Display() != "Ctrl+Alt+Shift+K" || DefaultSummon.Display() != "Ctrl+Alt+/" {
		t.Fatalf("%q %q", s.Display(), DefaultSummon.Display())
	}
}

func TestRequiresCtrlOrAlt(t *testing.T) {
	cases := map[Shortcut]bool{
		{VK: 'A', Label: "A"}:                        false,
		{VK: 'A', Mods: ModShift, Label: "A"}:        false,
		{VK: 'A', Mods: ModCtrl, Label: "A"}:         true,
		{VK: 'A', Mods: ModAlt, Label: "A"}:          true,
		{VK: 'A', Mods: ModCtrl | ModAlt, Label: ""}: false,
	}
	for s, want := range cases {
		if s.IsAcceptable() != want {
			t.Errorf("%+v", s)
		}
	}
}

func TestMapsToRegisterHotKeyModifiers(t *testing.T) {
	if DefaultSummon.Win32Modifiers() != 1|2 {
		t.Error("default")
	}
	if (Shortcut{Mods: ModCtrl | ModShift}).Win32Modifiers() != 2|4 {
		t.Error("ctrl+shift")
	}
}

func TestLabelsKeysWithoutCharacters(t *testing.T) {
	cases := []struct {
		vk    uint32
		chars string
		want  string
	}{
		{VKSpace, " ", "Space"}, {VKF1, "", "F1"}, {VKF1 + 11, "", "F12"}, {'K', "k", "K"}, {VKOem2, "/", "/"},
		{VKNumpad0 + 7, "7", "Num 7"}, {VKLeft, "", "←"},
	}
	for _, c := range cases {
		if got := KeyLabel(c.vk, c.chars); got != c.want {
			t.Errorf("KeyLabel(%#x) = %q, want %q", c.vk, got, c.want)
		}
	}
}

func TestShortcutSurvivesSaveAndLoad(t *testing.T) {
	got, ok := DecodeShortcut(EncodeShortcut(DefaultSummon))
	if !ok || got != DefaultSummon {
		t.Fatalf("%+v", got)
	}
	for _, bad := range []string{"", "{", `{"vk":0}`} {
		if _, ok := DecodeShortcut(bad); ok {
			t.Errorf("%q", bad)
		}
	}
}

func TestRejectsWindowsShortcuts(t *testing.T) {
	if id, ok := Conflict(Shortcut{VK: VKEscape, Mods: ModCtrl | ModShift, Label: "Esc"}); !ok || id != "builtin:taskManager" {
		t.Fatalf("%q", id)
	}
	if id, ok := Conflict(Shortcut{VK: VKTab, Mods: ModAlt, Label: "Tab"}); !ok || id != "builtin:appSwitcher" {
		t.Fatalf("%q", id)
	}
	// 修飾キーがすべて一致するものだけ
	if _, ok := Conflict(Shortcut{VK: VKEscape, Mods: ModCtrl | ModShift | ModAlt, Label: "Esc"}); ok {
		t.Fatal("extra modifier")
	}
}

func TestDefaultSummonIsFreeOnAStockSystem(t *testing.T) {
	if _, ok := Conflict(DefaultSummon); ok {
		t.Fatal("default conflicts")
	}
}

func TestFlagsSingleModifierShortcutsAsAppProne(t *testing.T) {
	prone := []Shortcut{
		{VK: VKOem2, Mods: ModCtrl, Label: "/"},
		{VK: VKOem2, Mods: ModCtrl | ModShift, Label: "/"},
		{VK: 'K', Mods: ModAlt, Label: "K"},
	}
	for _, s := range prone {
		if !IsAppShortcutProne(s) {
			t.Errorf("%+v", s)
		}
	}
	if IsAppShortcutProne(DefaultSummon) || IsAppShortcutProne(Shortcut{VK: 'K', Mods: ModCtrl | ModAlt | ModShift, Label: "K"}) {
		t.Error("ctrl+alt is safe")
	}
}

func TestEveryReservedIDHasAName(t *testing.T) {
	for _, r := range BuiltInReserved {
		if strings.Contains(En.SystemShortcutName(r.ID), "#") || strings.Contains(Ja.SystemShortcutName(r.ID), "番号") {
			t.Errorf("%s", r.ID)
		}
	}
}
