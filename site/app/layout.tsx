import { Analytics } from "@vercel/analytics/next";
import type { Metadata } from "next";
import { Bricolage_Grotesque, JetBrains_Mono, Noto_Sans_JP } from "next/font/google";
import type { ReactNode } from "react";
import "./globals.css";

const display = Bricolage_Grotesque({ subsets: ["latin"], variable: "--font-display-face", display: "swap" });
const mono = JetBrains_Mono({ subsets: ["latin"], variable: "--font-mono-face", display: "swap" });
const japanese = Noto_Sans_JP({ subsets: ["latin"], weight: ["400", "600"], variable: "--font-ja-face", display: "swap" });

const TITLE = "Hachibu — Claude Code usage, always in view";
const DESCRIPTION =
  "A small strip on top of your Mac's screen that shows Claude Code's model, effort, and usage limits. Free and open source. Unofficial.";

export const metadata: Metadata = {
  metadataBase: new URL("https://cc-hachibu.vercel.app"),
  title: TITLE,
  description: DESCRIPTION,
  icons: { icon: "/app-icon.png" },
  openGraph: { title: TITLE, description: DESCRIPTION, type: "website", images: ["/og.png"] },
  // Xやチャットにリンクを貼ったときの横長の画像。作り直すときはtools/og.htmlを1200x630で撮る
  twitter: { card: "summary_large_image", title: TITLE, description: DESCRIPTION, images: ["/og.png"] },
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en" className={`${display.variable} ${mono.variable} ${japanese.variable}`}>
      <body>
        {children}
        {/* Vercelのアクセス解析。訪問者を識別する情報は集めない */}
        <Analytics />
      </body>
    </html>
  );
}
