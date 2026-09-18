# Slashstrip

Touch Barを搭載していないMacで、Claude Codeの状態（モデル、effort、使用率）とスラッシュコマンドのボタンを、画面上の細い帯に常駐させるmacOSアプリです。

Slashstripは個人が作った非公式のツールで、Anthropic, PBCとは関係がなく、同社の承認や支援も受けていません。

## できること

- 帯に、モデル、effort、使用率（5時間枠、週枠）を出します。例: `Opus5 1M xhigh · S12 W76`
- 使用率が閾値を超えると、その数字を黄（既定は70%以上）や赤（既定は90%以上）の札で囲みます。閾値はメニューで変えられます
- `/review`、`/model`、`/usage`などのボタンを押すと、前面のiTerm2で動いているClaude Codeのセッションに入力します
- 許可プロンプトや、選択肢のある質問が出ているときは、答えのボタンを出します
- 状態のボタンを押すと、動いているセッションの一覧を開始順の番号付きで出します。選んだセッションのタブへ移ります
- 表示モードは、フル、コンパクト（状態の枠だけ）、使用率だけ、端に収納（小さなつまみだけ）、隠す（答えが要るときだけ出す）の5つです。マウスを乗せると一時的にフルへ広がります
- 帯を右クリックすると、表示モード、不透明度、色の閾値をその場で切り替えられます
- メニューバーの項目に、使用率や状態を文字で出せます
- ショートカット（既定は⌥⌘/）で帯を呼び出せます。組み合わせはメニューの「ショートカットを変更…」で変えられます

帯のボタンは押してもSlashstripを前面に出しません。ターミナルを前面にしたまま操作できます。

## 現時点の前提

コマンドの送信と許可への応答は、`~/.claude/btt`に作者のTouch Bar連携スクリプト一式がある環境でだけ動きます。このスクリプトは、Claude Codeのhooksから各セッションの状態を書き出し、帯に出す内容を決め、iTerm2へ送る役目を持っています。現在このリポジトリには含まれていません。

スクリプトが無い環境では、statusLineから受け取った情報で状態の枠だけを出します（コマンドは送りません）。スクリプトに頼らずに単体で完結させる作業は、issueで進めます。

モデル別の週枠（Fableの枠など）は、Claude CodeのstatusLineには含まれません。スクリプトが別に取得している環境でだけ表示されます。

## 必要なもの

- macOS 14以降
- Xcode Command Line Tools（Xcode本体は不要です）
- iTerm2（コマンドの送信を使う場合）

## ビルドと起動

```sh
scripts/build-app.sh            # build/Slashstrip.appを作る
scripts/build-app.sh --install  # ~/Applicationsに置いて起動する
scripts/test.sh                 # テスト
```

署名用の証明書が無い環境では、アドホック署名になります。この場合、ビルドし直すたびに、macOSが「iTerm2の操作」を許可するかを尋ねます。

## statusLineの設定（スクリプトが無い環境）

`~/.claude/settings.json`のstatusLineから`scripts/statusline.sh`を呼びます。受け取ったJSONを`~/Library/Application Support/Slashstrip/statusline.json`に保存し、Slashstripはそれを読みます。

```json
{
  "statusLine": { "type": "command", "command": "/path/to/slashstrip/scripts/statusline.sh" }
}
```

すでに別のstatusLineを使っている場合は、そのコマンドを引数に渡します。同じJSONがそのコマンドにも渡り、ステータス行の表示はそのコマンドの出力になります。

```json
{
  "statusLine": { "type": "command", "command": "/path/to/slashstrip/scripts/statusline.sh ~/.claude/my-statusline.sh" }
}
```

使用率（rate_limits）がstatusLineに入るのはPro/Maxのプランで、そのセッションの最初の応答の後からです。

## データの扱いと権限

- Slashstripはネットワークに接続しません。読むのは、hooksやstatusLineが書いたローカルのファイルだけです
- Claude Codeの資格情報（Keychainのトークン）は読みません
- ボタンを押した操作と、その結果は`~/Library/Logs/Slashstrip/actions.log`に残ります
- コマンドの送信は、iTerm2のAppleScriptで、押した時点で前面にあるセッションに文字を入力する方法で行います。初めて送るときに、macOSが許可を求めます

## 商標

Claude、Claude CodeはAnthropic, PBCの商標です。Touch Barは、米国およびその他の国や地域で登録されたApple Inc.の商標です。

## ライセンス

MIT
