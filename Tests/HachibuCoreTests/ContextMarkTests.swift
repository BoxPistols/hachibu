import Testing
@testable import HachibuCore

struct ContextMarkTests {
    @Test func takesTheContextLengthOutOfTheStatusText() {
        let s = ContextMark.split("Examplemodel5 1M xhigh · S2 W81")
        #expect(s == ContextMark.Split(model: "Examplemodel5", context: "1M", rest: " xhigh · S2 W81"))
        #expect(ContextMark.split("Examplemodel5 200K") == ContextMark.Split(model: "Examplemodel5", context: "200K", rest: ""))
    }

    @Test func leavesTextWithoutAContextLengthAlone() {
        #expect(ContextMark.split("Examplemodel5 xhigh · S2") == nil)
        #expect(ContextMark.split("S2 W81 F55") == nil)
        // 使用率の語（"M12"のような枠の頭文字＋数字）をコンテキスト長と取り違えない
        #expect(ContextMark.split("Examplemodel5 M12 W81") == nil)
    }

    @Test func spellsItOutForTheMenu() {
        #expect(ContextMark.spelledOut("Examplemodel5 1M xhigh · S2 W81") == "Examplemodel5 · 1M context · xhigh")
        #expect(ContextMark.spelledOut("Examplemodel5 xhigh · S2") == nil)
    }
}
