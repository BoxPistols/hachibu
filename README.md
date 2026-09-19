<img src="docs/images/app-icon.png" width="96" alt="">

# Slashstrip

A small strip that stays on top of your Mac's screen and shows Claude Code's model, effort, and usage limits at a glance.

![Slashstrip](docs/images/strip-basic.png)

Slashstrip is an unofficial tool made by an individual. It is not affiliated with, endorsed by, or sponsored by Anthropic, PBC.

[Download Slashstrip for macOS](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip) (macOS 14 or later). macOS blocks the first launch because the app is not notarized yet. [Install](#install) explains how to open it.

[日本語の説明はこちら](#日本語)

## What it shows

`Opus5 1M xhigh · S42 W76 F55` reads as: model, effort, then usage. S is the 5-hour limit, W the weekly limit, and F a per-model weekly limit (the first letter of the model's name).

- A usage number turns yellow at 70% and red at 90% by default. You can change both thresholds
- Hover over the strip or open the menu bar item to see when each limit resets
- The strip never takes focus. It stays above other windows, including full-screen apps
- It keeps running whether or not a terminal is open. Turn on Launch at Login from the menu bar to bring it back after a restart

## Where the numbers come from

Slashstrip works on its own. No other tools are needed.

- Model and effort: the conversation Claude Code saved most recently on this Mac, and `~/.claude/settings.json`. If you set up the statusLine below, its values are used when they are newer
- Usage: you choose one of two sources

1. Usage API (optional). The first time Slashstrip opens, it asks whether to use it. If you turn it on, Slashstrip uses the login that Claude Code saved in your keychain to ask Anthropic's usage endpoint every 5 minutes. S, W, and F then stay up to date even when you are not using the terminal. This endpoint is not publicly documented, and we found nothing showing that Anthropic allows third-party apps to use it. It may stop working, or Anthropic may consider it against its terms; any consequence would fall on your account. The login is used only for this request and is not saved or logged
2. statusLine (official). Claude Code's statusLine passes the 5-hour and weekly usage to a command you choose. It updates only while you use Claude Code in the terminal, and it does not include per-model limits. See [statusLine setup](#statusline-setup)

You can switch the usage API on or off at any time from the menu bar item.

## Stays out of the way

Pick how much of the strip you want to see. Hover to expand it to Standard temporarily.

| Mode | Shows |
|---|---|
| Standard | Model, effort, and usage |
| Usage only | Usage numbers such as `S93 W76 F55` |
| Tucked | A small handle at the edge of the screen. Its ring shows the highest usage level |
| Hidden | Nothing. Show it with the shortcut or from the menu bar |

![Usage only](docs/images/strip-usage-critical.png)

Right-click the strip to switch modes, opacity, and color thresholds. Hovering over an item previews it on the strip before you choose.

![Right-click menu](docs/images/menu-context.png)

You can also show usage in the menu bar, and summon the strip with a keyboard shortcut (⌥⌘/ by default). When you record a new shortcut, Slashstrip rejects combinations that macOS already uses, and saves it only after you press it once more and it arrives.

The interface follows your macOS language: Japanese if it comes first in your preferred languages, English otherwise.

## Install

1. Download [Slashstrip-macos.zip](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip). It always points to the latest release. Slashstrip needs macOS 14 or later
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
- Read [Data](#data) for what the app reads and sends

### Permissions it may ask for

- Keychain. If you turn on the usage API, macOS may ask whether to allow access to the "Claude Code-credentials" item
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

## statusLine setup

This is optional. Use it if you leave the usage API off, or if you want the model and effort of the session you are working in.

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

## Using only the Claude Code desktop app

- With the usage API turned on, usage keeps updating whether you use the terminal or the desktop app
- With the usage API off, usage comes only from the terminal's statusLine, so it stops updating while you use only the desktop app. The strip keeps showing the last known values, dims them after 30 minutes, and tells you when they were last updated. See [#6](https://github.com/BoxPistols/slashstrip/issues/6)

## Data

- Slashstrip reads local files: Claude Code's saved conversations (only to find the most recently used model), `~/.claude/settings.json`, and the statusLine file above
- It connects to the network only if you turn on the usage API, and then only to `api.anthropic.com`
- It reads the Claude Code login from the keychain only for that request, and does not save or log it
- Settings changes and failed usage requests are logged to `~/Library/Logs/Slashstrip/actions.log`

## Build from source

You need macOS 14 or later and the Xcode Command Line Tools (the full Xcode app is not needed).

```sh
scripts/build-app.sh            # builds build/Slashstrip.app
scripts/build-app.sh --install  # copies it to ~/Applications and launches it
scripts/build-app.sh --zip      # also writes build/Slashstrip-macos.zip and its SHA-256
scripts/test.sh                 # runs the tests
```

## Contributing

Issues and pull requests are welcome. Please run `scripts/test.sh` before sending a pull request. UI text lives in `Sources/SlashstripCore/Strings.swift`; when you change it, update both English and Japanese.

## Trademarks

Claude and Claude Code are trademarks of Anthropic, PBC. Touch Bar is a trademark of Apple Inc., registered in the U.S. and other countries and regions.

## License

MIT

---

## 日本語

Macの画面の最前面に小さな帯を常駐させ、Claude Codeのモデル、effort、使用率をひと目で確かめられるようにするアプリです。

Slashstripは個人が作った非公式のツールで、Anthropic, PBCとは関係がなく、同社の承認や支援も受けていません。

![Slashstripの帯](docs/images/ja-strip-basic.png)

[macOS版をダウンロード](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip)（macOS 14以降）。まだAppleの公証を受けていないため、初回の起動はmacOSに止められます。開き方は[インストール](#インストール)にあります。

### 表示の読み方

`Opus5 1M xhigh · S42 W76 F55`は、モデル、effort、使用率の順です。Sは5時間枠、Wは週枠、Fはモデル別の週枠（モデル名の頭文字）です。

- 使用率が既定で70%以上なら黄、90%以上なら赤の札で数字を囲みます。閾値はどちらも変えられます
- 帯にマウスを乗せるか、メニューバーの項目を開くと、各枠がいつリセットされるかが分かります
- 帯を押してもSlashstripは前面に出ません。フルスクリーンのアプリを含め、ほかの窓の上に出ます
- ターミナルを開いているかどうかに関係なく動き続けます。メニューバーの項目で「ログイン時に起動」を有効にすると、Macを再起動しても戻ります

### 値の出どころ

Slashstripだけで動きます。ほかの道具は要りません。

- モデルとeffort: このMacでClaude Codeが最後に保存した会話と、`~/.claude/settings.json`から読みます。下のstatusLineを設定していれば、そちらが新しいときはそちらを使います
- 使用率: 次の2つから選べます

1. 使用率API（任意）。初めて起動したときに、使うかどうかを尋ねます。有効にすると、Claude Codeがキーチェーンに保存したログイン情報を使い、5分ごとにAnthropicの使用率のエンドポイントへ問い合わせます。S、W、Fが、ターミナルを使っていないときも更新されます。このエンドポイントは公開されていないもので、Anthropicがサードパーティのアプリからの利用を認めているという根拠は見つかっていません。使えなくなることや、規約に反すると判断されることがありえ、その場合の影響はあなたのアカウントに及びます。ログイン情報はこの問い合わせにだけ使い、保存も記録もしません
2. statusLine（公式）。Claude CodeのstatusLineが、5時間枠と週枠の使用率を指定したコマンドに渡します。更新されるのはターミナルでClaude Codeを使っている間だけで、モデル別の枠は含まれません。[statusLineの設定](#statuslineの設定)を参照してください

使用率APIは、メニューバーの項目からいつでも有効・無効を切り替えられます。

### 表示を控えめにする

表示モードは4つです。マウスを乗せると一時的に標準へ広がります。

| モード | 出るもの |
|---|---|
| 標準 | モデル、effort、使用率 |
| 使用率だけ | `S93 W76 F55`のような使用率だけ |
| 端に収納 | 画面の端の小さなつまみだけ。輪の色で、いちばん高い使用率の段階を示す |
| 隠す | 何も出さない。ショートカットかメニューバーから出す |

帯を右クリックすると、表示モード、不透明度、色の閾値を切り替えられます。項目にマウスを乗せると、選ぶ前に帯へ仮に反映します。

![右クリックメニュー](docs/images/ja-menu-context.png)

メニューバーに使用率を出したり、ショートカット（既定は⌥⌘/）で帯を呼び出したりもできます。ショートカットを登録するときは、macOSが使っている組み合わせを弾き、もう一度押して届いたときだけ保存します。

画面の文言は、macOSの言語設定に合わせて日本語か英語になります。

### インストール

1. [Slashstrip-macos.zip](https://github.com/BoxPistols/slashstrip/releases/latest/download/Slashstrip-macos.zip)をダウンロードします。このリンクは常に最新のリリースを指します。macOS 14以降が必要です
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
- アプリが何を読み、何を送るかは[データの扱い](#データの扱い)にあります

#### 求められることがある権限

- キーチェーン。使用率APIを有効にすると、「Claude Code-credentials」の項目へのアクセスを許可するかをmacOSが尋ねることがあります
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

### statusLineの設定

任意です。使用率APIを使わない場合や、作業中のセッションのモデルとeffortを出したい場合に設定します。

`statusline.sh`は、Claude CodeがstatusLineに渡すJSONを`~/Library/Application Support/Slashstrip/statusline.json`に保存し、Slashstripはそれを読みます。リリースから入手して（クローンした場合は`scripts/statusline.sh`）、実行できるようにします。

```sh
curl -L -o ~/.claude/slashstrip-statusline.sh https://github.com/BoxPistols/slashstrip/releases/latest/download/statusline.sh
chmod +x ~/.claude/slashstrip-statusline.sh
```

そのうえで、`~/.claude/settings.json`のstatusLineから呼びます。すでに別のstatusLineを使っている場合は、そのコマンドを引数に渡してください。同じJSONがそのコマンドにも渡り、ステータス行の表示はそのコマンドの出力になります。設定の例は英語の節にあります。

使用率（rate_limits）がstatusLineに入るのはPro/Maxのプランで、そのセッションの最初の応答の後からです。

### デスクトップアプリだけで使う場合

- 使用率APIを有効にしていれば、ターミナルとデスクトップアプリのどちらを使っていても使用率は更新されます
- 使用率APIを使わない場合、使用率はターミナルのstatusLineからしか届かないため、デスクトップアプリだけを使っている間は更新されません。帯は最後に分かっている値を出し続け、30分を過ぎると薄く表示して、何時点の値かを添えます。[#6](https://github.com/BoxPistols/slashstrip/issues/6)を参照してください

### データの扱い

- 読むのはローカルのファイルです。Claude Codeが保存した会話（最後に使われたモデルを知るためだけ）、`~/.claude/settings.json`、上のstatusLineのファイル
- ネットワークに接続するのは使用率APIを有効にした場合だけで、接続先は`api.anthropic.com`だけです
- キーチェーンのログイン情報は、その問い合わせのときだけ読み、保存も記録もしません
- 設定の変更と、使用率の取得に失敗したことは`~/Library/Logs/Slashstrip/actions.log`に残ります

### ソースからビルド

macOS 14以降と、Xcode Command Line Tools（Xcode本体は不要です）が必要です。

```sh
scripts/build-app.sh            # build/Slashstrip.appを作る
scripts/build-app.sh --install  # ~/Applicationsに置いて起動する
scripts/build-app.sh --zip      # build/Slashstrip-macos.zipとSHA-256も作る
scripts/test.sh                 # テスト
```

### 開発への参加

issueとプルリクエストを歓迎します。プルリクエストを送る前に`scripts/test.sh`を実行してください。画面の文言は`Sources/SlashstripCore/Strings.swift`にあります。変えるときは英語と日本語の両方を直してください。

### 商標

Claude、Claude CodeはAnthropic, PBCの商標です。Touch Barは、米国およびその他の国や地域で登録されたApple Inc.の商標です。

### ライセンス

MIT
