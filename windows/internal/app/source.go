//go:build windows

package app

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
)

// Source は基本表示の供給元。ほかの道具に頼らず、帯の状態の枠に出す値を集める。
//
//   - モデル名とeffort: statusLineのJSON（statusline.shが書く）、最後の会話記録、settings.json
//   - 使用率: 利用者が有効にしていれば使用率API（5分ごと）、そうでなければstatusLineのrate_limits
//
// 読み取りは別のゴルーチンで行う（WSLのファイルは\\wsl.localhost越しなので遅いことがある）。
// 集めた値はOnUpdateでUIのスレッドへ渡し、帯の文字はUIのスレッドで組み立てる（言語の切り替えをすぐに反映するため）
type Source struct {
	OnUpdate func(core.SourceState, []claudeDir)

	consent atomic.Bool
	kick    chan struct{}
	fetched chan fetchResult

	// ここから下は読み取りのゴルーチンだけが触る
	st              core.SourceState
	statusLineMod   time.Time
	transcriptPath  string
	lastScan        time.Time
	dirs            []claudeDir
	settingsPath    string
	settingsMod     time.Time
	lastFetch       time.Time
	fetching        bool
	consentObserved bool
}

const (
	tickInterval = 2 * time.Second
	// 会話記録は全プロジェクトを見るので、頻繁には探さない
	transcriptScanInterval = 30 * time.Second
	// 表示のために頻繁に叩かない。失敗しても次の周期まで再試行しない
	fetchInterval = 5 * time.Minute
)

// 画像などの大きな行が続くと、末尾にモデル名が無いことがある。見つかるまで読む範囲を広げる
var transcriptTailSizes = []int64{64 << 10, 512 << 10, 4 << 20}

func newSource(useAPI bool) *Source {
	s := &Source{kick: make(chan struct{}, 1), fetched: make(chan fetchResult, 1)}
	s.consent.Store(useAPI)
	return s
}

func (s *Source) Start() {
	go s.run()
}

// SetUsageAPI は「使用率をAPIから取得」を切り替えたとき。有効にしたらすぐ取りに行く
func (s *Source) SetUsageAPI(on bool) {
	s.consent.Store(on)
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *Source) run() {
	ticker := time.NewTicker(tickInterval)
	s.tick(true)
	for {
		select {
		case <-ticker.C:
			s.tick(false)
		case <-s.kick:
			s.consentChanged()
			s.tick(false)
		case r := <-s.fetched:
			s.applyFetch(r)
			s.publish()
		}
	}
}

func (s *Source) tick(first bool) {
	now := time.Now()
	s.readStatusLine()
	if first || now.Sub(s.lastScan) >= transcriptScanInterval {
		s.lastScan = now
		s.scan()
	}
	s.readSettings()
	s.maybeFetch(now)
	s.publish()
}

// publish は2秒ごとに今の値をUIへ渡す。値が変わらなくても、時間が経って古くなった使用率を薄くするために渡す
func (s *Source) publish() {
	st := s.st
	st.UseAPI = s.consent.Load()
	st.StatusLineLimits = append([]core.UsageLimit(nil), s.st.StatusLineLimits...)
	st.APILimits = append([]core.UsageLimit(nil), s.st.APILimits...)
	if s.st.StatusLine != nil {
		snap := *s.st.StatusLine
		st.StatusLine = &snap
	}
	dirs := append([]claudeDir(nil), s.dirs...)
	Post(func() {
		if s.OnUpdate != nil {
			s.OnUpdate(st, dirs)
		}
	})
}

// MARK: - 読み取り

func (s *Source) readStatusLine() {
	path := statusLineFile()
	info, err := os.Stat(path)
	if err != nil || info.ModTime().Equal(s.statusLineMod) {
		return
	}
	s.statusLineMod = info.ModTime()
	data, err := readShared(path, 1<<20)
	if err != nil {
		return
	}
	var obj map[string]any
	if json.Unmarshal(data, &obj) != nil || obj == nil {
		return
	}
	parsed := core.ParseStatusLine(obj)
	s.st.StatusLine = &parsed
	s.st.StatusLineModified = info.ModTime()
	// rate_limitsは最初の応答の前には入らない。直前の値を空で上書きしない
	if len(parsed.Limits) > 0 {
		s.st.StatusLineLimits = parsed.Limits
		s.st.StatusLineLimitsAt = info.ModTime()
	}
}

