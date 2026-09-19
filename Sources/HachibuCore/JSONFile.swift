import Foundation

public enum JSONFile {
    /// JSONの辞書を読む。ファイルが無い、書きかけ、辞書でない場合はnil
    public static func object(at url: URL) -> [String: Any]? {
        guard let data = try? Data(contentsOf: url) else { return nil }
        return (try? JSONSerialization.jsonObject(with: data)) as? [String: Any]
    }
}
