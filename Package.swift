// swift-tools-version:5.10
import PackageDescription

let package = Package(
    name: "Slashstrip",
    platforms: [.macOS(.v14)],
    targets: [
        // 画面を持たない判断部分。Swift Testingで検査する（Command Line ToolsではXCTestが動かない）
        .target(name: "SlashstripCore", path: "Sources/SlashstripCore"),
        .executableTarget(name: "Slashstrip", dependencies: ["SlashstripCore"], path: "Sources/Slashstrip"),
        .testTarget(name: "SlashstripCoreTests", dependencies: ["SlashstripCore"], path: "Tests/SlashstripCoreTests"),
    ]
)
