# Slashstrip

A small floating strip for Macs without a Touch Bar. It shows Claude Code's model, effort, and usage limits at a glance, and puts slash commands one click away.

![Slashstrip demo](docs/images/demo.gif)

Slashstrip is an unofficial tool made by an individual. It is not affiliated with, endorsed by, or sponsored by Anthropic, PBC.

[Download Slashstrip for macOS](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip) (macOS 14 or later). macOS blocks the first launch because the app is not notarized yet. [Install](#install) explains how to open it.

[日本語の説明はこちら](#日本語)

## What it does

![Full strip](docs/images/strip-full.png)

- Shows the model, effort, and usage (5-hour and weekly limits), for example `Opus5 1M xhigh · S42 W76`
- Marks a usage number yellow at 70% and red at 90% by default. You can change both thresholds
- Sends `/review`, `/model`, `/usage`, and other commands to the Claude Code session in the front iTerm2 tab with one click
- Shows answer buttons while Claude Code waits for a permission or asks a multiple-choice question
- Lists running sessions in start order when you click the status. Pick one to jump to its tab
- Never takes focus. Your terminal stays in front while you click the strip

![Waiting for permission](docs/images/strip-permission.png)

## Stays out of the way

Pick how much of the strip you want to see. Hover to expand it temporarily.

| Mode | Shows |
|---|---|
| Full | Status and every button |
| Compact | Status only, plus answer buttons when needed |
| Usage only | Usage numbers such as `S93 W76`, plus answer buttons when needed |
| Tucked | A small handle at the edge of the screen. Its color shows the state |
| Hidden | Nothing, until Claude Code needs an answer |

![Usage only](docs/images/strip-usage-critical.png)

Right-click the strip to switch modes, opacity, and color thresholds. Hovering over an item previews it on the strip before you choose.

![Context menu](docs/images/menu-context.png)

You can also show usage in the menu bar, and summon the strip with a keyboard shortcut (⌥⌘/ by default). When you record a new shortcut, Slashstrip rejects combinations that macOS already uses, then asks you to press it once more to confirm it actually arrives.

![Shortcut recorder](docs/images/recorder-verify.png)

Turn on Launch at Login from the menu bar so the strip comes back after a restart. It keeps running whether or not a terminal is open.

The interface follows your macOS language: Japanese if it comes first in your preferred languages, English otherwise.

## Current status

Slashstrip reads state from one of two sources.

- With the author's Touch Bar bridge scripts in `~/.claude/btt`, everything above works. The scripts write each session's state from Claude Code hooks, decide which buttons to show, and send input to iTerm2. They are not published yet
- Without them, Slashstrip shows the model, effort, and usage from Claude Code's statusLine. Command buttons, answer buttons, and the session list are not available in this mode yet

Building the hooks bridge into the app itself is tracked in [#1](https://github.com/BoxPistols/slashstrip/issues/1). Per-model weekly limits are not part of the statusLine data, so they appear only when the bridge scripts provide them.

## Using only the Claude Code desktop app

- The 5-hour and weekly usage come from the statusLine, which only the terminal version of Claude Code has. While you use only the desktop app, these numbers stop updating. The strip keeps showing the last known values, dims them after 30 minutes, and tells you when they were last updated
- With the bridge scripts, hooks from the desktop app are recorded too, so the state color follows desktop sessions. Command buttons appear only while iTerm2 or Terminal is in front. For a permission request in the desktop app, the answer button brings the app to the front
- Without the bridge scripts, nothing updates in the desktop app, because there is no statusLine there. There is currently no documented way to read usage limits outside the terminal. See [#6](https://github.com/BoxPistols/slashstrip/issues/6)

## Install

1. Download [Slashstrip-macos.zip](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip). It always points to the latest release. Slashstrip needs macOS 14 or later, and iTerm2 if you want to send commands
2. Unzip it (Safari may already have done this) and move `Slashstrip.app` to your Applications folder
3. Open Slashstrip. The first time, macOS stops it. Follow the next section once

### First launch: macOS blocks it once

Slashstrip is not distributed through the App Store, and it is not yet signed with an Apple Developer ID or notarized by Apple ([#3](https://github.com/BoxPistols/slashstrip/issues/3)). macOS cannot check where it came from, so it refuses to open it the first time. This is expected, and you only need to allow it once per download.

These are Apple's own steps from [Open a Mac app from an unknown developer](https://support.apple.com/guide/mac-help/open-a-mac-app-from-an-unknown-developer-mh40616/mac):

1. Try to open Slashstrip, then close the warning. Do not move the app to the Trash
2. Choose Apple menu > System Settings, then click Privacy & Security in the sidebar
3. Go to Security. Next to the message about Slashstrip, click Open (Open Anyway on some versions of macOS). The button appears for about an hour after you tried to open the app
4. Click Open Anyway, enter your login password, then click OK

From then on Slashstrip opens normally, including at login.

If you prefer Terminal, you can remove the "downloaded from the internet" flag that macOS put on this copy, then open it:

```sh
xattr -dr com.apple.quarantine /Applications/Slashstrip.app
open /Applications/Slashstrip.app
```

This skips the check only for this copy. Do either of these only if you trust where the file came from.

### Before you allow it

Allowing an app that Apple has not checked is a decision you make. Some ways to check this one first:

- The source is all in this repository. You can [build it yourself](#build-from-source), and macOS does not block an app you built on your own Mac
- Compare the checksum with the SHA-256 on the [release page](https://github.com/BoxPistols/slashstrip/releases/latest): `shasum -a 256 ~/Downloads/Slashstrip-macos.zip`
- Slashstrip does not connect to the network (see [Data](#data))

### Permissions it asks for

- Control iTerm2 (Automation). Asked the first time you press a command button, because Slashstrip types the command into your terminal session. You can change it in System Settings > Privacy & Security > Automation
- Login Items. When you turn on Launch at Login, macOS tells you that Slashstrip will open at login. You can manage it in System Settings > General > Login Items (Login Items & Extensions on newer versions of macOS)

After you update to a new version, macOS may ask again, because the app is not signed with a fixed Developer ID yet.

### Uninstall

1. If you turned on Launch at Login, turn it off from the menu bar item first
2. Quit Slashstrip from the menu bar item, and move `Slashstrip.app` to the Trash
3. To remove its settings and logs as well:

```sh
defaults delete dev.local.slashstrip
rm -rf ~/Library/Logs/Slashstrip ~/"Library/Application Support/Slashstrip"
```

4. If you set up `statusline.sh`, remove it from the statusLine in `~/.claude/settings.json`

## Build from source

You need macOS 14 or later and the Xcode Command Line Tools (the full Xcode app is not needed).

```sh
scripts/build-app.sh            # builds build/Slashstrip.app
scripts/build-app.sh --install  # copies it to ~/Applications and launches it
scripts/build-app.sh --zip      # also writes build/Slashstrip-macos.zip and its SHA-256
scripts/test.sh                 # runs the tests
```

Without a signing certificate the app is ad-hoc signed, so macOS asks again for permission to control iTerm2 after each rebuild.

## statusLine setup (without the bridge scripts)

`statusline.sh` saves the JSON that Claude Code passes to the statusLine to `~/Library/Application Support/Slashstrip/statusline.json`, which Slashstrip reads. Download it from the release (or use `scripts/statusline.sh` in a clone) and make it executable:

```sh
curl -L -o ~/.claude/slashstrip-statusline.sh https://github.com/BoxPistols/slashstrip/releases/latest/download/statusline.sh
chmod +x ~/.claude/slashstrip-statusline.sh
```

Then call it from the statusLine in `~/.claude/settings.json`:

```json
{
  "statusLine": { "type": "command", "command": "~/.claude/slashstrip-statusline.sh" }
}
```

If you already use a statusLine command, pass it as an argument. It receives the same JSON, and its output becomes your status line.

```json
{
  "statusLine": { "type": "command", "command": "~/.claude/slashstrip-statusline.sh ~/.claude/my-statusline.sh" }
}
```

Usage limits (`rate_limits`) appear in the statusLine data on Pro and Max plans, after the first response in a session.

## Data

- Slashstrip does not connect to the network. It only reads local files written by hooks or the statusLine
- It does not read Claude Code credentials (Keychain tokens)
- Each button press and its result are logged to `~/Library/Logs/Slashstrip/actions.log`
- Commands are typed into the front iTerm2 session through AppleScript

## Contributing

Issues and pull requests are welcome. Please run `scripts/test.sh` before sending a pull request. UI text lives in `Sources/SlashstripCore/Strings.swift`; when you change it, update both English and Japanese.

## Trademarks

Claude and Claude Code are trademarks of Anthropic, PBC. Touch Bar is a trademark of Apple Inc., registered in the U.S. and other countries and regions.

## License

MIT

---

## 日本語

Touch Barを搭載していないMacで、Claude Codeのモデル、effort、使用率を画面上の細い帯に常時表示し、スラッシュコマンドをワンクリックで送れるようにするアプリです。

Slashstripは個人が作った非公式のツールで、Anthropic, PBCとは関係がなく、同社の承認や支援も受けていません。

[macOS版をダウンロード](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip)（macOS 14以降）。まだAppleの公証を受けていないため、初回の起動はmacOSに止められます。開き方は[インストール](#インストール)にあります。

![Slashstripの帯](docs/images/ja-strip-full.png)

### できること

- モデル、effort、使用率（5時間枠、週枠）を出します。例: `Opus5 1M xhigh · S42 W76`
- 使用率が既定で70%以上なら黄、90%以上なら赤の札で数字を囲みます。閾値はどちらも変えられます
- `/review`、`/model`、`/usage`などのボタンを押すと、前面のiTerm2のタブで動いているClaude Codeのセッションに入力します
- 許可プロンプトや、選択肢のある質問が出ているときは、答えのボタンを出します
- 状態のボタンを押すと、動いているセッションの一覧を開始順の番号付きで出します。選んだセッションのタブへ移ります
- 帯を押してもSlashstripは前面に出ません。ターミナルを前面にしたまま操作できます

### 表示を控えめにする

表示モードは5つです。マウスを乗せると一時的にフルへ広がります。

| モード | 出るもの |
|---|---|
| フル | 状態とすべてのボタン |
| コンパクト | 状態の枠だけ。答えが要るときは応答ボタンも出す |
| 使用率だけ | `S93 W76`のような使用率だけ。答えが要るときは応答ボタンも出す |
| 端に収納 | 画面の端の小さなつまみだけ。状態は色で示す |
| 隠す | 何も出さない。Claude Codeが答えを待っているときだけ出す |

帯を右クリックすると、表示モード、不透明度、色の閾値を切り替えられます。項目にマウスを乗せると、選ぶ前に帯へ仮に反映します。

![右クリックメニュー](docs/images/ja-menu-context.png)

メニューバーに使用率を出したり、ショートカット（既定は⌥⌘/）で帯を呼び出したりもできます。ショートカットを登録するときは、macOSが使っている組み合わせを弾き、登録後にもう一度押してもらって、実際に届くことを確かめます。

メニューバーの項目で「ログイン時に起動」を有効にすると、Macを再起動しても帯が戻ります。ターミナルを開いているかどうかに関係なく動き続けます。

画面の文言は、macOSの言語設定に合わせて日本語か英語になります。

### 現時点の前提

Slashstripは、次のどちらかから状態を読みます。

- `~/.claude/btt`に作者のTouch Bar連携スクリプトがある環境では、上のすべてが動きます。このスクリプトは、Claude Codeのhooksから各セッションの状態を書き出し、帯に出すボタンを決め、iTerm2へ入力を送ります。まだ公開していません
- スクリプトが無い環境では、Claude CodeのstatusLineからモデル、effort、使用率を出します。コマンドのボタン、応答のボタン、セッションの一覧は、このモードではまだ使えません

hooksとの連携をアプリに組み込む作業は[#1](https://github.com/BoxPistols/slashstrip/issues/1)で進めます。モデル別の週枠はstatusLineに含まれないため、スクリプトが別に取得している環境でだけ表示されます。

### デスクトップアプリだけで使う場合

- 5時間枠と週枠の使用率は、ターミナル版のClaude Codeだけが持つstatusLineから受け取ります。デスクトップアプリだけを使っている間は、この値が更新されません。帯は最後に分かっている値を出し続け、30分を過ぎると薄く表示して、何時点の値かを添えます
- 連携スクリプトがある環境では、デスクトップアプリのhooksも記録されるので、状態の色はデスクトップアプリのセッションにも追従します。コマンドのボタンは、iTerm2かTerminalが前面のときだけ出ます。デスクトップアプリで許可を求められたときは、応答のボタンがデスクトップアプリを前面に出します
- 連携スクリプトが無い環境では、デスクトップアプリにstatusLineが無いため、何も更新されません。ターミナルの外で使用率を読む公開された手段は、いまのところありません。[#6](https://github.com/BoxPistols/slashstrip/issues/6)で扱います

### インストール

1. [Slashstrip-macos.zip](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip)をダウンロードします。このリンクは常に最新のリリースを指します。macOS 14以降が必要で、コマンドを送るにはiTerm2も必要です
2. 展開して（Safariでは自動で展開されることがあります）、`Slashstrip.app`をアプリケーションフォルダへ移します
3. Slashstripを開きます。初回はmacOSに止められるので、次の手順を一度だけ行ってください

#### 初回の起動：macOSに一度止められます

SlashstripはApp Storeでは配布しておらず、AppleのDeveloper IDでの署名とAppleによる公証もまだ受けていません（[#3](https://github.com/BoxPistols/slashstrip/issues/3)）。macOSは出どころを確かめられないため、初回は開きません。想定どおりの動きで、ダウンロードごとに一度許可すれば済みます。

Appleの手順（[開発元が不明なMacアプリを開く](https://support.apple.com/ja-jp/guide/mac-help/mh40616/mac)）は次のとおりです。

1. Slashstripを開こうとし、表示された警告を閉じます。アプリをゴミ箱には入れないでください
2. アップルメニュー ＞「システム設定」を選び、サイドバーで「プライバシーとセキュリティ」をクリックします
3. 「セキュリティ」に移動し、Slashstripについての表示の横にある「開く」（macOSの版によっては「このまま開く」）をクリックします。このボタンは、開こうとしてから約1時間だけ出ます
4. 「このまま開く」をクリックし、ログインパスワードを入力して「OK」をクリックします

これ以降は、ログイン時の起動も含めて普通に開きます。

ターミナルを使う場合は、macOSがこのコピーに付けた「インターネットからダウンロードした」印を外してから開く方法もあります。

```sh
xattr -dr com.apple.quarantine /Applications/Slashstrip.app
open /Applications/Slashstrip.app
```

これは、このコピーに限って確認を省きます。どちらの方法も、入手元を信頼できる場合にだけ行ってください。

#### 許可する前に

Appleが確認していないアプリを開くかどうかは、使う人の判断です。先に確かめる方法があります。

- ソースはすべてこのリポジトリにあります。[自分でビルド](#ソースからビルド)したアプリは、そのMacでは止められません
- [リリースのページ](https://github.com/BoxPistols/slashstrip/releases/latest)にあるSHA-256と照合できます: `shasum -a 256 ~/Downloads/Slashstrip-macos.zip`
- Slashstripはネットワークに接続しません（[データの扱い](#データの扱い)を参照）

#### 求められる権限

- iTerm2の操作（オートメーション）。コマンドのボタンを初めて押したときに尋ねられます。コマンドをターミナルのセッションへ入力するためです。「システム設定」＞「プライバシーとセキュリティ」＞「オートメーション」で変えられます
- ログイン項目。「ログイン時に起動」を有効にすると、ログイン時に開く旨をmacOSが知らせます。「システム設定」＞「一般」＞「ログイン項目」（新しいmacOSでは「ログイン項目と機能拡張」）で管理できます

まだ決まったDeveloper IDで署名していないため、新しい版に更新すると、もう一度尋ねられることがあります。

#### アンインストール

1. 「ログイン時に起動」を有効にしていた場合は、先にメニューバーの項目から無効にします
2. メニューバーの項目からSlashstripを終了し、`Slashstrip.app`をゴミ箱に入れます
3. 設定とログも消す場合は、次を実行します

```sh
defaults delete dev.local.slashstrip
rm -rf ~/Library/Logs/Slashstrip ~/"Library/Application Support/Slashstrip"
```

4. `statusline.sh`を設定していた場合は、`~/.claude/settings.json`のstatusLineから外します

### ソースからビルド

macOS 14以降と、Xcode Command Line Tools（Xcode本体は不要です）が必要です。

```sh
scripts/build-app.sh            # build/Slashstrip.appを作る
scripts/build-app.sh --install  # ~/Applicationsに置いて起動する
scripts/build-app.sh --zip      # build/Slashstrip-macos.zipとSHA-256も作る
scripts/test.sh                 # テスト
```

署名用の証明書が無い環境ではアドホック署名になるため、ビルドし直すたびに、macOSがiTerm2の操作を許可するかを尋ねます。

### statusLineの設定（スクリプトが無い環境）

`statusline.sh`は、Claude CodeがstatusLineに渡すJSONを`~/Library/Application Support/Slashstrip/statusline.json`に保存し、Slashstripはそれを読みます。リリースから入手して（クローンした場合は`scripts/statusline.sh`）、実行できるようにします。

```sh
curl -L -o ~/.claude/slashstrip-statusline.sh https://github.com/BoxPistols/slashstrip/releases/latest/download/statusline.sh
chmod +x ~/.claude/slashstrip-statusline.sh
```

そのうえで、`~/.claude/settings.json`のstatusLineから呼びます。すでに別のstatusLineを使っている場合は、そのコマンドを引数に渡してください。同じJSONがそのコマンドにも渡り、ステータス行の表示はそのコマンドの出力になります。設定の例は英語の節にあります。

使用率（rate_limits）がstatusLineに入るのはPro/Maxのプランで、そのセッションの最初の応答の後からです。

### データの扱い

- Slashstripはネットワークに接続しません。読むのは、hooksやstatusLineが書いたローカルのファイルだけです
- Claude Codeの資格情報（Keychainのトークン）は読みません
- ボタンを押した操作と、その結果は`~/Library/Logs/Slashstrip/actions.log`に残ります
- コマンドは、iTerm2のAppleScriptで前面のセッションに入力します

### 開発への参加

issueとプルリクエストを歓迎します。プルリクエストを送る前に`scripts/test.sh`を実行してください。画面の文言は`Sources/SlashstripCore/Strings.swift`にあります。変えるときは英語と日本語の両方を直してください。

### 商標

Claude、Claude CodeはAnthropic, PBCの商標です。Touch Barは、米国およびその他の国や地域で登録されたApple Inc.の商標です。

### ライセンス

MIT
