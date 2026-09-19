import AppKit

// Dockに出さない常駐アプリとして起動する（メニューバーとストリップだけを持つ）
let app = NSApplication.shared
// 開発用: 画面に何も出さずに各画面を画像へ書き出して終わる
if let dir = Snapshot.outputDirectory {
    app.setActivationPolicy(.prohibited)
    Snapshot.run(to: dir)
    exit(0)
}
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
