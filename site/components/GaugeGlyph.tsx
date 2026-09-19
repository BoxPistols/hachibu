import type { Level } from "@/lib/level";

interface GaugeGlyphProps {
  /** 全体の幅（px） */
  width: number;
  /** 帯の太さ（px） */
  thickness: number;
  /** 切れ目の位置（0〜100） */
  percent: number;
  /** 左（使った側）の色を決める段階。指定しなければアプリのアイコンと同じ配色（左が灰、右が黄） */
  level?: Level;
  className?: string;
}

const FILL: Record<Level, string> = {
  normal: "var(--color-fog)",
  warning: "var(--color-yolk)",
  critical: "var(--color-brick)",
};

/** アプリのアイコンと同じ「斜めの切れ目で2つに分かれた帯」。比率はアプリと揃える（切れ目は太さの1/3、傾きは0.6） */
export function GaugeGlyph({ width, thickness, percent, level, className }: GaugeGlyphProps) {
  const gap = thickness / 3;
  const slant = thickness * 0.6;
  const reach = slant / 2 + gap / 2 + thickness * 0.3;
  const cut = Math.min(Math.max((width * percent) / 100, reach), width - reach);
  const left = `0,${thickness} ${cut - gap / 2 - slant / 2},${thickness} ${cut - gap / 2 + slant / 2},0 0,0`;
  const right = `${cut + gap / 2 - slant / 2},${thickness} ${width},${thickness} ${width},0 ${cut + gap / 2 + slant / 2},0`;
  const isGauge = level !== undefined;
  return (
    <svg width={width} height={thickness} viewBox={`0 0 ${width} ${thickness}`} className={className} aria-hidden="true">
      <polygon points={left} fill={isGauge ? FILL[level] : "var(--color-slate-light)"} />
      <polygon points={right} fill={isGauge ? "var(--color-fog)" : "var(--color-yolk)"} opacity={isGauge ? 0.3 : 1} />
    </svg>
  );
}
