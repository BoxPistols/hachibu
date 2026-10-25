import AppKit
import Carbon.HIToolbox

// Hachibu Lite。statusline.sh が保存した JSON を読み、メニューバーと画面上の帯に 5時間枠/週枠 の使用率を表示する
// Hachibu の軽量版（Swift 1 ファイル、AppKit のみ）。使用率は公式の statusLine からのみ受け取る（キーチェーン・非公開 API は使わない）
// 旧名は TokenBar（github.com/BoxPistols/tokenbar）。初回起動時に TokenBar の設定を引き継ぐ

enum Level: Int { case normal, warning, critical }

struct Limit {
    let label: String; let name: String; let pct: Int; let resets: Date?
    // リセット時刻を過ぎていれば古い値なので 0% 扱い（読み込み時と、直前の値を使い回すときの両方で使う）
    var current: Limit { resets.map { $0 < Date() } ?? false ? Limit(label: label, name: name, pct: 0, resets: nil) : self }
}

struct Snapshot {
    var model: String?
    var effort: String?
    var oneM = false
    var limits: [Limit] = []
    var updated: Date?

    static func load(_ file: URL) -> Snapshot? {
        guard let data = try? Data(contentsOf: file),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { return nil }
        var s = Snapshot()
        s.model = (obj["model"] as? [String: Any])?["display_name"] as? String
        s.effort = (obj["effort"] as? [String: Any])?["level"] as? String
        s.oneM = ((obj["context_window"] as? [String: Any])?["context_window_size"] as? NSNumber)?.intValue ?? 0 >= 1_000_000
        let rates = obj["rate_limits"] as? [String: Any] ?? [:]
        for (key, label, name) in [("five_hour", "S", "5時間枠"), ("seven_day", "W", "週枠")] {
            guard let r = rates[key] as? [String: Any],
                  let pct = (r["used_percentage"] as? NSNumber)?.doubleValue else { continue }
            let resets = (r["resets_at"] as? NSNumber).map { Date(timeIntervalSince1970: $0.doubleValue) }
            s.limits.append(Limit(label: label, name: name, pct: Int(pct.rounded()), resets: resets).current)
        }
        s.updated = (try? FileManager.default.attributesOfItem(atPath: file.path))?[.modificationDate] as? Date
        return s
    }
}

// 表示モード・メニューバー表示・しきい値などの設定。UserDefaults に保存する
enum Mode: String, CaseIterable { case standard, usage, tucked, hidden }
enum BarStyle: String, CaseIterable { case icon, gauge, usage, full }

struct Prefs {
    static let d = UserDefaults.standard
    static let warningChoices = [50, 60, 70, 80], criticalChoices = [80, 85, 90, 95]
    static let opacityChoices = [1.0, 0.85, 0.7, 0.5, 0.3]
    static var warning: Int { get { d.object(forKey: "warning") as? Int ?? 80 } set { d.set(newValue, forKey: "warning") } }
    static var critical: Int { get { d.object(forKey: "critical") as? Int ?? 90 } set { d.set(newValue, forKey: "critical") } }
    static var mode: Mode { get { Mode(rawValue: d.string(forKey: "mode") ?? "") ?? .standard } set { d.set(newValue.rawValue, forKey: "mode") } }
    static var lastVisible: Mode { get { Mode(rawValue: d.string(forKey: "lastVisible") ?? "") ?? .standard } set { d.set(newValue.rawValue, forKey: "lastVisible") } }
    static var barStyle: BarStyle { get { BarStyle(rawValue: d.string(forKey: "barStyle") ?? "") ?? .usage } set { d.set(newValue.rawValue, forKey: "barStyle") } }
    static var opacity: Double { get { d.object(forKey: "opacity") as? Double ?? 1.0 } set { d.set(newValue, forKey: "opacity") } }
    static var stripOrigin: NSPoint? {
        get { d.string(forKey: "stripOrigin").map { NSPointFromString($0) } }
        set { d.set(newValue.map { NSStringFromPoint($0) }, forKey: "stripOrigin") }
    }
    static func level(_ pct: Int) -> Level { pct >= critical ? .critical : pct >= warning ? .warning : .normal }

