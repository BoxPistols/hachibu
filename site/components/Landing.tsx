"use client";

import { useEffect, useState } from "react";
import {
  ARTICLE_URL,
  CONTENT,
  DOWNLOAD_URL,
  FIRST_LAUNCH_URL,
  HARA_HACHI_BU,
  RELEASES_URL,
  REPO_URL,
  TERMS_URL,
  type Lang,
} from "@/lib/content";
import { GaugeGlyph } from "./GaugeGlyph";
import { Playground } from "./Playground";

const HERO_GAUGE = { width: 1200, thickness: 150 } as const;

/** ページ全体。言語の切り替えだけを持ち、文言はすべてlib/contentから読む */
export function Landing() {
  const [lang, setLang] = useState<Lang>("en");
  const t = CONTENT[lang];

  // 最初の表示だけ、ブラウザの言語に合わせる
  useEffect(() => {
    if (navigator.language.startsWith("ja")) setLang("ja");
  }, []);

  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);

  return (
    <div className={lang === "ja" ? "font-ja" : undefined}>
      <header className="mx-auto flex max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <a href="#top" className="flex items-center gap-3 font-display text-lg font-semibold text-paper">
          <img src="/app-icon.png" alt="" width={32} height={32} />
          Hachibu
        </a>
        <nav className="flex items-center gap-2 text-sm sm:gap-4">
          <a href={ARTICLE_URL} className="hidden rounded-lg px-2 py-1.5 text-fog hover:text-paper sm:block">
            {t.nav.article}
          </a>
          <a href={REPO_URL} className="rounded-lg px-2 py-1.5 text-fog hover:text-paper">
            {t.nav.github}
          </a>
          <button
            type="button"
            onClick={() => setLang(lang === "en" ? "ja" : "en")}
            className="rounded-lg border border-white/15 px-3 py-1.5 text-fog hover:border-white/40 hover:text-paper"
          >
            {t.langLabel}
          </button>
        </nav>
      </header>

      <main id="top">
        <section className="relative overflow-hidden">
          <div className="pointer-events-none absolute inset-x-0 top-24 flex justify-center opacity-90 sm:top-16">
            <GaugeGlyph {...HERO_GAUGE} percent={HARA_HACHI_BU} className="h-auto w-[140%] max-w-none sm:w-[110%]" />
          </div>
          <div className="relative mx-auto max-w-6xl px-5 pb-20 pt-52 sm:px-8 sm:pt-80">
            <p className="font-mono text-sm text-yolk">{t.hero.eyebrow}</p>
            <h1 className="mt-4 max-w-4xl font-display text-4xl font-semibold leading-[1.12] text-paper sm:text-6xl">
              {t.hero.title}
            </h1>
            <p className="mt-6 max-w-2xl text-lg leading-relaxed text-fog">{t.hero.lead}</p>
            <p className="mt-3 max-w-2xl text-base text-fog/80">{t.hero.nameNote}</p>
            <div className="mt-9 flex flex-wrap items-center gap-4">
              <a
                href={DOWNLOAD_URL}
                className="rounded-lg bg-yolk px-6 py-3 text-base font-semibold text-ink transition-transform hover:-translate-y-0.5"
              >
                {t.hero.download}
              </a>
              <a href={REPO_URL} className="rounded-lg border border-white/20 px-6 py-3 text-base text-paper hover:border-white/50">
                {t.hero.source}
              </a>
            </div>
            <p className="mt-4 text-sm text-fog/80">{t.hero.requirements}</p>
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-16 sm:px-8">
          <h2 className="font-display text-3xl font-semibold text-paper">{t.playground.heading}</h2>
          <p className="mt-3 max-w-3xl text-base leading-relaxed text-fog">{t.playground.lead}</p>
          <div className="mt-8">
            <Playground text={t.playground} />
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-16 sm:px-8">
          <h2 className="font-display text-3xl font-semibold text-paper">{t.features.heading}</h2>
          <div className="mt-8 grid gap-x-10 gap-y-8 sm:grid-cols-2">
            {t.features.items.map((item) => (
              <div key={item.title}>
                <h3 className="text-lg font-semibold text-paper">{item.title}</h3>
                <p className="mt-2 text-base leading-relaxed text-fog">{item.body}</p>
              </div>
            ))}
          </div>
          <figure className="mt-12">
            <video
              src="/hachibu-demo.mp4"
              className="w-full rounded-xl border border-white/10"
              autoPlay
              loop
              muted
              playsInline
              aria-label={t.video.label}
            />
            <figcaption className="mt-3 text-sm text-fog/80">{t.video.label}</figcaption>
          </figure>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-16 sm:px-8">
          <h2 className="font-display text-3xl font-semibold text-paper">{t.sources.heading}</h2>
          <div className="mt-8 grid gap-6 lg:grid-cols-2">
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-6">
              <h3 className="text-lg font-semibold text-paper">{t.sources.statusLineTitle}</h3>
              <p className="mt-2 text-base leading-relaxed text-fog">{t.sources.statusLineBody}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/[0.03] p-6">
              <h3 className="text-lg font-semibold text-paper">{t.sources.apiTitle}</h3>
              <p className="mt-2 text-base leading-relaxed text-fog">{t.sources.apiBody}</p>
              <a href={TERMS_URL} className="mt-3 inline-block text-base text-yolk underline underline-offset-4">
                {t.sources.termsLink}
              </a>
            </div>
          </div>
        </section>

        <section className="mx-auto max-w-6xl px-5 py-16 sm:px-8">
          <h2 className="font-display text-3xl font-semibold text-paper">{t.install.heading}</h2>
          <p className="mt-3 max-w-3xl text-base leading-relaxed text-fog">{t.install.lead}</p>
          <ol className="mt-8 grid gap-6 lg:grid-cols-3">
            {t.install.steps.map((step, index) => (
              <li key={step.title} className="rounded-xl border border-white/10 bg-white/[0.03] p-6">
                <span className="font-mono text-sm text-yolk">{index + 1}</span>
                <h3 className="mt-2 text-lg font-semibold text-paper">{step.title}</h3>
                <p className="mt-2 text-base leading-relaxed text-fog">{step.body}</p>
              </li>
            ))}
          </ol>
          <div className="mt-8 flex flex-wrap items-center gap-x-6 gap-y-3 text-base">
            <a href={DOWNLOAD_URL} className="rounded-lg bg-yolk px-6 py-3 font-semibold text-ink">
              {t.hero.download}
            </a>
            <a href={FIRST_LAUNCH_URL[lang]} className="text-yolk underline underline-offset-4">
              {t.install.guideLink}
            </a>
            <a href={RELEASES_URL} className="text-fog underline underline-offset-4">
              {t.install.checksum}
            </a>
          </div>
        </section>
      </main>

      <footer className="mx-auto max-w-6xl border-t border-white/10 px-5 py-10 text-sm leading-relaxed text-fog/80 sm:px-8">
        <p>{t.footer.unofficial}</p>
        <p>{t.footer.notTheFoodApp}</p>
        <p className="mt-2">
          {t.footer.trademarks} {t.footer.license}
        </p>
        <p className="mt-2 text-xs text-fog/60">{t.footer.privacy}</p>
      </footer>
    </div>
  );
}
