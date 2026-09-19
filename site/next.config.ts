import type { NextConfig } from "next";

// 1ページの静的なLP。サーバーの処理は無いので、静的に書き出す
const nextConfig: NextConfig = {
  output: "export",
  images: { unoptimized: true },
};

export default nextConfig;