    /// TokenBar（local.tokenbar）の設定を一度だけ写す。こちらで設定を変えた後に戻されないよう、写したら印を付ける
    static func migrateFromTokenBar() {
        guard !d.bool(forKey: "migratedFromTokenBar"), let old = UserDefaults(suiteName: "local.tokenbar") else { return }
        d.set(true, forKey: "migratedFromTokenBar")
        // メニューバーの中の位置も写す（macOS が NSStatusItem の名前で覚えている）
        for key in ["warning", "critical", "mode", "lastVisible", "barStyle", "opacity", "stripOrigin", "NSStatusItem Preferred Position Item-0"] {
            if let v = old.object(forKey: key), d.object(forKey: key) == nil { d.set(v, forKey: key) }
        }
        LoginItem.migrateFromTokenBar()
    }
}

// Hachibu と同じ配色。黄は黒字、赤は白字で札（背景色）にする
let yellow = NSColor(srgbRed: 242/255, green: 201/255, blue: 76/255, alpha: 1)
let red = NSColor(srgbRed: 217/255, green: 48/255, blue: 37/255, alpha: 1)
func levelColor(_ l: Level) -> NSColor? { l == .critical ? red : l == .warning ? yellow : nil }

/// "Opus5.5• medium · S12 W76" の形の文字列。使用率の語だけを札で囲む
func statusText(_ s: Snapshot?, full: Bool, font: NSFont, plain: NSColor? = nil) -> NSAttributedString {
    let out = NSMutableAttributedString()
    var base: [NSAttributedString.Key: Any] = [.font: font]
    if let plain { base[.foregroundColor] = plain }
    guard let s, !s.limits.isEmpty || full else { out.append(NSAttributedString(string: "CC –", attributes: base)); return out }
    if full {
        var head = [s.model.map { $0.replacingOccurrences(of: " ", with: "") + (s.oneM ? "•" : "") }, s.effort].compactMap { $0 }.joined(separator: " ")
        if !head.isEmpty && !s.limits.isEmpty { head += " ·" }
        if !head.isEmpty { out.append(NSAttributedString(string: head, attributes: base)) }
    }
    for l in s.limits {
        var a = base
        if let c = levelColor(Prefs.level(l.pct)) { a[.backgroundColor] = c; a[.foregroundColor] = c == red ? NSColor.white : NSColor.black }
        if out.length > 0 { out.append(NSAttributedString(string: " ", attributes: base)) }
        out.append(NSAttributedString(string: " \(l.label)\(l.pct) ", attributes: a))
    }
    return out
}

func worstLevel(_ s: Snapshot?) -> Level { Level(rawValue: s?.limits.map { Prefs.level($0.pct).rawValue }.max() ?? 0) ?? .normal }

// MARK: - 画面上の帯

final class StripView: NSView {
    let label = NSTextField(labelWithString: "")
    var onHover: ((Bool) -> Void)?
    var menuProvider: (() -> NSMenu)?
    var tucked = false { didSet { label.isHidden = tucked; needsDisplay = true } }
    var ring: NSColor = .systemGray { didSet { needsDisplay = true } }

    override init(frame: NSRect) {
        super.init(frame: frame)
        wantsLayer = true
        addSubview(label)
    }
    required init?(coder: NSCoder) { fatalError() }

    override func draw(_ r: NSRect) {
        if tucked {
            // しまった状態は小さな丸い取っ手。輪の色で最も高い段階を示す
            NSColor(white: 0.12, alpha: 0.9).setFill()
            let c = NSBezierPath(ovalIn: bounds.insetBy(dx: 2, dy: 2)); c.fill()
            ring.setStroke(); c.lineWidth = 3; c.stroke()
        } else {
            NSColor(white: 0.12, alpha: 0.9).setFill()
            NSBezierPath(roundedRect: bounds, xRadius: 6, yRadius: 6).fill()
        }
    }

    override func layout() {
        super.layout()
        label.sizeToFit()
        label.setFrameOrigin(NSPoint(x: 8, y: (bounds.height - label.frame.height) / 2))
    }

    override func updateTrackingAreas() {
        trackingAreas.forEach(removeTrackingArea)
        addTrackingArea(NSTrackingArea(rect: bounds, options: [.mouseEnteredAndExited, .activeAlways, .inVisibleRect], owner: self))
    }
    override func mouseEntered(with e: NSEvent) { onHover?(true) }
    override func mouseExited(with e: NSEvent) { onHover?(false) }
    override func rightMouseDown(with e: NSEvent) {
        if let m = menuProvider?() { NSMenu.popUpContextMenu(m, with: e, for: self) }
    }
    override func mouseUp(with e: NSEvent) { Prefs.stripOrigin = window?.frame.origin }
}

