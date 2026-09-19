import { CRITICAL_AT, WARNING_AT } from "./content";

export type Level = "normal" | "warning" | "critical";

export function levelOf(percent: number): Level {
  if (percent >= CRITICAL_AT) return "critical";
  if (percent >= WARNING_AT) return "warning";
  return "normal";
}
