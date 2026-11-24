Changes

- Hachibu for Windows now updates itself. When it finds a new version, it downloads and checks it in the background, then shows Update to Hachibu x.y.z and Restart at the top of both menus. Click it, or the notification, to replace `Hachibu.exe` and restart. Settings stay. Before replacing, Hachibu checks the zip's SHA-256 against the one in the release notes, and the product name and version inside `Hachibu.exe`. When Check Now finds a new version, Hachibu asks whether to update and restart
- When Hachibu runs from a folder it cannot write to (such as Program Files) or from inside a zip that was not extracted, the menu item opens the download page as before
- `statusline.sh` no longer leaves `.statusline.*` files behind in `%LOCALAPPDATA%\Hachibu` when Claude Code stops it midway. If you use it, download the new `statusline.sh` from this release and put it where your statusLine points
- Some Japanese wording in the menus now reads the same way as in Hachibu for macOS

How to update

This is the last update you need to do by hand. Download `Hachibu-windows-x64.zip`, quit Hachibu from its menu, and replace `Hachibu.exe` with the one in the zip. Settings stay. The app is not code-signed yet, so Windows SmartScreen may stop the first launch again: click More info, then Run anyway.

---

変更点

- Hachibu for Windowsが自分で更新するようになりました。新しいバージョンを見つけると、裏で取得して確かめ、どちらのメニューの先頭にも「Hachibu x.y.zに更新して再起動」と出します。この項目か通知を押すと、`Hachibu.exe`を置き換えて開き直します。設定はそのまま残ります。置き換える前に、zipのSHA-256をリリースノートの値と突き合わせ、`Hachibu.exe`に入っている製品名とバージョンを確かめます。「今すぐ確認」で見つけたときは、更新して再起動するかを尋ねます
- 書き込めないフォルダ（Program Filesなど）や、展開していないzipの中から動いているときは、これまでどおり項目を押すとダウンロードのページが開きます
- Claude Codeが途中で止めたときに、`statusline.sh`が`%LOCALAPPDATA%\Hachibu`に`.statusline.*`のファイルを残さないようにしました。使っている場合は、このリリースの`statusline.sh`をダウンロードし、statusLineが指す場所に置き直してください
- メニューの日本語の言い方を、macOS向けのHachibuとそろえました

更新のしかた

手で入れ替えるのは今回が最後です。`Hachibu-windows-x64.zip`をダウンロードし、メニューからHachibuを終了してから、`Hachibu.exe`をzipの中のものと入れ替えてください。設定はそのまま残ります。まだコード署名をしていないため、初回の起動がWindows SmartScreenにもう一度止められることがあります。「詳細情報」を押してから「実行」を押してください。
