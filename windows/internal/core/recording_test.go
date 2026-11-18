package core

import (
	"reflect"
	"testing"
)

// fakeRegistrar はRegisterHotKeyの代わりの登録器。いま登録されている組み合わせと、他のアプリが押さえている組み合わせを持つ
type fakeRegistrar struct {
	takenByOtherApps map[string]bool
	registered       *Shortcut
}

func (f *fakeRegistrar) Register(s Shortcut) bool {
	if f.takenByOtherApps[s.Display()] {
		return false
	}
	f.registered = &s
	return true
}

func (f *fakeRegistrar) Unregister() { f.registered = nil }

// memoryStore はディスクに書かない保存先。テストごとに別のものを使う
type memoryStore map[string]any

func (m memoryStore) Bool(key string) bool      { v, _ := m[key].(bool); return v }
func (m memoryStore) String(key string) string  { v, _ := m[key].(string); return v }
func (m memoryStore) Set(key string, value any) { m[key] = value }

var (
	scA       = Shortcut{VK: VKOem2, Mods: ModCtrl | ModAlt, Label: "/"}
	scB       = Shortcut{VK: 'J', Mods: ModCtrl | ModAlt | ModShift, Label: "J"}
	reservedC = Shortcut{VK: VKTab, Mods: ModAlt, Label: "Tab"}
	scD       = Shortcut{VK: 'K', Mods: ModCtrl | ModAlt | ModShift, Label: "K"}
)

func eq(a *Shortcut, b *Shortcut) bool { return reflect.DeepEqual(a, b) }
func ptr(s Shortcut) *Shortcut         { return &s }

// opened は保存済みの組み合わせがsavedで、記録の窓を開いた直後の状態
func opened(saved *Shortcut, registrar *fakeRegistrar) (*HotKeyCenter, *ShortcutSettings, *fakeRegistrar) {
	if registrar == nil {
		registrar = &fakeRegistrar{}
	}
	settings := SummonSettings(memoryStore{})
	if saved != nil {
		settings.Save(*saved)
	} else {
		settings.SaveDisabled()
	}
	center := NewHotKeyCenter(registrar, settings)
	center.Suspend()
	return center, settings, registrar
}

func TestCancelAfterAnAcceptedThenRejectedComboRestoresTheOriginal(t *testing.T) {
	// 手順: Bを登録 → Windowsが使うCで弾かれる → キャンセル
	center, settings, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	if rec.Phase.Kind != Verifying || rec.Phase.Candidate != scB {
		t.Fatalf("%+v", rec.Phase)
	}
	id, _ := Conflict(reservedC)
	rec.Press(reservedC, id, false, center)
	if rec.Phase.Kind != Rejected || rec.Phase.Rejection != Reserved || rec.Phase.ReservedID != "builtin:appSwitcher" {
		t.Fatalf("%+v", rec.Phase)
	}
	rec.Cancel(center)
	if !eq(center.Shortcut, &scA) || !eq(registrar.registered, &scA) || !eq(settings.Load(), &scA) {
		t.Fatal("not restored")
	}
}

func TestEscOrCancelWhileVerifyingRestoresTheOriginal(t *testing.T) {
	center, settings, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	if !eq(registrar.registered, &scB) {
		t.Fatal("B should be registered")
	}
	rec.Cancel(center)
	if !eq(registrar.registered, &scA) || !eq(settings.Load(), &scA) {
		t.Fatal("not restored")
	}
}