final class Strip {
    let panel: NSPanel
    let view = StripView(frame: .zero)
    var hovering = false
    // 「今すぐ更新」の結果。帯の右クリックから押した人にも見えるよう、帯にも出す
    var flash: String?

    init() {
        panel = NSPanel(contentRect: NSRect(x: 0, y: 0, width: 100, height: 22), styleMask: [.borderless, .nonactivatingPanel], backing: .buffered, defer: false)
        // フォーカスを奪わず、全スペース・フルスクリーンの上にも出す（Hachibu と同じ設定）
        panel.level = .floating
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .stationary, .ignoresCycle]
        panel.isFloatingPanel = true
        panel.hidesOnDeactivate = false
        panel.backgroundColor = .clear
        panel.isOpaque = false
        panel.hasShadow = true
        panel.isMovableByWindowBackground = true
        panel.contentView = view
    }

    func update(_ s: Snapshot?) {
        let mode = Prefs.mode
        // 隠すと mouseExited が届かないことがあるので、ここでホバーを解く
        guard mode != .hidden else { hovering = false; panel.orderOut(nil); return }
        // しまった状態・使用率のみでも、ホバー中は一時的に標準表示に広げる
        let shown: Mode = hovering && mode != .standard ? .standard : mode
        view.tucked = shown == .tucked
        view.ring = levelColor(worstLevel(s)) ?? .systemGray
        let font = NSFont.monospacedDigitSystemFont(ofSize: 12, weight: .medium)
        // 結果を出す間は、しまった状態でも文字が読める大きさに広げる
        let tucked = shown == .tucked && flash == nil
        view.tucked = tucked
        view.label.attributedStringValue = flash.map { NSAttributedString(string: $0, attributes: [.font: font, .foregroundColor: NSColor.white]) }
            ?? statusText(s, full: shown == .standard, font: font, plain: .white)
        view.label.sizeToFit()
        let size = tucked ? NSSize(width: 16, height: 16) : NSSize(width: view.label.frame.width + 16, height: 22)
        panel.alphaValue = Prefs.opacity
        let origin = Prefs.stripOrigin ?? defaultOrigin(size)
        panel.setFrame(clamped(NSRect(origin: origin, size: size)), display: true)
        view.needsLayout = true
        panel.orderFrontRegardless()
    }

    // 初期位置はメニューバーのすぐ下の中央
    func defaultOrigin(_ size: NSSize) -> NSPoint {
        let v = NSScreen.main?.visibleFrame ?? NSRect(x: 0, y: 0, width: 1440, height: 900)
        return NSPoint(x: v.midX - size.width / 2, y: v.maxY - size.height - 4)
    }

    // 幅が変わっても画面の外にはみ出さないようにする
    func clamped(_ f: NSRect) -> NSRect {
        guard let v = (NSScreen.screens.first { $0.visibleFrame.intersects(f) } ?? NSScreen.main)?.visibleFrame else { return f }
        var r = f
        r.origin.x = min(max(r.minX, v.minX), v.maxX - r.width)
        r.origin.y = min(max(r.minY, v.minY), v.maxY - r.height)
        return r
    }
}

// MARK: - グローバルショートカット（Carbon。アクセシビリティ権限は不要）

final class HotKeys {
    static var handlers: [UInt32: () -> Void] = [:]
    static func register(_ id: UInt32, key: Int, mods: Int, _ action: @escaping () -> Void) {
        if handlers.isEmpty {
            var spec = EventTypeSpec(eventClass: OSType(kEventClassKeyboard), eventKind: UInt32(kEventHotKeyPressed))
            InstallEventHandler(GetApplicationEventTarget(), { _, event, _ in
                var hk = EventHotKeyID()
                GetEventParameter(event, EventParamName(kEventParamDirectObject), EventParamType(typeEventHotKeyID), nil, MemoryLayout<EventHotKeyID>.size, nil, &hk)
                DispatchQueue.main.async { HotKeys.handlers[hk.id]?() }
                return noErr
            }, 1, &spec, nil, nil)
        }
        handlers[id] = action
        var ref: EventHotKeyRef?
        RegisterEventHotKey(UInt32(key), UInt32(mods), EventHotKeyID(signature: OSType(0x48424C54), id: id), GetApplicationEventTarget(), 0, &ref)
    }
}

