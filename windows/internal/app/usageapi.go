//go:build windows

package app

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/BoxPistols/hachibu/windows/internal/core"
)

// 使用率APIの呼び出し。利用者が「使用率をAPIから取得」を有効にしたときだけ使う。
//
// Claude Codeが保存したOAuthのトークン（.claude\.credentials.json）を読み、Anthropicの使用率のエンドポイントへGETする。
// トークンはこの関数の中でしか使わず、保存もログへの出力もしない。User-AgentはClaude Codeを名乗らない
const usageURL = "https://api.anthropic.com/api/oauth/usage"

type fetchResult struct {
	limits []core.UsageLimit
	// failure はメニューに出す短い理由。空なら成功
	failure string
}

func fetchUsage(dirs []claudeDir) fetchResult {
	token := accessToken(dirs)
	if token == "" {
		return fetchResult{failure: "no Claude Code login found"}
	}
	req, err := http.NewRequest(http.MethodGet, usageURL, nil)
	if err != nil {
		return fetchResult{failure: "network"}
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("User-Agent", "HachibuWindows/"+Version)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fetchResult{failure: "network"}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fetchResult{failure: "HTTP " + strconv.Itoa(resp.StatusCode)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fetchResult{failure: "network"}
	}
	var obj map[string]any
	if json.Unmarshal(body, &obj) != nil {
		return fetchResult{failure: "unexpected response"}
	}
	limits := core.ParseUsageAPI(obj)
	if len(limits) == 0 {
		return fetchResult{failure: "unexpected response"}
	}
	return fetchResult{limits: limits}
}

// accessToken はClaude Codeのログイン。WindowsとWSLの両方にあれば、最後に書き換えられた方（いま使っている方）を使う
func accessToken(dirs []claudeDir) string {
	var best string
	var bestMod time.Time
	for _, d := range dirs {
		path := filepath.Join(d.Path, ".credentials.json")
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().After(bestMod) {
			continue
		}
		data, err := readShared(path, 1<<20)
		if err != nil {
			continue
		}
		var obj struct {
			ClaudeAiOauth struct {
				AccessToken string `json:"accessToken"`
			} `json:"claudeAiOauth"`
		}
		if json.Unmarshal(data, &obj) != nil || obj.ClaudeAiOauth.AccessToken == "" {
			continue
		}
		best = obj.ClaudeAiOauth.AccessToken
		bestMod = info.ModTime()
	}
	return best
}
