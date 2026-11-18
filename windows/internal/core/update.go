package core

import (
	"strconv"
	"strings"
)

// Release は公開されている最新の版。
type Release struct {
	// Version は "0.1.0"（タグの接頭辞は落とす）
	Version string
	URL     string
}

// LatestRelease はGitHubのリリースの一覧（新しい順）から、tagPrefixで始まる最新の版を探す。
// Windows版はmacOS版と同じリポジトリに置き、タグをwindows-vで分ける（/releases/latestはmacOS版を指す）。
// 下書きとプレリリース、版の番号として読めないタグ、GitHubのhttpsのページでないものは飛ばす。
// ページはそのままShellExecuteで開くので、応答が偽物でもプログラムやファイルを開かないようにする。
// 見つからなければfalse（まだ出ていないのは失敗ではない）
func LatestRelease(list []any, tagPrefix string) (Release, bool) {
	for _, raw := range list {
		r := dict(raw)
		if r["draft"] == true || r["prerelease"] == true {
			continue
		}
		tag, _ := str(r["tag_name"])
		page, _ := str(r["html_url"])
		if !strings.HasPrefix(tag, tagPrefix) || !strings.HasPrefix(page, "https://github.com/") {
			continue
		}
		version, ok := normalizedVersion(strings.TrimPrefix(tag, tagPrefix))
		if !ok {
			continue
		}
		return Release{Version: version, URL: page}, true
	}
	return Release{}, false
}

// IsNewer はlatestがcurrentより新しいか。どちらかが版の番号として読めなければfalse（開発中のビルドに知らせない）。
func IsNewer(latest, current string) bool {
	a, okA := versionComponents(latest)
	b, okB := versionComponents(current)
	if !okA || !okB {
		return false
	}
	for i := range max(len(a), len(b)) {
		x, y := 0, 0
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x > y
		}
	}
	return false
}

// ShouldNotify は知らせるべき新しい版か。すでに知らせた版（notified）は、メニューには出すが通知はしない。
func ShouldNotify(r Release, current, notified string) bool {
	return IsNewer(r.Version, current) && (notified == "" || IsNewer(r.Version, notified))
}

// normalizedVersion: "v0.4.0" → "0.4.0"。数字とドットだけのものを版の番号とみなす
func normalizedVersion(tag string) (string, bool) {
	s := trimSpace(tag)
	if strings.HasPrefix(s, "v") || strings.HasPrefix(s, "V") {
		s = s[1:]
	}
	if _, ok := versionComponents(s); !ok {
		return "", false
	}
	return s, true
}

func versionComponents(s string) ([]int, bool) {
	var out []int
	for _, p := range strings.Split(s, ".") {
		if p == "" {
			return nil, false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return nil, false
			}
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		out = append(out, n)
	}
	return out, true
}
