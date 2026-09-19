# Hachibuの紹介ページ

<https://cc-hachibu.vercel.app/>のソースです。Next.js 15の静的書き出しとTailwind CSS v4で作っています。

```sh
npm install
npm run dev     # http://localhost:3000
npm run build   # out/ に書き出す
```

画面に出る文言は、英語と日本語のどちらも`lib/content.ts`にあります。コンポーネントに文字列を直接書かず、ここを直してください。

`components/GaugeGlyph.tsx`と`components/Strip.tsx`は、アプリの見た目を実寸で再現しています。アプリ側の`Sources/Hachibu/MenuBarIcon.swift`と`Sources/Hachibu/StripView.swift`を変えたときは、こちらも合わせてください。

デプロイはVercelです。

```sh
vercel deploy --prod --yes --scope asagiri
```
