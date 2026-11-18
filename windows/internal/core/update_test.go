package core

import (
	"encoding/json"
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
	// 一覧は新しい順。macOS版（v）とLite（lite-v）のリリースも混ざる
	list := releases(t, `[
	  {"tag_name": "v0.6.0", "html_url": "https://example.com/v0.6.0", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.3.0", "html_url": "https://example.com/w030", "draft": true, "prerelease": false},
	  {"tag_name": "windows-v0.2.1", "html_url": "https://example.com/w021", "draft": false, "prerelease": true},
	  {"tag_name": "lite-v0.4.0", "html_url": "https://example.com/lite", "draft": false, "prerelease": false},
	  {"tag_name": "windows-vnext", "html_url": "https://example.com/next", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.2.0", "html_url": "https://example.com/w020", "draft": false, "prerelease": false},
	  {"tag_name": "windows-v0.1.0", "html_url": "https://example.com/w010", "draft": false, "prerelease": false}
	]`)
	r, ok := LatestRelease(list, "windows-v")
	if !ok || r.Version != "0.2.0" || r.URL != "https://example.com/w020" {
		t.Fatalf("%+v %v", r, ok)
	}
}

func TestNoWindowsReleaseYetIsNotAnError(t *testing.T) {
	list := releases(t, `[{"tag_name": "v0.6.0", "html_url": "https://example.com", "draft": false, "prerelease": false}]`)
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
	// 開発中のビルド（"dev"）や読めない版には知らせない
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
