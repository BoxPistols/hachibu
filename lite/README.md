# Hachibu Lite

Claude Code の 5 時間枠と週枠の使用率を、macOS のメニューバーと画面上の小さな帯に出しておくアプリです。[Hachibu](../README.md) の軽量版で、Swift のファイル 1 つと AppKit だけで書いています。macOS 13 で止まっている Intel の MacBook で動かすために作りました。旧名は TokenBar です。

A lightweight edition of Hachibu: one Swift file, AppKit only, for Intel Macs on macOS 13 or later. It shows Claude Code's 5-hour and weekly usage in the menu bar and in a small strip on screen. Its menus are in Japanese. See [Hachibu or Hachibu Lite](../README.md#hachibu-or-hachibu-lite) for how the two differ.

Hachibu Lite は個人が作った非公式のツールで、Anthropic, PBC とは関係がなく、同社の承認や支援も受けていません。

Hachibu とどちらを使うかは、[Hachibu と Hachibu Lite](../README.md#hachibu-と-hachibu-lite) を見てください。両方を同時に動かすと、⌥⌘/ が先に起動した方にしか効きません。どちらか一方を使ってください。

## 表示の読み方

`Fable5.1• xhigh · S2 W39` は、モデル、effort、使用率の順です。モデル名の後ろの点は 1M コンテキストを表します。S は 5 時間枠、W は週枠です。

- 使用率が 80% 以上なら黄、90% 以上なら赤の札で数字を囲みます。閾値はメニューから変えられます（黄は 50〜80%、赤は 80〜95%）
- メニューバーの項目を開くと、各枠がいつリセットされるか、モデル、effort、最終更新の時刻が分かります
- 帯は押しても前面に出ません。フルスクリーンのアプリを含め、ほかの窓の上に出ます。ドラッグで動かせ、位置は覚えます。「帯を初期位置に戻す」でメニューバーのすぐ下の中央に戻ります
- 15 秒ごとにファイルを読み直します。「今すぐ更新」（⌘R）は、読み直しと新しい版の確認をその場で行い、結果をメニューバーに少しの間出します

### 帯の表示モード

| モード | 出るもの |
|---|---|
| 標準 | モデル、effort、使用率 |
| 使用率のみ | `S2 W39` のような使用率だけ |
| しまう | 小さな丸い取っ手だけ。輪の色で、いちばん高い使用率の段階を示す |
| 非表示 | 何も出さない |

マウスを乗せると一時的に標準へ広がります。帯の右クリックメニューとメニューバーの項目から、モード、不透明度（100〜30%）、閾値を切り替えられます。

ショートカットは 2 つで、固定です。⌥⌘/ で帯の表示と非表示を切り替え、⌥⌘. で標準、使用率のみ、しまうの順に切り替えます。Carbon のホットキーなので、アクセシビリティの許可は要りません。

### メニューバーの表示

アイコンのみ、週枠ゲージ（丸いメーターに週枠の使用率を数字 1 つで添える）、使用率、すべて（帯の標準と同じ文字列）から選べます。

## 値の出どころ

Claude Code の statusLine（公式の拡張点）が渡す JSON だけを読みます。キーチェーンを読まず、使用率の API も呼びません。そのため、Hachibu で使用率 API を有効にしたときに出るモデル別の週枠（F）は出ません。

statusLine には Hachibu と同じ `statusline.sh` を使います。JSON は `~/Library/Application Support/Hachibu/statusline.json` に保存され、Hachibu と Hachibu Lite のどちらもそこを読みます。TokenBar の `statusline.sh` を使い続けている場合のために、`~/Library/Application Support/TokenBar/statusline.json` も読み、新しい方を使います。使用率（`rate_limits`）はセッションの最初の応答の後にしか入らないので、無いときは前回の値を残します。リセット時刻を過ぎた値は 0% として扱います。

ネットワークに接続するのは、新しい版の確認だけです。起動時と、その後 1 日に 1 回、このリポジトリのリリースの一覧（`api.github.com`）から `lite-v` で始まるタグを探します。送るのは User-Agent（`HachibuLite/版`）だけです。新しい版があれば、メニューの先頭に「Hachibu Lite x.y.z が出ています」と出し、押すとリリースのページを開きます。

## インストール

### ビルド済みのアプリを使う

[リリースの一覧](https://github.com/BoxPistols/hachibu/releases)から、タグが `lite-v` で始まる最新のものを開き、`HachibuLite-macos-x86_64.zip` を落として展開します。`HachibuLite.app` を `~/Applications` か `/Applications` に置きます（macOS 13 以降、Intel のみ）。Apple の公証を受けていないため、初回の起動は macOS に止められます。「システム設定」＞「プライバシーとセキュリティ」で一度だけ許可してください。

次に、同じリリースにある `statusline.sh` を置きます。

```sh
curl -L -o ~/.claude/hachibu-statusline.sh https://github.com/BoxPistols/hachibu/releases/latest/download/statusline.sh
chmod +x ~/.claude/hachibu-statusline.sh
```

`statusline.sh` は Hachibu のリリースと同じものなので、Hachibu の最新リリースから落としてかまいません。`~/.claude/settings.json` の statusLine からこれを呼びます。

```json
{
  "statusLine": { "type": "command", "command": "~/.claude/hachibu-statusline.sh" }
}
```

すでに別の statusLine のコマンドを使っているときは、そのコマンドを引数に渡すと併用できます（`~/.claude/hachibu-statusline.sh ~/.claude/my-statusline.sh`）。

### ソースからビルドする

Xcode のコマンドラインツールが必要です。

```sh
git clone https://github.com/BoxPistols/hachibu.git
cd hachibu/lite
./build.sh
cp -R HachibuLite.app ~/Applications/
open ~/Applications/HachibuLite.app
```

`build.sh` は Intel（`x86_64-apple-macos13`）向けにビルドします。Apple Silicon では Hachibu を使ってください。署名はアドホックです。statusLine には `scripts/statusline.sh` を使います。

## TokenBar から移るとき

初めて起動したときに、TokenBar の設定（帯の表示モード、不透明度、閾値、メニューバーの表示と位置、帯の位置）を一度だけ引き継ぎます。TokenBar で「ログイン時に起動」を有効にしていた場合は、TokenBar の登録（`~/Library/LaunchAgents/local.tokenbar.plist`）を消して、Hachibu Lite を登録します。

引き継いだ後は TokenBar を終了し、`TokenBar.app` を消してかまいません。statusLine は TokenBar のスクリプトのままでも動きますが、上の手順で Hachibu の `statusline.sh` に替えておくと、Hachibu に移るときにも設定を変えずに済みます。

メニューの「ログイン時に起動」を有効にすると、`~/Library/LaunchAgents/io.github.boxpistols.hachibu.lite.plist` を置いて、Mac を再起動しても戻ります。

## リリース

`build.sh` の `version` を上げ、`docs/releases/lite-v<版>.md` に本文を書いて main に入れ、Actions の Release Lite を手で起動します。タグ `lite-v<版>` とリリースを作り、`HachibuLite-macos-x86_64.zip` と `scripts/statusline.sh` を添付します。Hachibu の新しい版の確認と README のダウンロードの案内（`releases/latest`）が Hachibu を指し続けるよう、Lite のリリースは「最新」にしません。

## ライセンス

MIT（リポジトリの [LICENSE](../LICENSE)）
