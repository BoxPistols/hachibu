//go:build windows

package app

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/windows"
)

// claudeDir はClaude Codeの設定と会話記録の場所（.claudeフォルダ）。
// WindowsでClaude Codeを使っていれば%USERPROFILE%\.claude、WSLで使っていればWSLのホームの.claude
type claudeDir struct {
	Path string
	// Label は操作ログに出す名前（"Windows"、"WSL: Ubuntu"）
	Label string
}

// findClaudeDirs は今読める.claudeフォルダを探す。止まっているWSLは起こさない（\\wsl.localhostに触れると起動するため）
func findClaudeDirs() []claudeDir {
	var dirs []claudeDir
	seen := map[string]bool{}
	add := func(path, label string) {
		key := strings.ToLower(filepath.Clean(path))
		if seen[key] {
			return
		}
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			seen[key] = true
			dirs = append(dirs, claudeDir{Path: path, Label: label})
		}
	}
	if custom := os.Getenv("CLAUDE_CONFIG_DIR"); custom != "" {
		add(custom, "Windows")
	}
	if home, err := windows.KnownFolderPath(windows.FOLDERID_Profile, 0); err == nil {
		add(filepath.Join(home, ".claude"), "Windows")
	}
	for _, distro := range runningWSLDistros() {
		root := wslRoot(distro)
		if root == "" {
			continue
		}
		label := "WSL: " + distro
		add(filepath.Join(root, "root", ".claude"), label)
		homes, _ := os.ReadDir(filepath.Join(root, "home"))
		for _, h := range homes {
			if h.IsDir() {
				add(filepath.Join(root, "home", h.Name(), ".claude"), label)
			}
		}
	}
	return dirs
}

// wslRoot はディストリビューションのファイルを開く道筋。Windows 11と新しいWindows 10は\\wsl.localhost、古いものは\\wsl$
func wslRoot(distro string) string {
	for _, prefix := range []string{`\\wsl.localhost\`, `\\wsl$\`} {
		root := prefix + distro + `\`
		if _, err := os.Stat(root); err == nil {
			return root
		}
	}
	return ""
}

// runningWSLDistros は動いているWSLのディストリビューションの名前。WSLが無いか、どれも動いていなければ空
func runningWSLDistros() []string {
	system, err := windows.GetSystemDirectory()
	if err != nil {
		return nil
	}
	cmd := exec.Command(filepath.Join(system, "wsl.exe"), "--list", "--running", "--quiet")
	// 黒い窓を出さない
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	done := make(chan struct{})
	var out []byte
	go func() {
		out, _ = cmd.Output()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return nil
	}
	var names []string
	for _, line := range strings.Split(decodeWSLOutput(out), "\n") {
		name := strings.TrimSpace(line)
		// Docker Desktopが作るディストリビューションにはClaude Codeが無い
		if name == "" || strings.HasPrefix(strings.ToLower(name), "docker-desktop") {
			continue
		}
		names = append(names, name)
	}
	return names
}

// decodeWSLOutput はwsl.exeの出力を読む。既定はUTF-16LEで、WSL_UTF8=1のときはUTF-8
func decodeWSLOutput(b []byte) string {
	if bytes.IndexByte(b, 0) < 0 {
		return string(b)
	}
	b = bytes.TrimPrefix(b, []byte{0xFF, 0xFE})
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[2*i]) | uint16(b[2*i+1])<<8
	}
	return string(utf16.Decode(u))
}

func labels(dirs []claudeDir) string {
	var out []string
	for _, d := range dirs {
		out = append(out, d.Label+" ("+d.Path+")")
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ", ")
}

// openShared はファイルを読むために開く。書き手（statusline.shの置き換えや、Claude Codeの会話記録の書き換え）を邪魔しないよう、
// 書き込みと削除を許す共有モードで開く（os.Openは削除を許さない）
func openShared(path string) (*os.File, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(h), path), nil
}

// readShared はファイルを先頭からlimitバイトまで読む
func readShared(path string, limit int64) ([]byte, error) {
	f, err := openShared(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var buf bytes.Buffer
	_, err = buf.ReadFrom(io.LimitReader(f, limit))
	return buf.Bytes(), err
}
