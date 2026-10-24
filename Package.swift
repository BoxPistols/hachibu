// swift-tools-version:5.10
import PackageDescription

// HACHIBU_APP_STORE=1でビルドすると、使用率APIと更新確認を外したApp Store版になる（scripts/build-app.sh --app-store）
let appStore = Context.environment["HACHIBU_APP_STORE"] == "1"

let package = Package(
    name: "Hachibu",
    platforms: [.macOS(.v14)],
    targets: [
        // 画面を持たない判断部分。Swift Testingで検査する（Command Line ToolsではXCTestが動かない）
        .target(name: "HachibuCore", path: "Sources/HachibuCore"),
        .executableTarget(name: "Hachibu", dependencies: ["HachibuCore"], path: "Sources/Hachibu",
                          swiftSettings: appStore ? [.define("APP_STORE")] : []),
        .testTarget(name: "HachibuCoreTests", dependencies: ["HachibuCore"], path: "Tests/HachibuCoreTests"),
    ]
)
