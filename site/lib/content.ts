// 画面に出る文言はすべてここに置く（英語と日本語）。コンポーネントに文字列を直接書かない

export type Lang = "en" | "ja";

export const REPO_URL = "https://github.com/BoxPistols/hachibu";
export const DOWNLOAD_URL = `${REPO_URL}/releases/latest/download/Hachibu-macos.zip`;
export const RELEASES_URL = `${REPO_URL}/releases/latest`;
export const FIRST_LAUNCH_URL: Record<Lang, string> = {
  en: `${REPO_URL}#first-launch-allow-it-once-in-system-settings`,
  ja: `${REPO_URL}#インストール`,
};
export const TERMS_URL = "https://code.claude.com/docs/en/legal-and-compliance";

// アプリの既定の閾値と同じ
export const WARNING_AT = 70;
export const CRITICAL_AT = 90;
export const HARA_HACHI_BU = 80;

export const MODES = ["full", "usage", "tab"] as const;
export type Mode = (typeof MODES)[number];

export interface Feature {
  title: string;
  body: string;
}

export interface Step {
  title: string;
  body: string;
}

export interface Content {
  langLabel: string;
  nav: { github: string; download: string };
  hero: {
    eyebrow: string;
    title: string;
    lead: string;
    nameNote: string;
    download: string;
    requirements: string;
    source: string;
    gaugeCaption: string;
  };
  playground: {
    heading: string;
    lead: string;
    sliderLabel: string;
    modeLabel: string;
    modes: Record<Mode, string>;
    menuBarLabel: string;
    stripLabel: string;
    legend: { dot: string; s: string; w: string; f: string; warning: string; critical: string };
  };
  features: { heading: string; items: Feature[] };
  sources: {
    heading: string;
    statusLineTitle: string;
    statusLineBody: string;
    apiTitle: string;
    apiBody: string;
    termsLink: string;
  };
  install: { heading: string; lead: string; steps: Step[]; guideLink: string; checksum: string };
  footer: { unofficial: string; notTheFoodApp: string; trademarks: string; license: string };
  video: { label: string };
}

