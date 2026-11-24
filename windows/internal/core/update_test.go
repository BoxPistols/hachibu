package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func releases(t *testing.T, text string) []any {
	t.Helper()
	var list []any
	if err := json.Unmarshal([]byte(text), &list); err != nil {
		t.Fatal(err)
	}
	return list
}

func TestFindsTheNewestWindowsRelease(t *testing.T) {
	// 一覧は新しい順。macOS向け（v）とLite（lite-v）のリリースも混ざる
	list := releases(t, `[
	  {"tag_name": "v0.6.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/v0.6.0", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.3.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/w030", "draft": true, "prerelease": false},
	  {"tag_name": "windows-v0.2.1", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/w021", "draft": false, "prerelease": true},
	  {"tag_name": "lite-v0.4.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/lite", "draft": false, "prerelease": false},
	  {"tag_name": "windows-vnext", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/next", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.2.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/w020", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.1.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/w010", "draft": false, "prerelease": false}
	]`)
	r, ok := LatestRelease(list, "windows-v")
	if !ok || r.Version != "0.2.0" || r.URL != "https://github.com/BoxPistols/hachibu/releases/tag/w020" {
		t.Fatalf("%+v %v", r, ok)
	}
}

func TestOpensOnlyGitHubPages(t *testing.T) {
	// ページはShellExecuteで開くので、GitHubのhttpsのページでなければ使わない
	for _, page := range []string{"file:///C:/Windows/System32/calc.exe", `C:\evil.exe`, "http://github.com/x", "https://github.com.example.com/x", ""} {
		list := []any{map[string]any{"tag_name": "windows-v9.0.0", "html_url": page, "draft": false, "prerelease": false}}
		if r, ok := LatestRelease(list, "windows-v"); ok {
			t.Errorf("%q accepted: %+v", page, r)
		}
	}
}

func TestNoWindowsReleaseYetIsNotAnError(t *testing.T) {
	list := releases(t, `[{"tag_name": "v0.6.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/v0.6.0", "draft": false, "prerelease": false}]`)
	if _, ok := LatestRelease(list, "windows-v"); ok {
		t.Fatal("should not find one")
	}
	if _, ok := LatestRelease(nil, "windows-v"); ok {
		t.Fatal("empty list")
	}
}

func TestComparesVersionsNumerically(t *testing.T) {
	newer := [][2]string{{"0.4.0", "0.3.2"}, {"0.10.0", "0.9.9"}, {"1.0", "0.99.1"}, {"0.3.2.1", "0.3.2"}}
	for _, c := range newer {
		if !IsNewer(c[0], c[1]) {
			t.Errorf("%v", c)
		}
	}
	// 開発中のビルド（"dev"）や読めないバージョンには知らせない
	notNewer := [][2]string{{"0.3.2", "0.3.2"}, {"0.3.2", "0.3.2.0"}, {"0.3.1", "0.3.2"}, {"0.4.0", "dev"}, {"0.4.0-beta", "0.3.2"}, {"+1.0", "0.3.2"}}
	for _, c := range notNewer {
		if IsNewer(c[0], c[1]) {
			t.Errorf("%v", c)
		}
	}
}

func TestNotifiesEachVersionOnce(t *testing.T) {
	r := Release{Version: "0.4.0", URL: "https://example.com"}
	if !ShouldNotify(r, "0.3.2", "") || !ShouldNotify(r, "0.3.2", "0.3.3") {
		t.Error("should notify")
	}
	if ShouldNotify(r, "0.3.2", "0.4.0") || ShouldNotify(r, "0.4.0", "") {
		t.Error("should not notify")
	}
}

// 0.1.0のリリースの本文の末尾（リリースの手順が足すSHA-256の欄）
const body010 = "Hachibu for Windows 0.1.0\r\n\r\nSHA-256\r\n```\r\n" +
	"923870199315d46c8c2314d31584489b00943c017bd3ffe451db42327955c9da  Hachibu-windows-x64.zip\r\n" +
	"ecd6a76424062c1b7b940c4fdcb6d1a0940364bb8ac9f75e7c51d64038f3bbbc  statusline.sh\r\n" +
	"f1915e99a5cc21f5814bc6140f695abe97c5ee1f2d06b09b3ae6e1c50cd8abbb  statusline.ps1\r\n```"

