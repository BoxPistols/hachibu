#!/bin/bash
# Claude CodeのstatusLineから呼ぶ。受け取ったJSONをHachibuが読む場所へ書く。
# 引数にコマンドを渡すと、同じJSONをそのコマンドにも渡し、その出力をステータス行に使う（既存のstatusLineと併用するため）。
#
# ~/.claude/settings.jsonの例:
#   "statusLine": {"type": "command", "command": "/path/to/hachibu/scripts/statusline.sh"}
#   "statusLine": {"type": "command", "command": "/path/to/hachibu/scripts/statusline.sh ~/.claude/my-statusline.sh"}
set -u

DIR="$HOME/Library/Application Support/Hachibu"
/bin/mkdir -p "$DIR"
TMP="$(/usr/bin/mktemp "$DIR/.statusline.XXXXXX")"
/bin/cat > "$TMP"

# 次のコマンドには自分が受け取ったJSONを渡す。共有ファイルから読むと、同時に動く別セッションの内容を渡しうる
status=0
if [ $# -gt 0 ]; then
  "$@" < "$TMP" || status=$?
fi
# 同じディレクトリ内の置き換えなので、読み手が書きかけのファイルを見ることはない
/bin/mv -f "$TMP" "$DIR/statusline.json"
exit $status
