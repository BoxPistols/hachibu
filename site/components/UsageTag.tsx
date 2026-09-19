import { levelOf } from "@/lib/level";

interface UsageTagProps {
  letter: string;
  percent: number;
}

/** 使用率の語。黄や赤の段階にあるときだけ札で囲む（アプリと同じ規則、角丸は3） */
export function UsageTag({ letter, percent }: UsageTagProps) {
  const level = levelOf(percent);
  const text = `${letter}${percent}`;
  if (level === "normal") return <span>{text}</span>;
  return (
    <span
      className={
        level === "warning"
          ? "rounded-[3px] bg-yolk px-[3px] py-px text-ink"
          : "rounded-[3px] bg-brick px-[3px] py-px text-white"
      }
    >
      {text}
    </span>
  );
}
