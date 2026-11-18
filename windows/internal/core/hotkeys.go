package core

// HotKeyRegistrar は、どのアプリが前面でも効くショートカットを実際に登録する部分（アプリではRegisterHotKey）。テストでは偽物に差し替える。
type HotKeyRegistrar interface {
	// Register は登録できたらtrue。呼ぶ側が先にUnregisterしている
	Register(Shortcut) bool
	Unregister()
}

// SettingsStore は設定の保存先。アプリではsettings.json、テストではメモリ上の辞書（ディスクに書かない）。
type SettingsStore interface {
	Bool(key string) bool
	String(key string) string
	Set(key string, value any)
}

// ShortcutSettings はショートカットの保存先。
type ShortcutSettings struct {
	store       SettingsStore
	shortcutKey string
	disabledKey string
	fallback    *Shortcut
}

// SummonSettings はバーの呼び出し用。一度も設定していなければCtrl+Alt+/
func SummonSettings(store SettingsStore) *ShortcutSettings {
	d := DefaultSummon
	return &ShortcutSettings{store: store, shortcutKey: "summonShortcut", disabledKey: "summonShortcutDisabled", fallback: &d}
}

// LayoutCycleSettings は表示を順に切り替えるショートカットの保存先。利用者が設定するまでは無し
func LayoutCycleSettings(store SettingsStore) *ShortcutSettings {
	return &ShortcutSettings{store: store, shortcutKey: "cycleShortcut", disabledKey: "cycleShortcutDisabled"}
}

// Load は、無効にしていればnil。一度も設定していなければ既定。
func (s *ShortcutSettings) Load() *Shortcut {
	if s.store.Bool(s.disabledKey) {
		return nil
	}
	if saved, ok := DecodeShortcut(s.store.String(s.shortcutKey)); ok {
		return &saved
	}
	if s.fallback == nil {
		return nil
	}
	f := *s.fallback
	return &f
}

func (s *ShortcutSettings) Save(sc Shortcut) {
	s.store.Set(s.shortcutKey, EncodeShortcut(sc))
	s.store.Set(s.disabledKey, false)
}

func (s *ShortcutSettings) SaveDisabled() {
	s.store.Set(s.disabledKey, true)
}

// HotKeyEventKind はHotKeyCenterが知らせる出来事（操作ログに残す）。
type HotKeyEventKind int

const (
	HotKeyCommitted HotKeyEventKind = iota
	HotKeyDisabled
	HotKeyRegistrationFailed
)

type HotKeyEvent struct {
	Kind     HotKeyEventKind
	Shortcut Shortcut
}

// HotKeyCenter は呼び出しショートカットの登録と保存。
//
// 「登録」と「保存」を分けてある。記録の窓ではTryOutで登録だけを試し、届くことを確かめてからCommitで保存する。
// 保存済みの値は確かめるまで変わらないので、取り消すときはRegisterSavedで登録し直すだけで窓を開く前に戻る。
type HotKeyCenter struct {
	// Shortcut は保存済みの組み合わせ。無効にしていればnil
	Shortcut *Shortcut
	// RegistrationFailed は保存済みの組み合わせを登録できなかった（他のアプリが使用中）
	RegistrationFailed bool
	// OnPress は押されたとき（帯の呼び出し）
	OnPress func()
	// Interceptor は記録の窓を開いている間だけ使う。trueを返したらOnPressを呼ばない
	Interceptor func() bool
	OnEvent     func(HotKeyEvent)

	registrar HotKeyRegistrar
	settings  *ShortcutSettings
}

func NewHotKeyCenter(registrar HotKeyRegistrar, settings *ShortcutSettings) *HotKeyCenter {
	c := &HotKeyCenter{registrar: registrar, settings: settings, Shortcut: settings.Load()}
	c.RegisterSaved()
	return c
}

func (c *HotKeyCenter) emit(e HotKeyEvent) {
	if c.OnEvent != nil {
		c.OnEvent(e)
	}
}

// Fire は登録したショートカットが押されたとき、登録器から呼ぶ。
func (c *HotKeyCenter) Fire() {
	if c.Interceptor != nil && c.Interceptor() {
		return
	}
	if c.OnPress != nil {
		c.OnPress()
	}
}

// TryOut は保存はせずに登録だけ試す。成功している間は、保存済みの組み合わせの代わりにこれが効く。
func (c *HotKeyCenter) TryOut(candidate Shortcut) bool {
	c.registrar.Unregister()
	return c.registrar.Register(candidate)
}

// Commit は届くことを確かめた組み合わせを保存する。TryOutで登録したものがそのまま効き続ける。
func (c *HotKeyCenter) Commit(confirmed Shortcut) {
	s := confirmed
	c.Shortcut = &s
	c.RegistrationFailed = false
	c.settings.Save(confirmed)
	c.emit(HotKeyEvent{Kind: HotKeyCommitted, Shortcut: confirmed})
}

func (c *HotKeyCenter) Disable() {
	c.registrar.Unregister()
	c.Shortcut = nil
	c.RegistrationFailed = false
	c.settings.SaveDisabled()
	c.emit(HotKeyEvent{Kind: HotKeyDisabled})
}

// Suspend は記録の窓を開いている間に外す（今の組み合わせを押しても呼び出しが走らず、記録に回るように）。
func (c *HotKeyCenter) Suspend() {
	c.registrar.Unregister()
}

// RegisterSaved は保存済みの組み合わせを登録し直す。試していた組み合わせはこれで外れる。
func (c *HotKeyCenter) RegisterSaved() {
	c.registrar.Unregister()
	if c.Shortcut == nil {
		c.RegistrationFailed = false
		return
	}
	c.RegistrationFailed = !c.registrar.Register(*c.Shortcut)
	if c.RegistrationFailed {
		c.emit(HotKeyEvent{Kind: HotKeyRegistrationFailed, Shortcut: *c.Shortcut})
	}
}
