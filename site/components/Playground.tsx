"use client";

import { useState } from "react";
import type { Content, Mode } from "@/lib/content";
import { MODES } from "@/lib/content";
import { levelOf } from "@/lib/level";
import { GaugeGlyph } from "./GaugeGlyph";
import { Strip } from "./Strip";

interface PlaygroundProps {
  text: Content["playground"];
}

// 見本の値。週枠だけを訪問者が動かす
const SAMPLE = { model: "Opus5", effort: "xhigh", fiveHour: 42, perModel: 55, perModelLetter: "F" } as const;
const INITIAL_WEEKLY = 76;

/** 実寸のバーとメニューバーのメーターを、週枠の使用率を動かしながら確かめる場所 */
export function Playground({ text }: PlaygroundProps) {
  const [weekly, setWeekly] = useState<number>(INITIAL_WEEKLY);
  const [mode, setMode] = useState<Mode>("full");

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_20rem]">
      <div className="desk relative min-h-[19rem] overflow-hidden rounded-xl border border-white/10">
        <div className="flex h-7 items-center justify-end gap-3 border-b border-white/10 bg-black/35 px-3 strip-font text-[13px] text-fog">
          <span className="sr-only">{text.menuBarLabel}</span>
          <span className="flex items-center gap-[3px]" aria-hidden="true">
            <GaugeGlyph width={24} thickness={8} percent={weekly} level={levelOf(weekly)} />
            <span>{weekly}</span>
          </span>
          <span className="text-fog/60" aria-hidden="true">
            Sat 9:41
          </span>
        </div>
        <div className="absolute left-1/2 top-16 -translate-x-1/2">
          <Strip mode={mode} weekly={weekly} {...SAMPLE} />
        </div>
        <p className="absolute bottom-3 left-4 text-xs text-fog/60">
          {text.stripLabel} · {text.menuBarLabel}
        </p>
      </div>

      <div className="flex flex-col gap-6">
        <label className="flex flex-col gap-2 text-sm text-fog">
          <span className="flex items-baseline justify-between">
            <span>{text.sliderLabel}</span>
            <output className="font-mono text-base text-paper">{weekly}%</output>
          </span>
          <input
            type="range"
            min={0}
            max={100}
            value={weekly}
            onChange={(event) => setWeekly(Number(event.target.value))}
            className="slider"
          />
        </label>

        <fieldset className="flex flex-col gap-2 text-sm text-fog">
          <legend className="mb-2">{text.modeLabel}</legend>
          <div className="flex flex-wrap gap-2">
            {MODES.map((value) => (
              <button
                key={value}
                type="button"
                aria-pressed={mode === value}
                onClick={() => setMode(value)}
                className={`rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                  mode === value ? "border-yolk bg-yolk text-ink" : "border-white/15 text-fog hover:border-white/40"
                }`}
              >
                {text.modes[value]}
              </button>
            ))}
          </div>
        </fieldset>

        <ul className="flex flex-col gap-1.5 text-sm text-fog">
          <li className="flex items-center gap-2">
            <span className="h-1.5 w-1.5 rounded-full bg-ember" />
            {text.legend.dot}
          </li>
          <li>{text.legend.s}</li>
          <li>{text.legend.w}</li>
          <li>{text.legend.f}</li>
          <li className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-yolk" />
            {text.legend.warning}
          </li>
          <li className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-brick" />
            {text.legend.critical}
          </li>
        </ul>
      </div>
    </div>
  );
}