// MARK: - ログイン時の自動起動（~/Library/LaunchAgents の plist を置く／消す）

enum LoginItem {
    static let label = "io.github.boxpistols.hachibu.lite"
    static let agents = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/LaunchAgents")
    static let plist = agents.appendingPathComponent("\(label).plist")
    static var isEnabled: Bool { FileManager.default.fileExists(atPath: plist.path) }

    /// TokenBar でログイン時に起動していたら、TokenBar の登録を消してこちらを登録する（両方が起動しないように）
    static func migrateFromTokenBar() {
        let old = agents.appendingPathComponent("local.tokenbar.plist")
        guard FileManager.default.fileExists(atPath: old.path) else { return }
        try? FileManager.default.removeItem(at: old)
        set(true)
    }

    static func set(_ on: Bool) {
        if on {
            let dict: [String: Any] = ["Label": label, "ProgramArguments": ["/usr/bin/open", "-a", Bundle.main.bundlePath], "RunAtLoad": true]
            // 新しいアカウントでは LaunchAgents のフォルダがまだ無いことがある
            try? FileManager.default.createDirectory(at: agents, withIntermediateDirectories: true)
            (dict as NSDictionary).write(to: plist, atomically: true)
        } else {
            try? FileManager.default.removeItem(at: plist)
        }
    }
}

// MARK: - 新しい版の確認（1 日 1 回 GitHub のリリースを見る。送るのは User-Agent だけ）

enum Updates {
    static let current = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "0"
    // Hachibu と同じリポジトリに置くので、/releases/latest（Hachibu の版）ではなく一覧から lite-v のタグを探す
    static let url = URL(string: "https://api.github.com/repos/BoxPistols/hachibu/releases?per_page=30")!
    static let tagPrefix = "lite-v"
    static var available: (version: String, page: URL)?

    /// 確認の結果。失敗の理由を分けて、メニューに何が起きたかを出す
    enum Result: Equatable {
        case ok, offline, rateLimited, http(Int), badResponse
        /// メニューバーと帯に出す短い語（幅を取らないように）
        var short: String {
            switch self {
            case .ok: return Updates.available.map { "\($0.version) あり" } ?? "最新"
            case .offline: return "オフライン"
            case .rateLimited: return "確認は制限中"
            case .http, .badResponse: return "確認できず"
            }
        }
        /// メニューに出す説明
        var detail: String {
            switch self {
            case .ok: return Updates.available.map { "\($0.version) が出ています" } ?? "最新版です"
            case .offline: return "ネットワークにつながらず確認できませんでした"
            case .rateLimited: return "GitHub の回数制限で確認できませんでした（1 時間ほどで戻ります）"
            case .http(let code): return "GitHub が HTTP \(code) を返しました"
            case .badResponse: return "GitHub の応答を読めませんでした"
            }
        }
    }
    static private(set) var lastResult: Result?
    static private(set) var lastChecked: Date?
    // 確認中にもう一度呼ばれたら、新しく問い合わせずに同じ結果を渡す
    static private var waiting: [(Result) -> Void]?

    /// 前回の確認から minInterval 秒以内なら問い合わせず、前回の結果を渡す（⌘R の連打で GitHub の制限を使い切らないように）。
    /// 失敗しても available は前回の結果のまま残す
    static func check(timeout: TimeInterval = 30, minInterval: TimeInterval = 0, _ done: @escaping (Result) -> Void) {
        if let at = lastChecked, let r = lastResult, Date().timeIntervalSince(at) < minInterval { done(r); return }
        if waiting != nil { waiting?.append(done); return }
        waiting = [done]
        var req = URLRequest(url: url, timeoutInterval: timeout)
        req.setValue("HachibuLite/\(current)", forHTTPHeaderField: "User-Agent")
        // shared はこの Mac で毎回 10 秒以上かかり、30 秒で時間切れになることもあった。キャッシュも cookie も要らないので ephemeral にする
        URLSession(configuration: .ephemeral).dataTask(with: req) { data, response, error in
            let (result, found) = parse(data: data, status: (response as? HTTPURLResponse)?.statusCode, failed: error != nil)
            DispatchQueue.main.async {
                if result == .ok { available = found }
                lastResult = result
                lastChecked = Date()
                let callbacks = waiting ?? []
                waiting = nil
                callbacks.forEach { $0(result) }
            }
        }.resume()
    }

