//go:build windows

package app

import (
	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// hotKeyRegistrar はHotKeyCenterから使う登録器（RegisterHotKey）。1つの登録器が同時に登録するのは1つだけで、操作ごとに番号の違う登録器を持つ。
// どのアプリが前面でも、押されるとアプリの主な窓にWM_HOTKEYが届く
type hotKeyRegistrar struct {
	id         int32
	registered bool
	onPress    func()
}

var registrars = map[int32]*hotKeyRegistrar{}

func newHotKeyRegistrar(id int32) *hotKeyRegistrar {
	r := &hotKeyRegistrar{id: id, onPress: func() {}}
	registrars[id] = r
	return r
}

func (r *hotKeyRegistrar) Register(s core.Shortcut) bool {
	// 同じ番号の登録が残っていると失敗するので、先に外す
	r.Unregister()
	r.registered = w32.RegisterHotKey(mainHwnd, r.id, s.Win32Modifiers()|w32.MOD_NOREPEAT, s.VK)
	return r.registered
}

func (r *hotKeyRegistrar) Unregister() {
	if r.registered {
		w32.UnregisterHotKey(mainHwnd, r.id)
		r.registered = false
	}
}

// hotKeyPressed はWM_HOTKEYを受けたとき
func hotKeyPressed(id int32) {
	if r, ok := registrars[id]; ok {
		r.onPress()
	}
}
