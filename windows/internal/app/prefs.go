//go:build windows

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
	"golang.org/x/sys/windows"
)

// dataDir は設定、statusLineの値、操作ログを置く場所（%LOCALAPPDATA%\Hachibu）。消せばアプリの持ち物はすべて消える
func dataDir() string {
	base, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0)
	if err != nil {
		base = os.Getenv("LOCALAPPDATA")
	}
	dir := filepath.Join(base, "Hachibu")
	os.MkdirAll(dir, 0o755)
	return dir
}

// statusLineFile はstatusline.sh（またはstatusline.ps1）が書くファイル
func statusLineFile() string { return filepath.Join(dataDir(), "statusline.json") }

// Prefs は利用者の設定。キーはここにだけ書く。値はsettings.jsonに、変えるたびに書き出す。
// 撮影用モード（HACHIBU_SOURCE=demo）と試験用（HACHIBU_SUITE=名前）では、利用者の設定に触れないよう別のファイルを使う
type Prefs struct {
	mu     sync.Mutex
	path   string
	values map[string]any
}

func loadPrefs() *Prefs {
	name := "settings.json"
	if suite := os.Getenv("HACHIBU_SUITE"); suite != "" {
		name = "settings-" + filepath.Base(suite) + ".json"
	} else if isDemo() {
		name = "settings-demo.json"
	}
	p := &Prefs{path: filepath.Join(dataDir(), name), values: map[string]any{}}
	if data, err := os.ReadFile(p.path); err == nil {
		json.Unmarshal(data, &p.values)
		if p.values == nil {
			p.values = map[string]any{}
		}
	}
	return p
}

// save は書きかけのファイルを読ませないよう、別名で書いてから置き換える
func (p *Prefs) save() error {
	data, err := json.MarshalIndent(p.values, "", "  ")
	if err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}

func (p *Prefs) get(key string) (any, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.values[key]
	return v, ok
}

// Set は値を変えて書き出す。nilはキーを消す。
// 値はJSONを通してから持つ。ファイルから読んだときと同じ型（数はfloat64、辞書はmap[string]any）にしないと、
// 書いた直後と再起動の後で読み方が変わる（int32で入れた帯の位置を、float64として読めなかった）
func (p *Prefs) Set(key string, value any) {
	p.SetChecked(key, value)
}

// SetChecked はSetと同じで、ファイルに書けなかったらエラーを返す。値は書けなくても、動いている間は変えたものを使う
func (p *Prefs) SetChecked(key string, value any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if value == nil {
		delete(p.values, key)
	} else if data, err := json.Marshal(value); err == nil {
		var normalized any
		if json.Unmarshal(data, &normalized) == nil {
			p.values[key] = normalized
		}
	}
	return p.save()
}

func (p *Prefs) Bool(key string) bool {
	v, _ := p.get(key)
	b, _ := v.(bool)
	return b
}

func (p *Prefs) String(key string) string {
	v, _ := p.get(key)
	s, _ := v.(string)
	return s
}

func (p *Prefs) number(key string) (float64, bool) {
	v, _ := p.get(key)
	f, ok := v.(float64)
	return f, ok
}

var opacityChoices = []float64{1.0, 0.85, 0.7, 0.5, 0.3}

// UsageAPIConsent は使用率APIを使うことへの同意。nilはまだ尋ねていない
func (p *Prefs) UsageAPIConsent() *bool {
	v, ok := p.get("usageAPIConsent")
	if b, isBool := v.(bool); ok && isBool {
		return &b
	}
	return nil
}

func (p *Prefs) UsageAPIEnabled() bool {
	c := p.UsageAPIConsent()
	return c != nil && *c
}

// Language は画面の言語（"en"、"ja"）。空はWindowsに合わせる
func (p *Prefs) Language() string { return p.String("language") }

func (p *Prefs) RestLayout() core.StripLayout {
	if l, ok := core.ParseLayout(p.String("restLayout")); ok {
		return l
	}
	return core.LayoutFull
}

func (p *Prefs) Opacity() float64 {
	if v, ok := p.number("opacity"); ok {
		for _, c := range opacityChoices {
			if c == v {
				return v
			}
		}
	}
	return 1.0
}

func (p *Prefs) Thresholds() core.Thresholds {
	w, _ := p.number("warningThreshold")
	c, _ := p.number("criticalThreshold")
	if t, ok := core.NewThresholds(int(w), int(c)); ok {
		return t
	}
	return core.DefaultThresholds
}

func (p *Prefs) SetThresholds(t core.Thresholds) {
	p.Set("warningThreshold", t.Warning)
	p.Set("criticalThreshold", t.Critical)
}

func (p *Prefs) TrayStyle() core.TrayStyle {
	if t, ok := core.ParseTrayStyle(p.String("trayStyle")); ok {
		return t
	}
	return core.TrayIcon
}

// UpdateAutoCheck は新しいバージョンを1日に1回確認するか。既定は確認する
func (p *Prefs) UpdateAutoCheck() bool {
	v, ok := p.get("updateAutoCheck")
	if b, isBool := v.(bool); ok && isBool {
		return b
	}
	return true
}

func (p *Prefs) LastUpdateCheck() time.Time {
	if t, err := time.Parse(time.RFC3339, p.String("lastUpdateCheck")); err == nil {
		return t
	}
	return time.Time{}
}

func (p *Prefs) SetLastUpdateCheck(t time.Time) {
	if t.IsZero() {
		p.Set("lastUpdateCheck", nil)
	} else {
		p.Set("lastUpdateCheck", t.UTC().Format(time.RFC3339))
	}
}

// stripAnchor は帯の置き場所。基準にする端と、その基準点の座標（画面の座標、画素）
type stripAnchor struct {
	Edge string `json:"edge"`
	X    int32  `json:"x"`
	Y    int32  `json:"y"`
}

func (p *Prefs) StripAnchor() (stripAnchor, bool) {
	v, ok := p.get("stripAnchor")
	if !ok {
		return stripAnchor{}, false
	}
	m, _ := v.(map[string]any)
	edge, _ := m["edge"].(string)
	x, okX := m["x"].(float64)
	y, okY := m["y"].(float64)
	if edge == "" || !okX || !okY {
		return stripAnchor{}, false
	}
	return stripAnchor{Edge: edge, X: int32(x), Y: int32(y)}, true
}

func (p *Prefs) SetStripAnchor(a *stripAnchor) {
	if a == nil {
		p.Set("stripAnchor", nil)
		return
	}
	p.Set("stripAnchor", map[string]any{"edge": a.Edge, "x": a.X, "y": a.Y})
}
