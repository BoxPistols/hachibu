package core

import (
	"strconv"
	"strings"
)

// Release は公開されている最新のバージョン。
type Release struct {
	// Version は "0.1.0"（タグの接頭辞は落とす）
	Version string
	URL     string
	// AssetURL は添付のHachibu-windows-x64.zip。無ければ、その場での置き換えはせずにページを開く
	AssetURL string
	// SHA256 はリリースの手順が本文の末尾に書くzipのSHA-256（小文字の16進）。
	// Windows向けはコード署名をしていないので、これと突き合わせられないときもその場では置き換えない
	SHA256 string
}

// WindowsAssetName はリリースに添付するx64向けのzipの名前（windows/build.sh --zip）
const WindowsAssetName = "Hachibu-windows-x64.zip"

// releaseDownloadPrefix は添付を取りに行ってよい場所。応答が偽物でも、ほかの場所からは取らない
const releaseDownloadPrefix = "https://github.com/BoxPistols/hachibu/releases/download/"

// CanInstallInPlace は、取得して確かめてから置き換えられるだけの情報があるか
func (r Release) CanInstallInPlace() bool { return r.AssetURL != "" && r.SHA256 != "" }

// LatestRelease はGitHubのリリースの一覧（新しい順）から、tagPrefixで始まる最新のバージョンを探す。
// Windows向けはmacOS向けと同じリポジトリに置き、タグをwindows-vで分ける（/releases/latestはmacOS向けを指す）。
// 下書きとプレリリース、バージョン番号として読めないタグ、GitHubのhttpsのページでないものは飛ばす。
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
		out := Release{Version: version, URL: page, AssetURL: assetURL(r["assets"], WindowsAssetName)}
		if body, ok := str(r["body"]); ok {
			out.SHA256 = SHA256InBody(body, WindowsAssetName)
		}
		return out, true
	}
	return Release{}, false
}

// assetURL は添付の一覧から、nameのファイルのダウンロードのURLを探す。このリポジトリのリリースのものでなければ使わない
func assetURL(raw any, name string) string {
	list, _ := raw.([]any)
	for _, a := range list {
		asset := dict(a)
		if n, _ := str(asset["name"]); n != name {
			continue
		}
		if u, _ := str(asset["browser_download_url"]); strings.HasPrefix(u, releaseDownloadPrefix) {
			return u
		}
	}
	return ""
}

// SHA256InBody はリリースの本文からnameのSHA-256を探す。リリースの手順はsha256sumの出力（"<16進64桁>  <名前>"）を本文の末尾に書く
func SHA256InBody(body, name string) string {
	for _, line := range strings.Split(body, "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || strings.TrimPrefix(f[1], "*") != name || len(f[0]) != 64 {
			continue
		}
		hex := strings.ToLower(f[0])
		if strings.Trim(hex, "0123456789abcdef") == "" {
			return hex
		}
	}
	return ""
}

// IsNewer はlatestがcurrentより新しいか。どちらかがバージョン番号として読めなければfalse（開発中のビルドに知らせない）。
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

// ShouldNotify は知らせるべき新しいバージョンか。すでに知らせたバージョン（notified）は、メニューには出すが通知はしない。
func ShouldNotify(r Release, current, notified string) bool {
	return IsNewer(r.Version, current) && (notified == "" || IsNewer(r.Version, notified))
}

// normalizedVersion: "v0.4.0" → "0.4.0"。数字とドットだけのものをバージョン番号とみなす
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
