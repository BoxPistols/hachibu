<img src="../docs/images/app-icon.png" width="96" alt="">

# Hachibu for Windows

Hachibu for Windows keeps Claude Code's model, effort, and usage limits in a small strip on top of your Windows screen. It is the Windows edition of [Hachibu](../README.md): the same strip, modes, colors, and menus, built for Windows with Go and the Win32 API. It is a single `Hachibu.exe` with nothing else to install.

It works with Claude Code on Windows and with Claude Code inside WSL.

Hachibu is an unofficial tool made by an individual. It is not affiliated with, endorsed by, or sponsored by Anthropic, PBC.

[Download Hachibu for Windows](https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip) (Windows 10 or 11, 64-bit). On Arm PCs it runs through Windows' x64 emulation. Windows SmartScreen stops the first launch because the app is not code-signed; [Install](#install) shows what to do.

[日本語の説明はこちら](#日本語)

## What it shows

`Opus5• xhigh · S42 W76 F93` reads as: model, effort, then usage. The small dot after the model name means a 1M context. S is the 5-hour limit, W the weekly limit, and F a per-model weekly limit (the first letter of the model's name). How to Read the Strip in the menu lists the same.

- A usage number turns yellow at 70% and red at 90% by default. You can change both thresholds
- Hover over the strip to see when each limit resets. The tray icon's menu and tooltip show it too
- The strip never takes the keyboard focus from the app you are using. It stays above other windows
- It keeps running whether or not a terminal is open. Turn on Launch at Sign-in from the menu to bring it back after a restart

## Where the numbers come from

- Model and effort: the conversation Claude Code saved most recently, and `settings.json` next to it. If you set up the statusLine below, its values are used when they are newer
- Usage: the usage API (optional) or the statusLine, as described in [Hachibu's README](../README.md#where-the-numbers-come-from). The usage API uses the login that Claude Code saved in `.credentials.json`; read the risks there before you turn it on

Hachibu looks for Claude Code's `.claude` folder in two places:

- Windows: `%USERPROFILE%\.claude`, or `CLAUDE_CONFIG_DIR` if you set it
- WSL: the home folders of each WSL distribution that is running, through `\\wsl.localhost`. Hachibu does not start WSL; it only reads distributions that are already running

The action log records which folders it reads.

## statusLine setup

This is optional. Use it if you leave the usage API off, or if you want the model and effort of the session you are working in. `statusline.sh` saves the JSON that Claude Code passes to the statusLine to `%LOCALAPPDATA%\Hachibu\statusline.json`, which Hachibu reads. Pick the steps for where you run Claude Code.

### Claude Code in WSL

Run this in WSL:

```sh
curl -L -o ~/.claude/hachibu-statusline.sh https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/statusline.sh
chmod +x ~/.claude/hachibu-statusline.sh
```

Then call it from the statusLine in `~/.claude/settings.json` in WSL:

```json
{
  "statusLine": { "type": "command", "command": "~/.claude/hachibu-statusline.sh" }
}
```

The script asks Windows for `%LOCALAPPDATA%` once (through `cmd.exe`) and remembers it in `~/.cache/hachibu/windows-dir`. If your WSL cannot run Windows programs, set `HACHIBU_DIR` to the folder instead, for example `HACHIBU_DIR=/mnt/c/Users/you/AppData/Local/Hachibu ~/.claude/hachibu-statusline.sh`.

### Claude Code on Windows

On Windows, Claude Code runs the statusLine through Git Bash when Git for Windows is installed, and through PowerShell otherwise. In PowerShell:

```powershell
# With Git Bash
Invoke-WebRequest https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/statusline.sh -OutFile "$HOME\.claude\hachibu-statusline.sh"
# Without Git Bash
Invoke-WebRequest https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/statusline.ps1 -OutFile "$HOME\.claude\hachibu-statusline.ps1"
```

Then call it from the statusLine in `%USERPROFILE%\.claude\settings.json`. Write paths with forward slashes; Git Bash drops backslashes.

```json
{
  "statusLine": { "type": "command", "command": "~/.claude/hachibu-statusline.sh" }
}
```

```json
{
  "statusLine": { "type": "command", "command": "powershell -NoProfile -ExecutionPolicy Bypass -File C:/Users/you/.claude/hachibu-statusline.ps1" }
}
```

If you already use a statusLine command, pass it as an argument to either script. It receives the same JSON, and its output becomes your status line.

Usage limits (`rate_limits`) appear in the statusLine data on Pro and Max plans, after the first response in a session.

## Stays out of the way

| Mode | Shows |
|---|---|
| Standard | Model, effort, and usage |
| Usage only | Usage numbers such as `S42 W76 F93` |
| Tucked | A small handle. Its ring shows the highest usage level |
| Hidden | Nothing. Show it with the shortcut or from the tray icon |

Right-click the strip to switch modes, opacity, and color thresholds. Hovering over an item previews it on the strip before you choose. Drag the dots at the left end to move the strip; Hachibu remembers which side of the screen you put it on.

The tray icon (the notification area at the right end of the taskbar) has the same menu. A tray icon cannot show text, so you can choose what it draws: the app icon, a weekly gauge (the slanted cut moves with the weekly usage, and the used side turns yellow and red), or the weekly percentage as a number. Its tooltip shows the full status.

Windows 11 puts new tray icons under Show hidden icons (^). To keep Hachibu on the taskbar, turn it on in Settings > Personalization > Taskbar > Other system tray icons. While the icon is hidden, the strip's right-click menu shows a shortcut to that setting, and everything in the tray icon's menu is also in the strip's menu.

Ctrl+Alt+/ shows the strip in any mode (press it again to go back). A second shortcut, which you set from Shortcut for Switching Display, steps through Standard, Usage only, and Tucked. When you record a shortcut, Hachibu rejects combinations that Windows uses (such as Alt+Tab or Ctrl+Shift+Esc) and combinations another app has registered. Press the new combination once more to confirm that it arrives, then click Save. Combinations with the Windows key are not recorded, because Windows uses most of them. To use the buttons from the keyboard, press Tab to move to them and Space to press one.

The interface follows your Windows display language: Japanese if it comes first, English otherwise. You can also pick English or Japanese from Language in the menu.

## Install

1. Download [Hachibu-windows-x64.zip](https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip) and extract it
2. Move `Hachibu.exe` to a folder where it can stay, such as `%LOCALAPPDATA%\Programs\Hachibu`. Launch at Sign-in remembers this location
3. Open `Hachibu.exe`. The first time, Windows SmartScreen shows "Windows protected your PC". Click More info, then Run anyway. Hachibu is not signed with a code-signing certificate yet, so SmartScreen does not know it
4. On its first run Hachibu asks whether to use the usage API

Before you allow it, you can check it: the source is all in this repository and you can [build it yourself](#build-from-source), and you can compare the checksum with the SHA-256 on the release page (`Get-FileHash Hachibu-windows-x64.zip`).

Hachibu runs one copy at a time. Opening `Hachibu.exe` again shows the strip.

### Updates

Hachibu checks GitHub for a new Windows version once a day. When it finds one, it downloads and checks the new version in the background, sends one Windows notification for that version, and shows Update to Hachibu x.y.z and Restart at the top of both menus. Click the notification or the menu item to replace `Hachibu.exe` and restart. Your settings stay. When Check Now finds a new version, Hachibu asks whether to update and restart.

Before replacing, Hachibu checks the zip's SHA-256 against the one in the release notes, and the product name and version inside `Hachibu.exe`. Windows cannot overwrite a running program, so Hachibu renames the old one to `Hachibu.exe.old`, puts the new one in its place, and deletes the old one when the new one starts.

This works from 0.2.0 on. With 0.1.0, or when Hachibu runs from a place it cannot replace (a folder you cannot write to, such as Program Files, or a zip you opened without extracting it), the menu item opens the download page instead. Download the new `Hachibu-windows-x64.zip`, quit Hachibu, and replace `Hachibu.exe` with the one in the zip.

Updates in the menu shows the version you are running, lets you turn off Check for Updates Automatically, and has Check Now.

### Uninstall

1. If you turned on Launch at Sign-in, turn it off from the menu
2. Quit Hachibu from the menu and delete `Hachibu.exe`
3. To remove its settings and logs as well, run `Remove-Item -Recurse "$env:LOCALAPPDATA\Hachibu"` in PowerShell
4. If you set up the statusLine script, remove it from the statusLine in `settings.json`

## Data

- Hachibu reads local files: Claude Code's saved conversations (only to find the most recently used model and effort), `settings.json`, and `statusline.json` above. In WSL it reads them through `\\wsl.localhost`
- It connects to the network for two things: the usage API, only if you turn it on (`api.anthropic.com`), and the update check once a day (`api.github.com`, which you can turn off). When the check finds a new version, Hachibu also downloads `Hachibu-windows-x64.zip` from the release on GitHub (`github.com`, which redirects to GitHub's download server). These requests send only the app's name and version as the User-Agent
- It reads the Claude Code login from `.credentials.json` only for the usage API request, and does not save or log it
- It writes its settings and the action log to `%LOCALAPPDATA%\Hachibu`, and keeps a downloaded update in its `update` folder until it restarts. Launch at Sign-in adds a `Hachibu` value under `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`

## How it differs from the macOS version

- A tray icon takes the place of the menu bar item. It draws the icon, a weekly gauge, or a number, and shows the full status in its tooltip and menu
- The default shortcut is Ctrl+Alt+/ instead of ⌥⌘/ (the same / key)
- Windows does not blur what is behind the strip, so the strip is a little more opaque
- In Task Manager it uses about 5 MB of memory

## Build from source

You need Go 1.26 or later. `build.sh` cross-compiles, so it runs on Linux, macOS, or Git Bash on Windows. It embeds the icon and the application manifest (per-monitor DPI and Common Controls 6.0, which the usage API dialog needs) with [go-winres](https://github.com/tc-hib/go-winres).

```sh
windows/build.sh        # builds windows/build/Hachibu.exe
windows/build.sh --zip  # also writes windows/build/Hachibu-windows-x64.zip and its SHA-256
cd windows && go test ./...
```

The decisions about what the strip shows live in `internal/core`, ported from `Sources/HachibuCore` and tested on any OS. The tests in `internal/app` run only on Windows; CI runs them on a Windows runner. When you change a rule there, change the Swift version too. UI text lives in `internal/core/strings.go`; update both English and Japanese.

For development, `Hachibu.exe --snapshot <folder>` draws the strip and the tray icons into PNG files and writes the menus to text, without showing any window. `HACHIBU_SOURCE=demo` shows made-up values, `HACHIBU_LANG=en|ja` picks the language, and `HACHIBU_SUITE=<name>` uses a separate settings file. With `HACHIBU_SUITE` set, `HACHIBU_OPEN=strip-menu|tray-menu|recorder|consent` opens that menu or window right after launch, for screenshots.

### Releasing

Set `VERSION` in `windows/build.sh`, write the notes in `docs/releases/windows-v<version>.md`, merge to `main`, and run the Release Windows workflow from the Actions tab. It tests, builds `Hachibu-windows-x64.zip`, and creates the tag `windows-v<version>` and the release with `statusline.sh`, `statusline.ps1`, and their SHA-256. The release is not marked as latest, so that Hachibu for macOS keeps pointing at its own releases. Then update the version in the download links in this README; the workflow run ends with a reminder.

## License

MIT (the repository's [LICENSE](../LICENSE))

---

## 日本語

Hachibu for Windowsは、Claude Codeのモデル、effort、使用率を、Windowsの画面の最前面の小さなバーに出し続けるアプリです。[Hachibu](../README.md#日本語)をWindowsで使うためのもので、バー、表示モード、色、メニューはmacOS向けのHachibuと同じです。GoとWin32 APIで書いていて、`Hachibu.exe`ひとつで動きます。ほかに入れるものはありません。

WindowsのClaude Codeと、WSLの中のClaude Codeのどちらでも使えます。

Hachibuは個人が作った非公式のツールで、Anthropic, PBCとは関係がなく、同社の承認や支援も受けていません。

[Hachibu for Windowsをダウンロード](https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip)（Windows 10または11、64ビット）。ArmのPCでは、Windowsのx64エミュレーションで動きます。コード署名をしていないため、初回の起動はWindows SmartScreenに止められます。手順は[インストール](#インストール)にあります。

### 表示の読み方

`Opus5• xhigh · S42 W76 F93`は、モデル、effort、使用率の順です。モデル名の後ろの小さな点は、1Mコンテキストを表します。Sは5時間枠、Wは週枠、Fはモデル別の週枠（モデル名の頭文字）です。同じ内容は、メニューの「表示の読み方」にもあります。

- 使用率が既定で70%以上なら黄、90%以上なら赤の札で数字を囲みます。閾値はどちらも変えられます
- バーにマウスを乗せると、各枠がいつリセットされるかが分かります。トレイのアイコンのメニューとツールチップにも出ます
- バーを触っても、使っているアプリからキー入力が離れません。ほかの窓の上に出ます
- ターミナルを開いているかどうかに関係なく動き続けます。メニューの「サインイン時に起動」を有効にすると、PCを再起動しても戻ります

### 値の出どころ

- モデルとeffort: Claude Codeが最後に保存した会話と、その隣の`settings.json`から読みます。下のstatusLineを設定していれば、そちらが新しいときはそちらを使います
- 使用率: 使用率API（任意）かstatusLineです。内容は[HachibuのREADME](../README.md#値の出どころ)と同じです。使用率APIは、Claude Codeが`.credentials.json`に保存したログイン情報を使います。有効にする前に、そこに書いたリスクを読んでください

Claude Codeの`.claude`フォルダは、次の2か所から探します。

- Windows: `%USERPROFILE%\.claude`。`CLAUDE_CONFIG_DIR`を設定していればその場所
- WSL: 動いているWSLのディストリビューションのホームフォルダ（`\\wsl.localhost`越し）。HachibuはWSLを起動しません。すでに動いているものだけを読みます

どこを読んでいるかは操作ログに残ります。

### statusLineの設定

任意です。使用率APIを使わない場合や、作業中のセッションのモデルとeffortを出したい場合に設定します。`statusline.sh`は、Claude CodeがstatusLineに渡すJSONを`%LOCALAPPDATA%\Hachibu\statusline.json`に保存し、Hachibuはそれを読みます。Claude Codeをどこで動かしているかで手順が変わります。

#### WSLのClaude Code

WSLで次を実行します。

```sh
curl -L -o ~/.claude/hachibu-statusline.sh https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/statusline.sh
chmod +x ~/.claude/hachibu-statusline.sh
```

そのうえで、WSLの`~/.claude/settings.json`のstatusLineから呼びます（設定の例は英語の節にあります）。スクリプトは、Windowsの`%LOCALAPPDATA%`を一度だけ`cmd.exe`で調べ、`~/.cache/hachibu/windows-dir`に覚えます。WSLからWindowsのプログラムを呼べない設定にしている場合は、`HACHIBU_DIR`に場所を渡してください（例：`HACHIBU_DIR=/mnt/c/Users/you/AppData/Local/Hachibu ~/.claude/hachibu-statusline.sh`）。

#### WindowsのClaude Code

WindowsのClaude Codeは、Git for Windowsが入っていればGit Bashで、無ければPowerShellでstatusLineを呼びます。Git Bashがあれば`statusline.sh`を、無ければ`statusline.ps1`を`%USERPROFILE%\.claude`に置き、`%USERPROFILE%\.claude\settings.json`のstatusLineから呼びます。ダウンロードのコマンドと設定の例は英語の節にあります。道筋は`/`で区切って書いてください。Git Bashは`\`を消してしまいます。

すでに別のstatusLineを使っている場合は、そのコマンドをどちらのスクリプトにも引数で渡せます。同じJSONがそのコマンドにも渡り、ステータス行の表示はそのコマンドの出力になります。

使用率（rate_limits）がstatusLineに入るのはPro/Maxのプランで、そのセッションの最初の応答の後からです。

### 表示を控えめにする

| モード | 出るもの |
|---|---|
| 標準 | モデル、effort、使用率 |
| 使用率だけ | `S42 W76 F93`のような使用率だけ |
| 端に収納 | 小さなつまみだけ。輪の色で、いちばん高い使用率の段階を示す |
| 隠す | 何も出さない。ショートカットかトレイのアイコンから出す |

バーを右クリックすると、表示モード、不透明度、色の閾値を切り替えられます。項目にマウスを乗せると、選ぶ前にバーへ仮に反映します。左端の点をドラッグするとバーを動かせます。画面のどちら側に置いたかを覚えます。

トレイのアイコン（タスクバーの右端の通知領域）にも同じメニューがあります。トレイのアイコンには文字を並べられないので、描くものを選べます。アイコンのみ、週枠のメーター（斜めの切れ目が週枠の使用率に合わせて動き、使った側に黄や赤の色が付く）、週枠の使用率を数字で、の3つです。ツールチップには状態の全文が出ます。

Windows 11は、新しいトレイのアイコンを「隠れているインジケーターを表示します」（^）の中にしまいます。タスクバーに出しておくには、「設定」＞「個人用設定」＞「タスク バー」＞「その他のシステム トレイ アイコン」でHachibuをオンにしてください。隠れている間は、バーの右クリックメニューにその設定を開く項目が出ます。トレイのアイコンのメニューでできることは、すべてバーの右クリックメニューからもできます。

Ctrl+Alt+/で、どのモードでもバーを呼び出せます（もう一度押すと戻ります）。メニューの「表示切り替えのショートカット…」で2つ目のショートカットを設定すると、標準、使用率だけ、端に収納を順に切り替えられます。ショートカットを登録するときは、Windowsが使っている組み合わせ（Alt+TabやCtrl+Shift+Escなど）と、ほかのアプリが登録している組み合わせを弾きます。新しい組み合わせをもう一度押して届くことを確かめてから、「保存」を押します。Windowsキーを使う組み合わせは、ほとんどをWindowsが使っているので記録しません。キーボードでボタンを使うときは、Tabでボタンに移り、Spaceで押します。

画面の文言は、Windowsの表示言語に合わせて日本語か英語になります。メニューの「言語」から英語か日本語を選ぶこともできます。

### インストール

1. [Hachibu-windows-x64.zip](https://github.com/BoxPistols/hachibu/releases/download/windows-v0.1.0/Hachibu-windows-x64.zip)をダウンロードして展開します
2. `Hachibu.exe`を、置いたままにできるフォルダ（`%LOCALAPPDATA%\Programs\Hachibu`など）に移します。「サインイン時に起動」はこの場所を覚えます
3. `Hachibu.exe`を開きます。初回はWindows SmartScreenが「WindowsによってPCが保護されました」と出すので、「詳細情報」を押してから「実行」を押します。まだコード署名の証明書で署名していないため、SmartScreenがこのアプリを知りません
4. 初回は、使用率APIを使うかを尋ねます

許可する前に確かめる方法があります。ソースはすべてこのリポジトリにあり、[自分でビルド](#ソースからビルド)できます。リリースのページにあるSHA-256と照合することもできます（`Get-FileHash Hachibu-windows-x64.zip`）。

同時に動くのは1つだけです。`Hachibu.exe`をもう一度開くと、バーを呼び出します。

#### アップデート

Hachibuは1日に1回、GitHubでHachibu for Windowsの新しいバージョンを確認します。見つけたら、裏で新しいバージョンを取得して確かめ、そのバージョンにつき1回だけWindowsの通知を出し、どちらのメニューの先頭にも「Hachibu x.y.zに更新して再起動」と出します。通知かこの項目を押すと、`Hachibu.exe`を置き換えて開き直します。設定はそのまま残ります。「今すぐ確認」で見つけたときは、更新して再起動するかを尋ねます。

置き換える前に、zipのSHA-256をリリースノートの値と突き合わせ、`Hachibu.exe`に入っている製品名とバージョンを確かめます。Windowsは動いているプログラムを上書きできないので、古いほうの名前を`Hachibu.exe.old`に変えてから新しいほうを置き、新しいほうが起動したときに古いほうを消します。

この仕組みは0.2.0からです。0.1.0を使っているときや、置き換えられない場所（Program Filesのような書き込めないフォルダや、展開せずに開いたzipの中）から動いているときは、項目を押すとダウンロードのページが開きます。新しい`Hachibu-windows-x64.zip`をダウンロードし、Hachibuを終了してから、`Hachibu.exe`をzipの中のものと入れ替えてください。

メニューの「アップデート」には、動いているバージョン、新しいバージョンを自動で確認するかの切り替え、「今すぐ確認」があります。

#### アンインストール

1. 「サインイン時に起動」を有効にしていた場合は、先にメニューから無効にします
2. メニューからHachibuを終了し、`Hachibu.exe`を消します
3. 設定とログも消す場合は、PowerShellで`Remove-Item -Recurse "$env:LOCALAPPDATA\Hachibu"`を実行します
4. statusLineのスクリプトを設定していた場合は、`settings.json`のstatusLineから外します

### データの扱い

- 読むのはローカルのファイルです。Claude Codeが保存した会話（最後に使われたモデルとeffortを知るためだけ）、`settings.json`、上の`statusline.json`。WSLのものは`\\wsl.localhost`越しに読みます
- ネットワークに接続するのは2つだけです。使用率APIを有効にした場合の`api.anthropic.com`と、1日に1回の新しいバージョンの確認の`api.github.com`です（確認は止められます）。新しいバージョンが見つかったときは、GitHubのリリースから`Hachibu-windows-x64.zip`も取得します（`github.com`から、GitHubのダウンロード用のサーバーに転送されます）。これらの通信で送るのは、User-Agentのアプリ名とバージョンだけです
- `.credentials.json`のログイン情報は、使用率APIへの問い合わせのときだけ読み、保存も記録もしません
- 設定と操作ログは`%LOCALAPPDATA%\Hachibu`に書きます。取得した新しいバージョンは、開き直すまでその中の`update`フォルダに置きます。「サインイン時に起動」を有効にすると、`HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run`に`Hachibu`という値を足します

### macOS向けのHachibuとの違い

- メニューバーの項目の代わりに、トレイのアイコンを使います。アイコン、週枠のメーター、数字のどれかを描き、状態の全文はツールチップとメニューに出します
- 既定のショートカットは⌥⌘/ではなくCtrl+Alt+/です（同じ「/」のキー）
- Windowsはバーの後ろをぼかさないので、バーを少し不透明にしています
- タスク マネージャーで見るメモリの使用量は約5 MBです

### ソースからビルド

Go 1.26以降が要ります。`build.sh`はWindows向けに作るので、Linux、macOS、WindowsのGit Bashのどれでも動きます。アイコンとアプリケーションマニフェスト（画面ごとの拡大率への対応と、使用率APIの確認の窓に要るCommon Controls 6.0）は[go-winres](https://github.com/tc-hib/go-winres)で埋め込みます。コマンドは英語の節にあります。

バーに何を出すかの判断は`internal/core`にあり、`Sources/HachibuCore`から移したものです。どのOSでもテストできます。`internal/app`のテストはWindowsでしか動かないので、CIはWindowsのランナーで動かします。判断を変えるときは、Swiftで書いたmacOS向けのコードも合わせて直してください。画面の文言は`internal/core/strings.go`にあります。英語と日本語の両方を直してください。

開発用に、`Hachibu.exe --snapshot <フォルダ>`で、窓を出さずにバーとトレイのアイコンをPNGに、メニューを文字に書き出せます。`HACHIBU_SOURCE=demo`は架空の値を出し、`HACHIBU_LANG=en|ja`は言語を選び、`HACHIBU_SUITE=<名前>`は別の設定ファイルを使います。`HACHIBU_SUITE`を付けたときだけ、`HACHIBU_OPEN=strip-menu|tray-menu|recorder|consent`で起動の直後にそのメニューや窓を開けます（画面写真を撮るため）。

#### リリースの手順

`windows/build.sh`の`VERSION`を上げ、`docs/releases/windows-v<バージョン>.md`に変更点を書いて`main`に入れ、ActionsタブからRelease Windowsのワークフローを実行します。テスト、`Hachibu-windows-x64.zip`の組み立て、タグ`windows-v<バージョン>`とリリースの作成（`statusline.sh`、`statusline.ps1`とSHA-256を添える）までを行います。macOS向けのHachibuの新しいバージョンの確認が自分のリリースを指し続けるよう、Hachibu for Windowsのリリースは「最新」にしません。そのあと、このREADMEのダウンロードのリンクのバージョンを上げてください（ワークフローの最後にも、その知らせが出ます）。

### ライセンス

MIT（リポジトリの[LICENSE](../LICENSE)）
