import type { UsageWindow } from "@/lib/desktop";

export function windowPace(window: UsageWindow, alwaysShow: boolean, now = Date.now()) {
  if (window.used >= 100) return { tone: "danger", label: "额度已用尽", tick: null };
  const fallbackSeconds = window.kind === "5h" ? 5 * 3600
    : window.kind === "daily" ? 24 * 3600
    : window.kind === "monthly" ? 30 * 24 * 3600
    : 7 * 24 * 3600;
  const duration = (window.durationSeconds > 0 ? window.durationSeconds : fallbackSeconds) * 1000;
  const remaining = new Date(window.resetsAt).getTime() - now;
  const elapsed = duration - remaining;
  if (!window.resetsAt || window.used <= 0 || !Number.isFinite(elapsed) || elapsed < duration * 0.1 || remaining <= 0) {
    return { tone: window.used >= 90 ? "danger" : window.used >= 80 ? "warning" : "normal", label: "", tick: null };
  }
  const projectedUsed = window.used * duration / elapsed;
  const tone = projectedUsed >= 100 ? "danger" : projectedUsed >= 90 ? "warning" : "normal";
  const label = tone === "danger" ? "按当前速度可能提前用尽"
    : tone === "warning" ? `按当前速度，重置时约剩余 ${Math.max(1, Math.round(100 - projectedUsed))}%`
    : alwaysShow ? `按当前速度，重置时约剩余 ${Math.round(100 - projectedUsed)}%` : "";
  return { tone, label, tick: Math.min(98, Math.max(2, elapsed / duration * 100)) };
}
