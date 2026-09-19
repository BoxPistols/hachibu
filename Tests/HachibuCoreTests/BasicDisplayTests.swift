import Foundation
import Testing
@testable import HachibuCore

@Suite struct UsageAPIParseTests {
    /// 2026-09-18に実際の応答で確認した形。値は架空
    static let response: [String: Any] = [
        "five_hour": ["utilization": 2.0, "resets_at": "2030-01-01T10:00:00.807700+00:00"],
        "seven_day": ["utilization": 81.0, "resets_at": "2030-01-03T00:00:00.807721+00:00"],
        "limits": [
            ["kind": "session", "percent": 2, "resets_at": "2030-01-01T10:00:00.807700+00:00", "scope": NSNull()],
            ["kind": "weekly_all", "percent": 81, "resets_at": "2030-01-03T00:00:00.807721+00:00", "scope": NSNull()],
            ["kind": "weekly_scoped", "percent": 55, "resets_at": "2030-01-02T23:59:59.807925+00:00",
             "scope": ["model": ["id": NSNull(), "display_name": "Examplemodel"], "surface": NSNull()]],
            ["kind": "something_new", "percent": 3],
        ],
    ]

    @Test func readsTheThreeLimits() {
        let limits = UsageAPI.parse(Self.response)
        #expect(limits.map(\.kind) == [.fiveHour, .weekly, .model("Examplemodel")])
        #expect(limits.map(\.percent) == [2, 81, 55])
        #expect(Usage.statusText(model: "Opus5 1M", effort: "xhigh", limits: limits) == "Opus5 1M xhigh · S2 W81 E55")
    }

    @Test func fallsBackToUtilizationAsPercent() {
        var obj = Self.response
        obj["limits"] = nil
        // utilizationは百分率。2.0を比率と見て200%にも、0.5を50%にもしない
        #expect(UsageAPI.parse(obj).map(\.percent) == [2, 81])
        #expect(UsageAPI.parse(["five_hour": ["utilization": 0.5]]).map(\.percent) == [1])
    }

    @Test func skipsScopedLimitWithoutName() {
        let obj: [String: Any] = ["limits": [["kind": "weekly_scoped", "percent": 10, "scope": ["model": ["display_name": " "]]]]]
        #expect(UsageAPI.parse(obj).isEmpty)
    }
}

@Suite struct ModelInfoTests {
    @Test func compactsModelIDs() {
        #expect(ModelInfo.compactID("claude-opus-5") == "Opus5")
        #expect(ModelInfo.compactID("claude-opus-5[1m]") == "Opus5 1M")
        #expect(ModelInfo.compactID("claude-sonnet-5-20260101") == "Sonnet5")
        #expect(ModelInfo.compactID("claude-haiku-4-5-20251001") == "Haiku4.5")
        #expect(ModelInfo.compactID("gpt-4") == nil)
    }

    @Test func findsTheLastModelInTheTranscript() {
        let tail = """
        {"type":"assistant","message":{"model":"claude-sonnet-5","content":[]}}
        {"type":"user","message":{"content":"model is fine"}}
        {"type":"assistant","message":{"model":"claude-opus-5","content":[]}}
        """
        #expect(ModelInfo.lastModelID(inTranscriptTail: tail) == "claude-opus-5")
        #expect(ModelInfo.lastModelID(inTranscriptTail: "{\"type\":\"user\"}") == nil)
    }

    @Test func combinesTranscriptModelWithContextFromSettings() {
        #expect(ModelInfo.displayName(transcriptModelID: "claude-opus-5", settingsModel: "opus[1m]") == "Opus5 1M")
        #expect(ModelInfo.displayName(transcriptModelID: "claude-opus-5[1m]", settingsModel: "opus[1m]") == "Opus5 1M")
        #expect(ModelInfo.displayName(transcriptModelID: "claude-sonnet-5", settingsModel: nil) == "Sonnet5")
        // 設定が別の系統のモデルを指しているときは、その1Mを付けない
        #expect(ModelInfo.displayName(transcriptModelID: "claude-sonnet-5", settingsModel: "opus[1m]") == "Sonnet5")
        #expect(ModelInfo.displayName(transcriptModelID: "claude-sonnet-5", settingsModel: "claude-sonnet-5[1m]") == "Sonnet5 1M")
    }

    @Test func fallsBackToTheSettingsAlias() {
        #expect(ModelInfo.displayName(transcriptModelID: nil, settingsModel: "opus[1m]") == "Opus 1M")
        #expect(ModelInfo.displayName(transcriptModelID: nil, settingsModel: "sonnet") == "Sonnet")
        #expect(ModelInfo.displayName(transcriptModelID: nil, settingsModel: nil) == nil)
    }
}
