// swift-tools-version:5.10
import PackageDescription

let package = Package(
    name: "Hachibu",
    platforms: [.macOS(.v14)],
    targets: [
        // 画面を持たない判断部分。Swift Testingで検査する（Command Line ToolsではXCTestが動かない）
        .target(name: "HachibuCore", path: "Sources/HachibuCore"),
        .executableTarget(name: "Hachibu", dependencies: ["HachibuCore"], path: "Sources/Hachibu"),
        .testTarget(name: "HachibuCoreTests", dependencies: ["HachibuCore"], path: "Tests/HachibuCoreTests"),
    ]
)