    /// 結果と、見つけた新しい版（無ければ nil）を返す
    private static func parse(data: Data?, status: Int?, failed: Bool) -> (Result, (version: String, page: URL)?) {
        guard !failed, let status else { return (.offline, nil) }
        // GitHub は回数制限を 403 か 429 で返す
        if status == 403 || status == 429 { return (.rateLimited, nil) }
        guard status == 200 else { return (.http(status), nil) }
        guard let list = data.flatMap({ try? JSONSerialization.jsonObject(with: $0) as? [[String: Any]] }) else { return (.badResponse, nil) }
        // 一覧は新しい順。下書きと試験版は除く
        let release = list.first {
            ($0["tag_name"] as? String)?.hasPrefix(tagPrefix) == true && $0["draft"] as? Bool != true && $0["prerelease"] as? Bool != true
        }
        // Lite のリリースがまだ無いのは失敗ではない
        guard let release, let tag = release["tag_name"] as? String,
              let page = (release["html_url"] as? String).flatMap(URL.init(string:)) else { return (.ok, nil) }
        let latest = String(tag.dropFirst(tagPrefix.count))
        // "0.10.0" > "0.9.0" になるよう数値として比べる
        let newer = latest.compare(current, options: .numeric) == .orderedDescending
        return (.ok, newer ? (latest, page) : nil)
    }
}

// MARK: - アプリ本体

final class AppDelegate: NSObject, NSApplicationDelegate {
    let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
    // Hachibu と同じ statusline.sh が書く場所を読む。TokenBar の statusline.sh のままの人のために旧い場所も見て、新しい方を使う
    let files = ["Hachibu", "TokenBar"].map {
        FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("Library/Application Support/\($0)/statusline.json")
    }
    let strip = Strip()
    var snapshot: Snapshot?
    var timer: Timer?
    var updateTimer: Timer?
    // 「今すぐ更新」の結果をメニューバーと帯に少しの間だけ出す。flashID は、後から押した分の表示を先の分のタイマーが消さないための番号
    var flashText: String?
    var flashID = 0

    func applicationDidFinishLaunching(_ n: Notification) {
        strip.view.menuProvider = { [unowned self] in makeMenu() }
        strip.view.onHover = { [unowned self] h in strip.hovering = h; strip.update(snapshot) }
        // ⌥⌘/ で帯の表示／非表示、⌥⌘. で 標準→使用率のみ→しまう を切り替える
        HotKeys.register(1, key: kVK_ANSI_Slash, mods: optionKey | cmdKey) { [unowned self] in
            setMode(Prefs.mode == .hidden ? Prefs.lastVisible : .hidden)
        }
        HotKeys.register(2, key: kVK_ANSI_Period, mods: optionKey | cmdKey) { [unowned self] in
            let cycle: [Mode] = [.standard, .usage, .tucked]
            setMode(cycle[((cycle.firstIndex(of: Prefs.mode) ?? -1) + 1) % cycle.count])
        }
        refresh()
        timer = Timer.scheduledTimer(withTimeInterval: 15, repeats: true) { [weak self] _ in self?.refresh() }
        // 確認の結果はメニューにしか出ないので、メニューだけ作り直す
        Updates.check { [weak self] _ in self?.rebuildMenu() }
        updateTimer = Timer.scheduledTimer(withTimeInterval: 24 * 60 * 60, repeats: true) { [weak self] _ in
            Updates.check { _ in self?.rebuildMenu() }
        }
    }

    func refresh() {
        let loaded = files.compactMap { Snapshot.load($0) }.sorted { ($0.updated ?? .distantPast) > ($1.updated ?? .distantPast) }
        var s = loaded.first
        // Hachibu の statusline.sh は rate_limits の無い JSON でも書く。そのときは使用率の入った新しい方か、直前の値を使う
        if s?.limits.isEmpty == true { s?.limits = (loaded.first { !$0.limits.isEmpty }?.limits ?? snapshot?.limits ?? []).map(\.current) }
        snapshot = s
        rebuildMenu()
        updateBar()
        strip.update(snapshot)
    }

    func rebuildMenu() { item.menu = makeMenu() }

