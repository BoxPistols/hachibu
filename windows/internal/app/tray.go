//go:build windows

package app

import (
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"github.com/BoxPistols/hachibu/windows/internal/w32"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Tray はタスクバーの通知領域（システムトレイ）のアイコン。macOS版のメニューバーの項目にあたる。
// トレイには文字を並べられないので、使用率は絵（週枠のメーター）か数字1つで示し、全文はツールチップとメニューに出す
type Tray struct {
	owner w32.HWND
	added bool
	icon  w32.HICON
	key   string
}

const trayID = 1

func newTray(owner w32.HWND) *Tray {
	return &Tray{owner: owner}
}

func (t *Tray) data() w32.NOTIFYICONDATAW {
	return w32.NOTIFYICONDATAW{Wnd: t.owner, ID: trayID}
}

// Refresh はアイコンとツールチップを今の値で描き直す。エクスプローラーが再起動したときはaddedをfalseにしてから呼ぶ
func (t *Tray) Refresh(style core.TrayStyle, limits []core.UsageLimit, th core.Thresholds, tip string) {
	dpi := w32.GetDpiForSystem()
	size := w32.GetSystemMetricsForDpi(w32.SM_CXSMICON, dpi)
	if size <= 0 {
		size = 16
	}
	dark := taskbarIsDark()
	// 同じ絵なら作り直さない（2秒ごとの更新で、アイコンを作っては捨てないように）
	key := strings.Join([]string{string(style), core.UsageOnlyText(limits), itoa(th.Warning), itoa(th.Critical), itoa(int(size)), boolText(dark)}, "|")
	d := t.data()
	d.Flags = w32.NIF_TIP | w32.NIF_SHOWTIP | w32.NIF_MESSAGE
	d.CallbackMessage = msgTray
	setUTF16(d.Tip[:], tip)
	if key != t.key || !t.added {
		c := paintTrayIcon(style, limits, th, size, dark)
		icon := c.HIcon()
		c.Free()
		if t.icon != 0 {
			defer w32.DestroyIcon(t.icon)
		}
		t.icon = icon
		t.key = key
	}
	d.Flags |= w32.NIF_ICON
	d.Icon = t.icon
	if !t.added {
		if w32.ShellNotifyIcon(w32.NIM_ADD, &d) {
			t.added = true
			v := t.data()
			v.Version = w32.NOTIFYICON_VERSION_4
			w32.ShellNotifyIcon(w32.NIM_SETVERSION, &v)
		}
		return
	}
	if !w32.ShellNotifyIcon(w32.NIM_MODIFY, &d) {
		// エクスプローラーが落ちて戻ったなどで消えていれば、足し直す
		t.added = false
		t.key = ""
	}
}

// Notify は通知（Windowsのトースト）を出す。押されるとNIN_BALLOONUSERCLICKが届く
func (t *Tray) Notify(title, body string) {
	if !t.added {
		return
	}
	d := t.data()
	d.Flags = w32.NIF_INFO
	d.InfoFlags = w32.NIIF_INFO | w32.NIIF_NOSOUND
	setUTF16(d.InfoTitle[:], title)
	setUTF16(d.Info[:], body)
	w32.ShellNotifyIcon(w32.NIM_MODIFY, &d)
}

func (t *Tray) Remove() {
	if t.added {
		d := t.data()
		w32.ShellNotifyIcon(w32.NIM_DELETE, &d)
		t.added = false
	}
}

// setUTF16 は固定長の欄に文字列を入れる。入り切らなければ切り詰め、最後をNULにする
func setUTF16(dst []uint16, s string) {
	u := utf16.Encode([]rune(s))
	if len(u) > len(dst)-1 {
		u = u[:len(dst)-1]
	}
	n := copy(dst, u)
	dst[n] = 0
}

func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// taskbarIsDark はタスクバーが暗い色か（「設定」＞「個人用設定」＞「色」の「Windowsモード」）
func taskbarIsDark() bool {
	return !themeValue("SystemUsesLightTheme")
}

// appsAreDark はアプリの窓が暗い色か（「アプリモード」）
func appsAreDark() bool {
	return !themeValue("AppsUseLightTheme")
}

func themeValue(name string) bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue(name)
	if err != nil {
		// 値が無いのは古いWindows。タスクバーは暗く、アプリは明るいのが既定
		return name == "AppsUseLightTheme"
	}
	return v != 0
}

// trayIconHidden は、Windows 11がトレイのアイコンを「隠れているアイコン」の中にしまっているか。
// HKCU\Control Panel\NotifyIconSettingsの、このアプリの項目のIsPromotedが1ならタスクバーに出ている。項目が無ければ分からないのでfalse
func trayIconHidden() bool {
	exe, err := windowsExecutable()
	if err != nil {
		return false
	}
	root, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\NotifyIconSettings`, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return false
	}
	defer root.Close()
	names, _ := root.ReadSubKeyNames(-1)
	for _, name := range names {
		k, err := registry.OpenKey(root, name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		path, _, _ := k.GetStringValue("ExecutablePath")
		promoted, _, perr := k.GetIntegerValue("IsPromoted")
		k.Close()
		if strings.EqualFold(expandKnownFolder(path), exe) {
			return perr != nil || promoted == 0
		}
	}
	return false
}

// expandKnownFolder は"{F1B32785-6FBA-4FCF-9D55-7B8E7F157091}\Programs\..."のような、既知のフォルダの番号で始まる道筋を展開する
func expandKnownFolder(path string) string {
	if !strings.HasPrefix(path, "{") {
		return path
	}
	end := strings.Index(path, "}")
	if end < 0 {
		return path
	}
	guid, err := windows.GUIDFromString(path[:end+1])
	if err != nil {
		return path
	}
	base, err := windows.KnownFolderPath((*windows.KNOWNFOLDERID)(&guid), 0)
	if err != nil {
		return path
	}
	return filepath.Join(base, path[end+1:])
}