func TestCancelKeepsAShortcutThatWasOffBeforeOpening(t *testing.T) {
	center, settings, registrar := opened(nil, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.Cancel(center)
	if center.Shortcut != nil || registrar.registered != nil || settings.Load() != nil {
		t.Fatal("should stay off")
	}
}

func TestNothingIsSavedUntilSaveIsClicked(t *testing.T) {
	center, settings, _ := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	// 確かめる前にアプリが終わっても、次に読むのは元の値
	if !eq(settings.Load(), &scA) || !eq(center.Shortcut, &scA) {
		t.Fatal("saved too early")
	}
	// 「保存」は、届くことを確かめるまで効かない
	if rec.Save(center) || !eq(settings.Load(), &scA) {
		t.Fatal("early save")
	}
	if !rec.Arrived() || rec.Phase.Kind != Verified || !rec.CanSave() {
		t.Fatalf("%+v", rec.Phase)
	}
	// 届いただけでは保存しない
	if !eq(settings.Load(), &scA) {
		t.Fatal("saved on arrival")
	}
	if !rec.Save(center) || rec.Phase.Kind != Saved || !eq(settings.Load(), &scB) || !eq(center.Shortcut, &scB) {
		t.Fatalf("%+v", rec.Phase)
	}
}

func TestCancelAfterVerifyingRestoresTheOriginal(t *testing.T) {
	center, settings, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.Arrived()
	rec.Cancel(center)
	if !eq(settings.Load(), &scA) || !eq(registrar.registered, &scA) {
		t.Fatal("not restored")
	}
}

func TestADifferentComboAfterVerifyingNeedsVerifyingAgain(t *testing.T) {
	center, settings, _ := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.Arrived()
	rec.Press(scD, "", false, center)
	if rec.Phase.Kind != Verifying || rec.Phase.Candidate != scD || rec.CanSave() || !eq(settings.Load(), &scA) {
		t.Fatalf("%+v", rec.Phase)
	}
}

func TestAComboTakenByAnotherAppReleasesThePreviousTrial(t *testing.T) {
	registrar := &fakeRegistrar{takenByOtherApps: map[string]bool{scD.Display(): true}}
	center, settings, _ := opened(&scA, registrar)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.Press(scD, "", false, center)
	if rec.Phase.Kind != Rejected || rec.Phase.Rejection != Taken {
		t.Fatalf("%+v", rec.Phase)
	}
	// 試していたBは外れていて、押しても何も起きない
	if registrar.registered != nil || !eq(settings.Load(), &scA) {
		t.Fatal("B still registered")
	}
}

func TestAComboUsedByAnotherActionOfThisAppIsRejected(t *testing.T) {
	center, settings, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", true, center)
	if rec.Phase.Kind != Rejected || rec.Phase.Rejection != Taken || registrar.registered != nil || !eq(settings.Load(), &scA) {
		t.Fatalf("%+v", rec.Phase)
	}
}

func TestTheLayoutCycleShortcutHasNoDefaultAndItsOwnStorage(t *testing.T) {
	store := memoryStore{}
	summon := SummonSettings(store)
	cycle := LayoutCycleSettings(store)
	if !eq(summon.Load(), ptr(DefaultSummon)) || cycle.Load() != nil {
		t.Fatal("defaults")
	}
	cycle.Save(scB)
	if !eq(cycle.Load(), &scB) || !eq(summon.Load(), ptr(DefaultSummon)) {
		t.Fatal("separate storage")
	}
	summon.SaveDisabled()
	if !eq(cycle.Load(), &scB) || summon.Load() != nil {
		t.Fatal("disable summon only")
	}
}

func TestArrivalIsIgnoredUnlessVerifying(t *testing.T) {
	center, settings, _ := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(reservedC, "builtin:appSwitcher", false, center)
	if rec.Arrived() || rec.Save(center) || !eq(settings.Load(), &scA) {
		t.Fatal("should ignore")
	}
}

func TestPressesAfterSavingAreIgnored(t *testing.T) {
	center, settings, _ := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.Arrived()
	rec.Save(center)
	rec.Press(scD, "", false, center)
	if rec.Phase.Kind != Saved || rec.Phase.Candidate != scB || !eq(settings.Load(), &scB) {
		t.Fatalf("%+v", rec.Phase)
	}
}

func TestTurningOffSavesDisabled(t *testing.T) {
	center, settings, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	rec.Press(scB, "", false, center)
	rec.TurnOff(center)
	if center.Shortcut != nil || registrar.registered != nil || settings.Load() != nil {
		t.Fatal("should be off")
	}
}

func TestMissingModifierIsRejectedBeforeRegistering(t *testing.T) {
	center, _, registrar := opened(&scA, nil)
	var rec ShortcutRecording
	plain := Shortcut{VK: 'A', Mods: ModShift, Label: "A"}
	rec.Press(plain, "", false, center)
	if rec.Phase.Kind != Rejected || rec.Phase.Rejection != NeedsModifier || registrar.registered != nil {
		t.Fatalf("%+v", rec.Phase)
	}
}

func TestLoadsDefaultWhenNothingIsSaved(t *testing.T) {
	registrar := &fakeRegistrar{}
	center := NewHotKeyCenter(registrar, SummonSettings(memoryStore{}))
	if !eq(center.Shortcut, ptr(DefaultSummon)) || !eq(registrar.registered, ptr(DefaultSummon)) {
		t.Fatal("default")
	}
}

func TestReportsWhenTheSavedShortcutCannotBeRegistered(t *testing.T) {
	registrar := &fakeRegistrar{takenByOtherApps: map[string]bool{DefaultSummon.Display(): true}}
	var events []HotKeyEvent
	center := NewHotKeyCenter(registrar, SummonSettings(memoryStore{}))
	center.OnEvent = func(e HotKeyEvent) { events = append(events, e) }
	center.RegisterSaved()
	if !center.RegistrationFailed || !reflect.DeepEqual(events, []HotKeyEvent{{Kind: HotKeyRegistrationFailed, Shortcut: DefaultSummon}}) {
		t.Fatalf("%+v", events)
	}
}

func TestInterceptorSwallowsThePress(t *testing.T) {
	center := NewHotKeyCenter(&fakeRegistrar{}, SummonSettings(memoryStore{}))
	pressed := 0
	center.OnPress = func() { pressed++ }
	center.Fire()
	center.Interceptor = func() bool { return true }
	center.Fire()
	center.Interceptor = nil
	center.Fire()
	if pressed != 2 {
		t.Fatal(pressed)
	}
}
