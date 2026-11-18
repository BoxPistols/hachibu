//go:build windows

package app

import (
	"os"
	"path/filepath"

	"github.com/BoxPistols/hachibu/windows/internal/w32"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// MARK: - サインイン時の起動

// サインイン時の起動は、HKCU\...\Runにこのアプリの道筋を書く（管理者の権限は要らない）。
// 利用者が「設定」＞「アプリ」＞「スタートアップ」（またはタスク マネージャー）でオフにすると、
// StartupApproved\Runの値の先頭のバイトが奇数（03など）になり、Runに書いてあっても起動しない
const (
	runKey      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKey = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	runValue    = "Hachibu"
)

func windowsExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

func loginItemEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(runValue)
	return err == nil
}

func loginItemDisabledBySystem() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, approvedKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	data, _, err := k.GetBinaryValue(runValue)
	return err == nil && len(data) > 0 && data[0]&1 == 1
}

func setLoginItem(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(runValue); err != nil && err != registry.ErrNotExist {
			return err
		}
		return nil
	}
	exe, err := windowsExecutable()
	if err != nil {
		return err
	}
	return k.SetStringValue(runValue, `"`+exe+`"`)
}

// MARK: - 1つだけ動かす

var instanceMutex windows.Handle

// handOff は先に起動した同じアプリがあれば、そちらに帯を出すよう知らせてtrueを返す。
// 撮影用や試験用の別の保存先で動かすときは、重ねて起動してよい
func handOff() bool {
	if isDemo() || os.Getenv("HACHIBU_SUITE") != "" {
		return false
	}
	name, _ := windows.UTF16PtrFromString(`Local\io.github.boxpistols.hachibu.windows`)
	h, err := windows.CreateMutex(nil, false, name)
	if err == windows.ERROR_ALREADY_EXISTS {
		if h != 0 {
			windows.CloseHandle(h)
		}
		if hwnd := w32.FindWindow(mainClass); hwnd != 0 {
			w32.PostMessage(hwnd, msgSummon, 0, 0)
		}
		return true
	}
	instanceMutex = h
	return false
}

// MARK: - 開く

func openURL(url string) {
	verb, _ := windows.UTF16PtrFromString("open")
	target, _ := windows.UTF16PtrFromString(url)
	windows.ShellExecute(0, verb, target, nil, nil, windows.SW_SHOWNORMAL)
}

// preferredLanguages はWindowsの表示言語の優先順（"ja-JP"など）
func preferredLanguages() []string {
	langs, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err != nil {
		return nil
	}
	return langs
}