    func updateBar() {
        let font = NSFont.monospacedDigitSystemFont(ofSize: 12, weight: .medium)
        item.button?.image = nil
        item.button?.attributedTitle = NSAttributedString()
        if let t = flashText {
            item.button?.attributedTitle = NSAttributedString(string: t, attributes: [.font: font])
            return
        }
        switch Prefs.barStyle {
        case .icon:
            let img = NSImage(systemSymbolName: "gauge.medium", accessibilityDescription: "Hachibu Lite")
            img?.isTemplate = true
            item.button?.image = img
        case .gauge:
            // 週枠の使用率を横棒で示し、数字を 1 つだけ添える
            let w = snapshot?.limits.first { $0.label == "W" }
            item.button?.image = gaugeImage(w?.pct ?? 0)
            item.button?.imagePosition = .imageLeft
            item.button?.attributedTitle = NSAttributedString(string: w.map { " W\($0.pct)" } ?? " W–", attributes: [.font: font])
        case .usage, .full:
            item.button?.attributedTitle = statusText(snapshot, full: Prefs.barStyle == .full, font: font)
        }
    }

    // バッテリー表示（横長の箱）と見分けられるよう、下が開いた丸いメーターにする。使った分だけ弧を塗る
    func gaugeImage(_ pct: Int) -> NSImage {
        let img = NSImage(size: NSSize(width: 16, height: 16), flipped: false) { r in
            let c = NSPoint(x: r.midX, y: r.midY), radius: CGFloat = 6
            let start: CGFloat = 225, sweep: CGFloat = 270
            let track = NSBezierPath()
            track.appendArc(withCenter: c, radius: radius, startAngle: start, endAngle: start - sweep, clockwise: true)
            track.lineWidth = 2.5; track.lineCapStyle = .round
            NSColor.labelColor.withAlphaComponent(0.25).setStroke(); track.stroke()
            guard pct > 0 else { return true }
            let used = NSBezierPath()
            used.appendArc(withCenter: c, radius: radius, startAngle: start, endAngle: start - sweep * CGFloat(min(pct, 100)) / 100, clockwise: true)
            used.lineWidth = 2.5; used.lineCapStyle = .round
            (levelColor(Prefs.level(pct)) ?? .labelColor).setStroke(); used.stroke()
            return true
        }
        return img
    }