// scan は.claudeフォルダを探し直し、最後に書かれた会話記録の末尾から、最後に使われたモデルとeffortを読む
func (s *Source) scan() {
	s.dirs = findClaudeDirs()
	var newestPath string
	var newestMod time.Time
	newestDir := ""
	for _, d := range s.dirs {
		projects, err := os.ReadDir(filepath.Join(d.Path, "projects"))
		if err != nil {
			continue
		}
		for _, p := range projects {
			if !p.IsDir() {
				continue
			}
			files, err := os.ReadDir(filepath.Join(d.Path, "projects", p.Name()))
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !strings.HasSuffix(f.Name(), ".jsonl") {
					continue
				}
				info, err := f.Info()
				if err != nil {
					continue
				}
				if info.ModTime().After(newestMod) {
					newestMod = info.ModTime()
					newestPath = filepath.Join(d.Path, "projects", p.Name(), f.Name())
					newestDir = d.Path
				}
			}
		}
	}
	// 設定は、最後に使われた会話記録と同じ場所のものを読む。会話記録が無ければ最初に見つかった場所
	switch {
	case newestDir != "":
		s.settingsPath = filepath.Join(newestDir, "settings.json")
	case len(s.dirs) > 0:
		s.settingsPath = filepath.Join(s.dirs[0].Path, "settings.json")
	default:
		s.settingsPath = ""
	}
	if newestPath == "" || (newestPath == s.transcriptPath && newestMod.Equal(s.st.TranscriptModified)) {
		return
	}
	s.transcriptPath = newestPath
	s.st.TranscriptModified = newestMod
	f, err := openShared(newestPath)
	if err != nil {
		return
	}
	defer f.Close()
	size, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return
	}
	for _, tail := range transcriptTailSizes {
		length := min(tail, size)
		buf := make([]byte, length)
		if _, err := f.ReadAt(buf, size-length); err != nil && err != io.EOF {
			return
		}
		text := string(buf)
		if id := core.LastModelID(text); id != "" {
			s.st.TranscriptModelID = id
			// 同じ応答の行に入っている。古い会話記録にはeffortが無いので、そのときは設定に任せる
			s.st.TranscriptEffort = core.LastEffort(text)
			return
		}
		if length == size {
			return
		}
	}
}

func (s *Source) readSettings() {
	if s.settingsPath == "" {
		s.st.SettingsModel, s.st.SettingsEffort = "", ""
		return
	}
	info, err := os.Stat(s.settingsPath)
	if err != nil {
		s.st.SettingsModel, s.st.SettingsEffort = "", ""
		s.settingsMod = time.Time{}
		return
	}
	if info.ModTime().Equal(s.settingsMod) {
		return
	}
	s.settingsMod = info.ModTime()
	data, err := readShared(s.settingsPath, 4<<20)
	if err != nil {
		return
	}
	var obj map[string]any
	if json.Unmarshal(data, &obj) != nil {
		return
	}
	s.st.SettingsModel, _ = obj["model"].(string)
	s.st.SettingsEffort, _ = obj["effortLevel"].(string)
}

// MARK: - 使用率API

func (s *Source) consentChanged() {
	s.lastFetch = time.Time{}
	s.st.APIFailure = ""
	if !s.consent.Load() {
		s.st.APILimits = nil
		s.st.APIAt = time.Time{}
	}
}

func (s *Source) maybeFetch(now time.Time) {
	if !s.consent.Load() || s.fetching || now.Sub(s.lastFetch) < fetchInterval {
		return
	}
	// 取得前に時刻を進めておく（失敗時に毎回叩き直さない）
	s.lastFetch = now
	s.fetching = true
	dirs := append([]claudeDir(nil), s.dirs...)
	go func() { s.fetched <- fetchUsage(dirs) }()
}

func (s *Source) applyFetch(r fetchResult) {
	s.fetching = false
	// 取得中に無効にされていたら、届いた値は使わずに捨てる
	if !s.consent.Load() {
		return
	}
	if r.failure != "" {
		s.st.APIFailure = r.failure
		return
	}
	s.st.APILimits = r.limits
	s.st.APIAt = time.Now()
	s.st.APIFailure = ""
}

// demoState は撮影と見た目の確認用の架空の値（HACHIBU_SOURCE=demo）。HACHIBU_DEMO=criticalで5時間枠が赤の場面になる
func demoState(now time.Time) core.SourceState {
	fiveHour := 42
	if os.Getenv("HACHIBU_DEMO") == "critical" {
		fiveHour = 93
	}
	limits := []core.UsageLimit{
		{Kind: core.KindFiveHour, Percent: fiveHour, ResetsAt: now.Add(2 * time.Hour)},
		{Kind: core.KindWeekly, Percent: 76, ResetsAt: now.Add(3 * 24 * time.Hour)},
		{Kind: core.KindModel, Model: "Fable", Percent: 93, ResetsAt: now.Add(2 * 24 * time.Hour)},
	}
	return core.SourceState{
		StatusLine:         &core.StatusLineSnapshot{Model: "Opus5 1M", Effort: "xhigh", Limits: limits},
		StatusLineModified: now,
		StatusLineLimits:   limits,
		StatusLineLimitsAt: now,
	}
}

func isDemo() bool { return os.Getenv("HACHIBU_SOURCE") == "demo" }
