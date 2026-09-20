"use client";

import { GoogleAnalytics } from "@next/third-parties/google";
import { Analytics } from "@vercel/analytics/next";
import { useEffect, useState } from "react";

interface SiteAnalyticsProps {
  gaId: string;
}

/// EUとイギリスからの訪問ではCookieを置かない。時間帯の設定で判定する（サーバーを持たない静的なページのため）
const COOKIE_CONSENT_REGIONS = ["Europe/", "Atlantic/Azores", "Atlantic/Canary", "Atlantic/Madeira", "Atlantic/Reykjavik"];
/// この端末を数えない印。`?analytics=off`で付き、`?analytics=on`で外れる
const OPT_OUT_KEY = "hachibu-analytics-off";
/// Vercelの計測を止める印（Vercelが読む）
const VERCEL_DISABLE_KEY = "va-disable";

function needsCookieConsent(timeZone: string): boolean {
  return COOKIE_CONSENT_REGIONS.some((region) => timeZone.startsWith(region));
}

/// 端末に印を付け外しする。プライベートウインドウなどで書けないことがあるので、失敗しても進める
function applyOptOut(off: boolean) {
  try {
    if (off) {
      localStorage.setItem(OPT_OUT_KEY, "1");
      localStorage.setItem(VERCEL_DISABLE_KEY, "1");
    } else {
      localStorage.removeItem(OPT_OUT_KEY);
      localStorage.removeItem(VERCEL_DISABLE_KEY);
    }
  } catch {
    // 書けないときは何もしない
  }
}

function isOptedOut(): boolean {
  try {
    return localStorage.getItem(OPT_OUT_KEY) === "1";
  } catch {
    return false;
  }
}

/**
 * Vercelの解析はCookieを使わないので、どの地域でも読み込む。
 * GA4はCookieを使うので、同意が要る地域では読み込まない。同意を求める窓は出さない。
 * `?analytics=off`を付けて開いた端末は、どちらの計測からも外す
 */
export function SiteAnalytics({ gaId }: SiteAnalyticsProps) {
  const [state, setState] = useState<{ count: boolean; google: boolean }>({ count: false, google: false });

  useEffect(() => {
    const param = new URLSearchParams(window.location.search).get("analytics");
    if (param === "off" || param === "on") applyOptOut(param === "off");

    if (isOptedOut()) {
      setState({ count: false, google: false });
      return;
    }
    let google = false;
    try {
      google = !needsCookieConsent(Intl.DateTimeFormat().resolvedOptions().timeZone ?? "");
    } catch {
      // 時間帯が読めないときはCookieを置かない
    }
    setState({ count: true, google });
  }, []);

  if (!state.count) return null;
  return (
    <>
      <Analytics />
      {state.google && <GoogleAnalytics gaId={gaId} />}
    </>
  );
}