func TestReadsTheZipAndItsSHA256(t *testing.T) {
	list := []any{map[string]any{
		"tag_name": "windows-v0.1.0", "html_url": "https://github.com/BoxPistols/hachibu/releases/tag/windows-v0.1.0",
		"draft": false, "prerelease": false, "body": body010,
		"assets": []any{
			map[string]any{"name": "statusline.sh", "browser_download_url": "https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/statusline.sh"},
			map[string]any{"name": "Hachibu-windows-x64.zip", "browser_download_url": "https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip"},
		},
	}}
	r, ok := LatestRelease(list, "windows-v")
	if !ok || !r.CanInstallInPlace() {
		t.Fatalf("%+v %v", r, ok)
	}
	if r.AssetURL != "https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip" ||
		r.SHA256 != "923870199315d46c8c2314d31584489b00943c017bd3ffe451db42327955c9da" {
		t.Fatalf("%+v", r)
	}
}

func TestDoesNotInstallWithoutTheZipOrItsSHA256(t *testing.T) {
	page := "https://github.com/BoxPistols/hachibu/releases/tag/windows-v0.2.0"
	zip := map[string]any{"name": "Hachibu-windows-x64.zip", "browser_download_url": "https://github.com/BoxPistols/hachibu/releases/download/windows-v0.2.0/Hachibu-windows-x64.zip"}
	cases := map[string]map[string]any{
		// 添付が無い（ページを開く形に戻る）
		"no asset": {"body": body010},
		// 本文にSHA-256が無い
		"no sha": {"assets": []any{zip}, "body": "notes"},
		// このリポジトリのリリース以外の場所
		"elsewhere":  {"body": body010, "assets": []any{map[string]any{"name": "Hachibu-windows-x64.zip", "browser_download_url": "https://example.com/Hachibu-windows-x64.zip"}}},
		"http":       {"body": body010, "assets": []any{map[string]any{"name": "Hachibu-windows-x64.zip", "browser_download_url": "http://github.com/BoxPistols/hachibu/releases/download/windows-v0.2.0/Hachibu-windows-x64.zip"}}},
		"other repo": {"body": body010, "assets": []any{map[string]any{"name": "Hachibu-windows-x64.zip", "browser_download_url": "https://github.com/someone/hachibu/releases/download/windows-v0.2.0/Hachibu-windows-x64.zip"}}},
	}
	for name, extra := range cases {
		rel := map[string]any{"tag_name": "windows-v0.2.0", "html_url": page, "draft": false, "prerelease": false}
		for k, v := range extra {
			rel[k] = v
		}
		r, ok := LatestRelease([]any{rel}, "windows-v")
		if !ok || r.URL != page {
			t.Errorf("%s: the release itself should still be found: %+v", name, r)
		}
		if r.CanInstallInPlace() {
			t.Errorf("%s: should not install in place: %+v", name, r)
		}
	}
}

func TestSHA256InBody(t *testing.T) {
	sum := "923870199315d46c8c2314d31584489b00943c017bd3ffe451db42327955c9da"
	for body, want := range map[string]string{
		body010: sum,
		// sha256sum -b の書き方と、大文字の16進
		strings.ToUpper(sum) + " *Hachibu-windows-x64.zip": sum,
		// 名前が違う、桁が足りない、16進でない
		sum + "  Hachibu-windows-arm64.zip":         "",
		sum[:63] + "  Hachibu-windows-x64.zip":      "",
		"g" + sum[1:] + "  Hachibu-windows-x64.zip": "",
		"": "",
	} {
		if got := SHA256InBody(body, WindowsAssetName); got != want {
			t.Errorf("%q → %q", body, got)
		}
	}
}
