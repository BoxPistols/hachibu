import Foundation
import SlashstripCore

/// 選んだセッションのタブへ移る。iTerm2のセッションGUIDで直接選ぶので、並び順や状態に左右されない
enum SessionFocus {
    // GUIDは引数で渡す（スクリプト本文に埋め込まない）
    private static let script = """
    on run argv
      set target to item 1 of argv
      tell application "iTerm2"
        repeat with w in windows
          repeat with t in tabs of w
            repeat with s in sessions of t
              if id of s is target then
                select s
                select t
                select w
                activate
                return "ok"
              end if
            end repeat
          end repeat
        end repeat
      end tell
      return "notfound"
    end run
    """

    static func focus(_ session: SessionInfo) {
        let label = "focus \(session.project)"
        if let guid = session.termGUID {
            ScriptRunner.run("/usr/bin/osascript", ["-e", script, guid], label: label)
        } else if let bundle = session.hostBundle, !bundle.isEmpty {
            // デスクトップアプリで動いているセッションはタブを選べないので、アプリを前面に出す
            ScriptRunner.run("/usr/bin/open", ["-b", bundle], label: label)
        } else {
            ActionLog.append("\(label): 移動先が分かりません")
        }
    }
}