export const CONTENT: Record<Lang, Content> = {
  en: {
    langLabel: "日本語",
    nav: { github: "GitHub", download: "Download" },
    hero: {
      eyebrow: "A macOS app for Claude Code",
      title: "I spend a worrying amount of my life checking my Claude Code usage.",
      lead: "So I made it stay on screen. Hachibu is a small strip on top of your Mac's screen that shows the model, effort, and usage limits. One glance is enough, and you can get back to your work.",
      nameNote: "The name comes from hara hachi bu (腹八分), the Japanese habit of eating until you are 80% full.",
      download: "Download for macOS",
      requirements: "macOS 14 or later · Apple Silicon and Intel · Free and open source (MIT)",
      source: "View the source",
      gaugeCaption: "80%",
    },
    playground: {
      heading: "This is the actual size",
      lead: "Move the weekly usage and watch the strip and the menu bar gauge react. Yellow starts at 70% and red at 90%; you can change both in the app.",
      sliderLabel: "Weekly usage",
      modeLabel: "Display",
      modes: { full: "Standard", usage: "Usage only", tab: "Tucked" },
      menuBarLabel: "Menu bar · Weekly gauge",
      stripLabel: "Strip",
      legend: {
        dot: "Dot after the model name: 1M context",
        s: "S: 5-hour limit",
        w: "W: weekly limit",
        f: "F: weekly limit for one model",
        warning: "70% or more",
        critical: "90% or more",
      },
    },
    features: {
      heading: "It stays out of the way",
      items: [
        {
          title: "Visible without opening anything",
          body: "The strip floats above other windows, including full-screen apps. You do not have to click a menu bar icon or type /usage to see where you stand.",
        },
        {
          title: "It never takes focus",
          body: "Clicking or dragging the strip does not bring Hachibu to the front, so the text you are typing is never interrupted.",
        },
        {
          title: "As small as you want",
          body: "Show only the usage, tuck it into a handle at the screen edge, or hide it until you press a shortcut. Hover to expand. A second shortcut steps through the displays.",
        },
        {
          title: "Or keep it in the menu bar",
          body: "The weekly gauge takes about as much room as the battery indicator. The icon's slanted cut moves with your weekly usage.",
        },
      ],
    },
    sources: {
      heading: "Where the numbers come from",
      statusLineTitle: "statusLine (official)",
      statusLineBody:
        "Claude Code passes the model, effort, context window, and the 5-hour and weekly usage to a statusLine command. A small script saves it for Hachibu. It updates while you use Claude Code in the terminal.",
      apiTitle: "Usage API (optional, off by default)",
      apiBody:
        "Hachibu can use the login that Claude Code saved in your keychain to ask Anthropic's usage endpoint, which adds per-model limits and keeps updating without a terminal. Anthropic's terms for Claude Code restrict third-party use of that login, so Hachibu shows you the wording and only turns this on if you choose to. You can turn it off at any time.",
      termsLink: "Read the terms",
    },
    install: {
      heading: "Install",
      lead: "Hachibu is not from the App Store and is not notarized yet, so macOS asks you to allow it once.",
      steps: [
        { title: "Download and move", body: "Unzip Hachibu-macos.zip and move Hachibu.app to your Applications folder." },
        { title: "Open it once", body: "macOS says “Hachibu.app” Not Opened. Click Done. Do not move it to the Trash." },
        {
          title: "Allow it in System Settings",
          body: "Open System Settings > Privacy & Security, scroll to Security, click Open Anyway next to the message about Hachibu, and enter your password.",
        },
      ],
      guideLink: "Steps with screenshots",
      checksum: "The SHA-256 of each download is on the release page.",
    },
    footer: {
      unofficial:
        "Hachibu is an unofficial tool made by an individual. It is not affiliated with, endorsed by, or sponsored by Anthropic, PBC.",
      notTheFoodApp: "It is also unrelated to the meal logging app of the same name.",
      trademarks: "Claude and Claude Code are trademarks of Anthropic, PBC.",
      license: "MIT License",
    },
    video: { label: "Demo: hover to expand, right-click to switch the display" },
  },
  ja: {
    langLabel: "English",
    nav: { github: "GitHub", download: "ダウンロード" },
    hero: {
      eyebrow: "Claude CodeのためのmacOSアプリ",
      title: "人生の多くの時間を、Claude Codeの残量確認に使っています。",
      lead: "なので画面に出しっぱなしにしました。Hachibuは、Macの画面の最前面に小さなバーを常駐させ、モデル、effort、使用率を出すアプリです。ひと目で済むので、すぐ手元の作業に戻れます。",
      nameNote: "名前は「腹八分」から取りました。",
      download: "macOS版をダウンロード",
      requirements: "macOS 14以降 · Apple SiliconとIntel · 無料、オープンソース（MIT）",
      source: "ソースを見る",
      gaugeCaption: "80%",
    },
    playground: {
      heading: "これが実際の大きさです",
      lead: "週枠の使用率を動かすと、バーとメニューバーのメーターが変わります。既定では70%から黄、90%から赤です。どちらもアプリの中で変えられます。",
      sliderLabel: "週枠の使用率",
      modeLabel: "表示",
      modes: { full: "標準", usage: "使用率だけ", tab: "端に収納" },
      menuBarLabel: "メニューバー · 週枠のメーター",
      stripLabel: "バー",
      legend: {
        dot: "モデル名の後ろの点：1Mコンテキスト",
        s: "S：5時間枠",
        w: "W：週枠",
        f: "F：モデル別の週枠",
        warning: "70%以上",
        critical: "90%以上",
      },
    },
    features: {
      heading: "作業の邪魔をしません",
      items: [
        {
          title: "開かなくても見える",
          body: "バーは、フルスクリーンのアプリを含め、ほかの窓の上に出ます。メニューバーのアイコンを押したり、/usageを打ったりしなくても、いまの残量が分かります。",
        },
        {
          title: "フォーカスを奪わない",
          body: "バーを押しても動かしても、Hachibuは前面に出ません。入力中の文字が途切れません。",
        },
        {
          title: "好きなだけ小さく",
          body: "使用率だけにする、画面の端のつまみに収納する、ショートカットを押すまで隠す、から選べます。マウスを乗せると広がります。2つ目のショートカットで、表示を順に切り替えられます。",
        },
        {
          title: "メニューバーだけでも",
          body: "「週枠のメーター」は、バッテリーの表示と同じくらいの幅です。アイコンの斜めの切れ目が、週枠の使用率に合わせて動きます。",
        },
      ],
    },
    sources: {
      heading: "値の出どころ",
      statusLineTitle: "statusLine（公式）",
      statusLineBody:
        "Claude Codeは、モデル、effort、コンテキスト幅、5時間枠と週枠の使用率をstatusLineのコマンドに渡します。小さなスクリプトがそれを保存し、Hachibuが読みます。ターミナルでClaude Codeを使っている間に更新されます。",
      apiTitle: "使用率API（任意、既定では無効）",
      apiBody:
        "Claude Codeがキーチェーンに保存したログイン情報を使って、Anthropicの使用率のエンドポイントに問い合わせることもできます。モデル別の枠が加わり、ターミナルを開いていなくても更新されます。Claude Codeの規約は、このログイン情報の第三者による利用を制限しています。Hachibuはその文面を示し、利用者が選んだ場合だけ有効にします。いつでも無効にできます。",
      termsLink: "規約を読む",
    },
    install: {
      heading: "インストール",
      lead: "HachibuはApp Storeで配布しておらず、Appleの公証もまだ受けていないため、macOSが一度だけ許可を求めます。",
      steps: [
        { title: "ダウンロードして移す", body: "Hachibu-macos.zipを展開し、Hachibu.appをアプリケーションフォルダへ移します。" },
        { title: "一度開く", body: "「“Hachibu.app”は開いていません」と出るので、「完了」を押します。ゴミ箱には入れないでください。" },
        {
          title: "システム設定で許可する",
          body: "「システム設定」＞「プライバシーとセキュリティ」を開き、「セキュリティ」の欄でHachibuについての表示の横にある「このまま開く」を押して、パスワードを入力します。",
        },
      ],
      guideLink: "画像付きの手順",
      checksum: "各ダウンロードのSHA-256は、リリースのページにあります。",
    },
    footer: {
      unofficial: "Hachibuは個人が作った非公式のツールで、Anthropic, PBCとは関係がなく、同社の承認や支援も受けていません。",
      notTheFoodApp: "同名の食事記録アプリとも関係ありません。",
      trademarks: "Claude、Claude CodeはAnthropic, PBCの商標です。",
      license: "MITライセンス",
    },
    video: { label: "デモ：マウスを乗せると広がり、右クリックで表示を切り替える" },
  },
};
