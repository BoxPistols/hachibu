import Foundation
import Testing
@testable import HachibuCore

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

    @Test func findsTheLastEffortInTheTranscript() {
        let tail = """
        {"type":"assistant","message":{"model":"claude-opus-5-5","content":[]},"effort":"xhigh","perTurnEffort":"xhigh"}
        {"type":"user","message":{"content":"effort is fine"}}
        {"type":"assistant","message":{"model":"claude-opus-5-5","content":[]},"effort":"medium","perTurnEffort":"medium"}
        """
        #expect(ModelInfo.lastEffort(inTranscriptTail: tail) == "medium")
        #expect(ModelInfo.lastEffort(inTranscriptTail: "{\"type\":\"user\"}") == nil)
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
