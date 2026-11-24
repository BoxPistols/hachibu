#!/bin/bash
# Claude CodeのstatusLineから呼ぶ。受け取ったJSONをHachibuが読む場所へ書く。
# 引数にコマンドを渡すと、同じJSONをそのコマンドにも渡し、その出力をステータス行に使う（既存のstatusLineと併用するため）。
#
# ~/.claude/settings.jsonの例:
#   "statusLine": {"type": "command", "command": "/path/to/hachibu/scripts/statusline.sh"}
#   "statusLine": {"type": "command", "command": "/path/to/hachibu/scripts/statusline.sh ~/.claude/my-statusline.sh"}
#
# 書く場所:
#   macOS                          ~/Library/Application Support/Hachibu（HachibuとHachibu Liteが読む）
#   WSL、WindowsのGit Bash          Windowsの%LOCALAPPDATA%\Hachibu（Windows向けのHachibuが読む）
#   HACHIBU_DIRを設定したとき       その場所
set -u

hachibu_dir() {
  if [ -n "${HACHIBU_DIR:-}" ]; then
    echo "$HACHIBU_DIR"
    return
  fi
  case "$(uname -s)" in
    Darwin)
      echo "$HOME/Library/Application Support/Hachibu"
      ;;
    MINGW* | MSYS* | CYGWIN*)
      # WindowsでClaude CodeがGit Bashを通して呼ぶとき
      echo "$(cygpath -u "${LOCALAPPDATA:-$USERPROFILE\\AppData\\Local}")/Hachibu"
      ;;
    *)
      if grep -qi microsoft /proc/sys/kernel/osrelease 2>/dev/null; then
        wsl_windows_dir
      else
        # WSLでないLinuxには読むアプリが無いが、書く場所だけは決めておく
        echo "${XDG_STATE_HOME:-$HOME/.local/state}/hachibu"
      fi
      ;;
  esac
}

# WSLから、Windows側の%LOCALAPPDATA%\Hachibuを探す。cmd.exeの起動は遅いので、一度調べたら覚えておく
wsl_windows_dir() {
  local cache="${XDG_CACHE_HOME:-$HOME/.cache}/hachibu/windows-dir"
  local dir=""
  [ -s "$cache" ] && dir="$(cat "$cache")"
  if [ -z "$dir" ] || [ ! -d "$(dirname "$dir")" ]; then
    dir=""
    local cmd=cmd.exe
    command -v "$cmd" >/dev/null 2>&1 || cmd=/mnt/c/Windows/System32/cmd.exe
    # Linuxのフォルダから起動すると警告が出るので、Windowsのドライブへ移ってから呼ぶ。
    # 標準入力は渡さない（渡すと、Claude Codeが送ったJSONをcmd.exeが先に読んでしまう）。
    # /uでUTF-16にして受け取る。既定の出力は日本語のWindowsではShift_JISで、ユーザー名に日本語があると道筋が化ける
    local win
    win="$(cd /mnt/c 2>/dev/null; "$cmd" /d /u /c "echo %LOCALAPPDATA%" </dev/null 2>/dev/null | iconv -f UTF-16LE -t UTF-8 2>/dev/null | tr -d '\r\n')"
    if [ -n "$win" ] && [ "$win" != "%LOCALAPPDATA%" ]; then
      local local_dir
      local_dir="$(wslpath -u "$win" 2>/dev/null)"
      # 読めた場所が実在するときだけ使い、覚える
      if [ -n "$local_dir" ] && [ -d "$local_dir" ]; then
        dir="$local_dir/Hachibu"
        /bin/mkdir -p "$(dirname "$cache")" && echo "$dir" > "$cache"
      fi
    fi
    # Windowsのプログラムを呼べない設定（interopが無効）などで分からなければ、HACHIBU_DIRで場所を渡してもらう
    [ -n "$dir" ] || dir="${XDG_STATE_HOME:-$HOME/.local/state}/hachibu"
  fi
  echo "$dir"
}

DIR="$(hachibu_dir)"
/bin/mkdir -p "$DIR" 2>/dev/null
TMP="$(/usr/bin/mktemp "$DIR/.statusline.XXXXXX" 2>/dev/null)"
if [ -z "$TMP" ]; then
  # 書く場所が使えなくても、併用しているstatusLineのコマンドは今までどおり動かす
  [ $# -gt 0 ] && exec "$@"
  /bin/cat >/dev/null
  exit 0
fi
# Claude Codeは間に合わなかったstatusLineのコマンドを止める。止められても書きかけを残さない
trap '/bin/rm -f "$TMP"' EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM
# 強制終了では上の後始末も動かないので、20回に1回、5分より古い書きかけを消す（毎回だとWSLから/mnt/cを見る分だけ遅くなる）
if [ $((RANDOM % 20)) -eq 0 ]; then
  /usr/bin/find "$DIR" -maxdepth 1 -name '.statusline.*' -mmin +5 -delete 2>/dev/null
fi
/bin/cat > "$TMP"

# 次のコマンドには自分が受け取ったJSONを渡す。共有ファイルから読むと、同時に動く別セッションの内容を渡しうる
status=0
if [ $# -gt 0 ]; then
  "$@" < "$TMP" || status=$?
fi
# 同じディレクトリ内の置き換えなので、読み手が書きかけのファイルを見ることはない。置き換えられなければ書きかけを残さない
/bin/mv -f "$TMP" "$DIR/statusline.json" || /bin/rm -f "$TMP"
exit $status
