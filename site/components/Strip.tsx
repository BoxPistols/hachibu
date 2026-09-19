import type { Mode } from "@/lib/content";
import { levelOf } from "@/lib/level";
import { UsageTag } from "./UsageTag";

interface StripProps {
  mode: Mode;
  model: string;
  effort: string;
  fiveHour: number;
  weekly: number;
  perModel: number;
  perModelLetter: string;
}

const RING: Record<string, string> = {
  normal: "border-transparent",
  warning: "border-yolk",
  critical: "border-brick",
};

function Grip() {
  return (
    <span className="grid grid-cols-2 gap-[3px]" aria-hidden="true">
      {Array.from({ length: 6 }, (_, i) => (
        <span key={i} className="h-[2px] w-[2px] rounded-full bg-fog/50" />
      ))}
    </span>
  );
}

/** アプリのバーを実寸（高さ28px、文字13px）で描く */
export function Strip({ mode, model, effort, fiveHour, weekly, perModel, perModelLetter }: StripProps) {
  const worst = levelOf(Math.max(fiveHour, weekly, perModel));
  if (mode === "tab") {
    return (
      <div className="glass flex h-7 w-5 items-center justify-center rounded-lg">
        <span className={`flex h-[14px] w-[14px] items-center justify-center rounded-full border-2 ${RING[worst]}`}>
          <span className="h-2 w-2 rounded-full bg-fog/70" />
        </span>
      </div>
    );
  }
  return (
    <div className="glass inline-flex h-7 items-center gap-1.5 rounded-lg pl-[7px] pr-[9px] strip-font text-[13px] font-medium text-fog">
      <Grip />
      {mode === "full" && (
        <span className="flex items-start">
          <span>{model}</span>
          <span className="mx-[2px] mt-[3px] h-1 w-1 rounded-full bg-ember" />
          <span>&nbsp;{effort}&nbsp;·</span>
        </span>
      )}
      <span className="flex items-center gap-[5px]">
        <UsageTag letter="S" percent={fiveHour} />
        <UsageTag letter="W" percent={weekly} />
        <UsageTag letter={perModelLetter} percent={perModel} />
      </span>
    </div>
  );
}
