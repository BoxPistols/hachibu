//go:build windows

package app

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// 操作ログ。%LOCALAPPDATA%\Hachibu\actions.logに1行ずつ追記する。256KBを超えたら後半だけ残す。
// 設定の変更や使用率の取得の失敗など、画面に出ない出来事の理由を残す。
// 書き込みは1本のゴルーチンで順に行い、呼んだ側（UIのスレッド）を待たせない
var (
	logLines   = make(chan string, 64)
	logPending sync.WaitGroup
	logOnce    sync.Once
)

const logMaxBytes = 256 * 1024

func actionLogPath() string { return filepath.Join(dataDir(), "actions.log") }

func logAction(message string) {
	logOnce.Do(func() {
		go func() {
			for line := range logLines {
				writeLog(line)
				logPending.Done()
			}
		}()
	})
	logPending.Add(1)
	logLines <- time.Now().Format("2006-01-02 15:04:05") + " " + message + "\r\n"
}

func writeLog(line string) {
	path := actionLogPath()
	if info, err := os.Stat(path); err == nil && info.Size() > logMaxBytes {
		if data, err := os.ReadFile(path); err == nil {
			os.WriteFile(path, data[len(data)-logMaxBytes/2:], 0o644)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	f.WriteString(line)
}

// flushLog は先に頼んだ書き込みが済むまで、最長でtimeoutだけ待つ。済んだらtrue。
// ディスクが応答しなくても、呼んだ側（終了の処理など）を止めないため
func flushLog(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		logPending.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
