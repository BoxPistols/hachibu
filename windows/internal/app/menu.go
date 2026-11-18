//go:build windows

package app

import (
	"strings"

	"github.com/BoxPistols/hachibu/windows/internal/w32"
)

// menuSet は1回開くメニュー（入れ子を含む）の項目の番号と、選んだとき・乗せたときの処理を持つ。
// 開いている間に届くWM_MENUSELECTで、乗せた項目の仮の表示を出す
type menuSet struct {
	actions  map[uint32]func()
	previews map[uint32]func()
	nextID   uint32
	bitmaps  []w32.HBITMAP
	scale    float32
	// dump は開発用の書き出し（--snapshot）で、メニューを作らずに項目を文字に並べる
	dump *strings.Builder
}

type menu struct {
	set   *menuSet
	h     w32.HMENU
	depth int
}

func newMenuSet(scale float32) *menuSet {
	return &menuSet{actions: map[uint32]func(){}, previews: map[uint32]func(){}, nextID: 100, scale: scale}
}

func (s *menuSet) root() *menu {
	m := &menu{set: s}
	if s.dump == nil {
		m.h = w32.CreatePopupMenu()
	}
	return m
}

// free はメニューに付けた絵を消す（メニューそのものはTrackPopupMenuの後にDestroyMenuで消す）
func (s *menuSet) free() {
	for _, b := range s.bitmaps {
		w32.DeleteObject(w32.HGDIOBJ(b))
	}
	s.bitmaps = nil
}

func (m *menu) id() uint32 {
	m.set.nextID++
	return m.set.nextID
}

func (m *menu) write(mark, title string) {
	b := m.set.dump
	b.WriteString(strings.Repeat("    ", m.depth))
	if mark != "" {
		b.WriteString(mark + " ")
	}
	b.WriteString(strings.ReplaceAll(title, "\t", "    "))
	b.WriteString("\n")
}

func (m *menu) insert(info w32.MENUITEMINFOW) {
	w32.InsertMenuItem(m.h, &info)
}

// item は押せる項目。checkedならチェックを付ける。shortcutは右端に出すキーの組み合わせ
func (m *menu) item(title string, checked bool, action func()) {
	m.itemWithShortcut(title, "", checked, action)
}

func (m *menu) itemWithShortcut(title, shortcut string, checked bool, action func()) {
	text := title
	if shortcut != "" {
		text += "\t" + shortcut
	}
	if m.set.dump != nil {
		mark := "  "
		if checked {
			mark = "✓"
		}
		m.write(mark, text)
		return
	}
	id := m.id()
	m.set.actions[id] = action
	info := w32.MENUITEMINFOW{Mask: w32.MIIM_ID | w32.MIIM_STRING | w32.MIIM_STATE | w32.MIIM_FTYPE, ID: id, TypeData: w32.UTF16(escapeMenu(text))}
	if checked {
		info.State = w32.MFS_CHECKED
	}
	m.insert(info)
}

// choice は選択肢の1つ（丸印で選ばれていることを示す）。乗せている間はpreviewで帯に仮に反映し、選ぶとcommitで確定する
func (m *menu) choice(title string, selected bool, preview, commit func()) {
	if m.set.dump != nil {
		mark := "  "
		if selected {
			mark = "●"
		}
		m.write(mark, title)
		return
	}
	id := m.id()
	m.set.actions[id] = commit
	if preview != nil {
		m.set.previews[id] = preview
	}
	info := w32.MENUITEMINFOW{Mask: w32.MIIM_ID | w32.MIIM_STRING | w32.MIIM_STATE | w32.MIIM_FTYPE,
		Type: w32.MFT_RADIOCHECK, ID: id, TypeData: w32.UTF16(escapeMenu(title))}
	if selected {
		info.State = w32.MFS_CHECKED
	}
	m.insert(info)
}

// info は読むための行（使用率、読み方、状態の知らせ）。押しても何もしない。dotがあれば行の左に色の点を付ける
func (m *menu) info(title string, dot *w32.ARGB, diameter float32) {
	if m.set.dump != nil {
		mark := "  "
		if dot != nil {
			mark = "•"
		}
		m.write(mark, title)
		return
	}
	id := m.id()
	info := w32.MENUITEMINFOW{Mask: w32.MIIM_ID | w32.MIIM_STRING | w32.MIIM_FTYPE, ID: id, TypeData: w32.UTF16(escapeMenu(title))}
	if dot != nil {
		bmp := dotBitmap(*dot, diameter, m.set.scale)
		m.set.bitmaps = append(m.set.bitmaps, bmp)
		info.Mask |= w32.MIIM_BITMAP
		info.BmpItem = bmp
	}
	m.insert(info)
}

// disabled は選べない項目（見出しと、選べない選択肢）。selectedなら丸印を付ける
func (m *menu) disabled(title string, selected bool) {
	if m.set.dump != nil {
		mark := "  "
		if selected {
			mark = "●"
		}
		m.write(mark, "("+title+")")
		return
	}
	info := w32.MENUITEMINFOW{Mask: w32.MIIM_ID | w32.MIIM_STRING | w32.MIIM_STATE | w32.MIIM_FTYPE,
		Type: w32.MFT_RADIOCHECK, ID: m.id(), State: w32.MFS_GRAYED, TypeData: w32.UTF16(escapeMenu(title))}
	if selected {
		info.State |= w32.MFS_CHECKED
	}
	m.insert(info)
}

func (m *menu) separator() {
	if m.set.dump != nil {
		m.write("", strings.Repeat("-", 24))
		return
	}
	m.insert(w32.MENUITEMINFOW{Mask: w32.MIIM_FTYPE, Type: w32.MFT_SEPARATOR})
}

// submenu は入れ子のメニュー。入れ子の中の項目に乗せたときも仮の表示を出す
func (m *menu) submenu(title string) *menu {
	sub := &menu{set: m.set, depth: m.depth + 1}
	if m.set.dump != nil {
		m.write("  ", title+" ▸")
		return sub
	}
	sub.h = w32.CreatePopupMenu()
	m.insert(w32.MENUITEMINFOW{Mask: w32.MIIM_ID | w32.MIIM_STRING | w32.MIIM_SUBMENU, ID: m.id(), SubMenu: sub.h,
		TypeData: w32.UTF16(escapeMenu(title))})
	return sub
}

// escapeMenu はメニューの文言の&を、アクセスキーの印ではなく文字として出す
func escapeMenu(s string) string {
	return strings.ReplaceAll(s, "&", "&&")
}
