The first release of Hachibu Lite, the lightweight edition of Hachibu. It was called TokenBar (github.com/BoxPistols/tokenbar) up to 0.2.0 and now lives in this repository.

Changes from TokenBar 0.2.0

- Renamed to Hachibu Lite (`HachibuLite.app`)
- Reads the same `statusline.sh` as Hachibu (`~/Library/Application Support/Hachibu/statusline.json`), and still reads TokenBar's file, using whichever is newer
- The update check looks for `lite-v` tags in this repository's releases
- Refresh Now (⌘R) also checks for a new version and shows the result in the menu bar for a moment
- On first launch, takes over TokenBar's settings and its Launch at Login entry

How to move from TokenBar

Download `HachibuLite-macos-x86_64.zip`, put `HachibuLite.app` in your Applications folder, and open it. Then quit TokenBar and delete `TokenBar.app`. Intel Macs on macOS 13 or later only; the app is not notarized, so allow the first launch in System Settings > Privacy & Security. See [lite/README.md](https://github.com/BoxPistols/hachibu/blob/main/lite/README.md).

---

Hachibuの軽量版、Hachibu Liteの最初のリリースです。0.2.0まではTokenBar（github.com/BoxPistols/tokenbar）という名前で、このリポジトリに移しました。

TokenBar 0.2.0からの変更点

- 名前をHachibu Lite（`HachibuLite.app`）に変えました
- Hachibuと同じ`statusline.sh`が書くファイル（`~/Library/Application Support/Hachibu/statusline.json`）を読みます。TokenBarのファイルも読み、新しい方を使います
- 新しい版の確認は、このリポジトリのリリースから`lite-v`のタグを探します
- 「今すぐ更新」（⌘R）で新しい版も確認し、結果をメニューバーに少しの間出します
- 初めて起動したときに、TokenBarの設定と「ログイン時に起動」の登録を引き継ぎます

TokenBarから移る手順

`HachibuLite-macos-x86_64.zip`を落として、`HachibuLite.app`をアプリケーションのフォルダに置いて開きます。そのあとTokenBarを終了し、`TokenBar.app`を消してください。macOS 13以降のIntel Macだけで動きます。公証を受けていないので、初回の起動は「システム設定」＞「プライバシーとセキュリティ」で許可してください。詳しくは[lite/README.md](https://github.com/BoxPistols/hachibu/blob/main/lite/README.md)にあります。
