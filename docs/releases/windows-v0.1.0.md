The first release of Hachibu for Windows. It keeps Claude Code's model, effort, and usage limits in a small strip on top of your screen, like Hachibu for macOS.

- The strip has the same modes (Standard, Usage only, Tucked, Hidden), colors, opacity, and right-click menu as the macOS version. Hover over it to expand it and to see when each limit resets
- A tray icon takes the place of the menu bar item. It can draw the app icon, a weekly gauge, or the weekly usage as a number
- Ctrl+Alt+/ shows the strip. You can record your own shortcuts; combinations that Windows uses are rejected
- Reads Claude Code on Windows (`%USERPROFILE%\.claude`) and in WSL distributions that are running, without starting WSL
- Usage comes from the statusLine or, if you turn it on, the usage API. `statusline.sh` now also works in WSL and in Git Bash on Windows, and `statusline.ps1` covers Windows without Git Bash
- Launch at Sign-in, English and Japanese, and a daily update check (tags `windows-v*`)

Download `Hachibu-windows-x64.zip`, extract it, and open `Hachibu.exe` (Windows 10 or 11, 64-bit). It is not code-signed yet, so Windows SmartScreen stops the first launch: click More info, then Run anyway. See [windows/README.md](https://github.com/BoxPistols/hachibu/blob/main/windows/README.md).

---

Hachibu for Windowsの最初のリリースです。macOS版のHachibuと同じように、Claude Codeのモデル、effort、使用率を画面の最前面の小さなバーに出し続けます。

- バーの表示モード（標準、使用率だけ、端に収納、隠す）、色、不透明度、右クリックメニューはmacOS版と同じです。マウスを乗せると広がり、各枠がいつリセットされるかが分かります
- メニューバーの項目の代わりに、トレイのアイコンを使います。アイコン、週枠のメーター、週枠の使用率の数字のどれかを描けます
- Ctrl+Alt+/でバーを呼び出します。ショートカットは自分で登録でき、Windowsが使っている組み合わせは弾きます
- WindowsのClaude Code（`%USERPROFILE%\.claude`）と、動いているWSLのディストリビューションのClaude Codeを読みます。WSLを起動はしません
- 使用率はstatusLineか、有効にした場合は使用率APIから取ります。`statusline.sh`はWSLとWindowsのGit Bashでも使えるようにし、Git Bashの無いWindows向けに`statusline.ps1`を足しました
- サインイン時の起動、英語と日本語、1日に1回の新しい版の確認（タグ`windows-v*`）

`Hachibu-windows-x64.zip`を落として展開し、`Hachibu.exe`を開きます（Windows 10または11、64ビット）。まだコード署名をしていないため、初回の起動はWindows SmartScreenに止められます。「詳細情報」を押してから「実行」を押してください。詳しくは[windows/README.md](https://github.com/BoxPistols/hachibu/blob/main/windows/README.md)にあります。
