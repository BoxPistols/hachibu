import AppKit

// Dockに出さない常駐アプリとして起動する（メニューバーとストリップだけを持つ）
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