    func makeMenu() -> NSMenu {
        let menu = NSMenu()
        if let u = Updates.available {
            menu.addItem(withTitle: "Hachibu Lite \(u.version) が出ています（いまは \(Updates.current)）", action: #selector(openUpdate), keyEquivalent: "").target = self
            menu.addItem(.separator())
        }
        if let s = snapshot {
            for l in s.limits {
                let when = l.resets.map { DateFormatter.localizedString(from: $0, dateStyle: .short, timeStyle: .short) } ?? "-"
                menu.addItem(withTitle: "\(l.name): \(l.pct)%  (リセット \(when))", action: nil, keyEquivalent: "")
            }
            if let m = s.model { menu.addItem(withTitle: "モデル: \(m)\(s.oneM ? "（1M）" : "")\(s.effort.map { " / effort \($0)" } ?? "")", action: nil, keyEquivalent: "") }
            if let u = s.updated { menu.addItem(withTitle: "最終更新: " + DateFormatter.localizedString(from: u, dateStyle: .none, timeStyle: .short), action: nil, keyEquivalent: "") }
        } else {
            menu.addItem(withTitle: "データ待ち（Claude Code で一度応答を受けると表示）", action: nil, keyEquivalent: "")
        }
        menu.addItem(.separator())
        let modeNames: [Mode: String] = [.standard: "標準", .usage: "使用率のみ", .tucked: "しまう", .hidden: "非表示"]
        addChoice(menu, "帯の表示", Mode.allCases.map { (modeNames[$0]!, $0 == Prefs.mode, $0.rawValue) }, #selector(pickMode), hint: "⌥⌘/ 表示切替・⌥⌘. モード切替")
        addChoice(menu, "帯の不透明度", Prefs.opacityChoices.map { ("\(Int($0 * 100))%", $0 == Prefs.opacity, String($0)) }, #selector(pickOpacity))
        let barNames: [BarStyle: String] = [.icon: "アイコンのみ", .gauge: "週枠ゲージ", .usage: "使用率", .full: "すべて"]
        addChoice(menu, "メニューバーの表示", BarStyle.allCases.map { (barNames[$0]!, $0 == Prefs.barStyle, $0.rawValue) }, #selector(pickBar))
        // 赤は黄より大きい値に保つ（Hachibu と同じ制約）。逆転する選択肢は選べなくする
        addChoice(menu, "黄にする使用率", Prefs.warningChoices.map { ("\($0)% 以上（残り \(100 - $0)%）", $0 == Prefs.warning, String($0)) }, #selector(pickWarning), enabled: { Int($0)! < Prefs.critical })
        addChoice(menu, "赤にする使用率", Prefs.criticalChoices.map { ("\($0)% 以上（残り \(100 - $0)%）", $0 == Prefs.critical, String($0)) }, #selector(pickCritical), enabled: { Int($0)! > Prefs.warning })
        menu.addItem(.separator())
        if let r = Updates.lastResult, let at = Updates.lastChecked {
            menu.addItem(withTitle: "新しい版の確認（\(DateFormatter.localizedString(from: at, dateStyle: .none, timeStyle: .short))）: \(r.detail)", action: nil, keyEquivalent: "")
        }
        let login = menu.addItem(withTitle: "ログイン時に起動", action: #selector(toggleLogin), keyEquivalent: "")
        login.target = self; login.state = LoginItem.isEnabled ? .on : .off
        menu.addItem(withTitle: "帯を初期位置に戻す", action: #selector(resetPosition), keyEquivalent: "").target = self
        menu.addItem(withTitle: "今すぐ更新", action: #selector(refreshNow), keyEquivalent: "r").target = self
        menu.addItem(withTitle: "終了", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        return menu
    }

    func addChoice(_ menu: NSMenu, _ title: String, _ choices: [(String, Bool, String)], _ action: Selector, hint: String? = nil, enabled: ((String) -> Bool)? = nil) {
        let sub = NSMenu()
        for (name, on, value) in choices {
            let i = sub.addItem(withTitle: name, action: (enabled?(value) ?? true) ? action : nil, keyEquivalent: "")
            i.target = self; i.state = on ? .on : .off; i.representedObject = value
        }
        if let hint { sub.addItem(.separator()); sub.addItem(withTitle: hint, action: nil, keyEquivalent: "") }
        let current = choices.first { $0.1 }?.0 ?? ""
        menu.setSubmenu(sub, for: menu.addItem(withTitle: "\(title): \(current)", action: nil, keyEquivalent: ""))
    }

    func setMode(_ m: Mode) {
        if m != .hidden { Prefs.lastVisible = m }
        Prefs.mode = m
        refresh()
    }

    @objc func pickMode(_ s: NSMenuItem) { setMode(Mode(rawValue: s.representedObject as! String)!) }
    @objc func pickOpacity(_ s: NSMenuItem) { Prefs.opacity = Double(s.representedObject as! String)!; refresh() }
    @objc func pickBar(_ s: NSMenuItem) { Prefs.barStyle = BarStyle(rawValue: s.representedObject as! String)!; refresh() }
    @objc func pickWarning(_ s: NSMenuItem) { Prefs.warning = Int(s.representedObject as! String)!; refresh() }
    @objc func pickCritical(_ s: NSMenuItem) { Prefs.critical = Int(s.representedObject as! String)!; refresh() }
    @objc func toggleLogin() { LoginItem.set(!LoginItem.isEnabled); refresh() }
    @objc func resetPosition() { Prefs.stripOrigin = nil; refresh() }
    // 使用率を読み直し、新しい版も確認する。メニューは押すと閉じるので、結果はメニューバーと帯に短く出す。
    // 使用率はその場で読み直すので、確認を待つ間も表示は消さない。手で押したときは 10 秒で打ち切り、1 分以内なら前回の結果を使う
    @objc func refreshNow() {
        refresh()
        let gotUsage = snapshot != nil
        Updates.check(timeout: 10, minInterval: 60) { [weak self] result in
            self?.rebuildMenu()
            // 使用率を読めなかったときは ✓ を付けない（詳しくはメニューの「データ待ち」）
            self?.flash(gotUsage ? "✓ \(result.short)" : "データ待ち")
        }
    }

    func flash(_ text: String) {
        flashID += 1
        let id = flashID
        setFlash(text)
        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) { [weak self] in
            guard self?.flashID == id else { return }
            self?.setFlash(nil)
        }
    }

    func setFlash(_ text: String?) {
        flashText = text
        strip.flash = text
        updateBar()
        strip.update(snapshot)
    }
    @objc func openUpdate() { if let u = Updates.available { NSWorkspace.shared.open(u.page) } }
}

let app = NSApplication.shared
// メニューバーの項目は AppDelegate を作るときにできるので、位置を含む設定はその前に写す
Prefs.migrateFromTokenBar()
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
