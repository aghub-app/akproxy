import assert from "node:assert/strict";
import test from "node:test";
import { windowPace } from "./usage-pace.ts";

const now = Date.parse("2026-09-23T12:00:00Z");

test("monthly pacing uses the vendor duration", () => {
  const pace = windowPace({ kind: "monthly", used: 30, resetsAt: new Date(now + 15 * 86400_000).toISOString(), durationSeconds: 30 * 86400 }, true, now);
  assert.equal(pace.tone, "normal");
  assert.equal(pace.label, "按当前速度，重置时约剩余 40%");
  assert.equal(pace.tick, 50);
});

test("a fast weekly burn warns before the bar is nearly empty", () => {
  const pace = windowPace({ kind: "weekly", used: 20, resetsAt: new Date(now + 6 * 86400_000).toISOString(), durationSeconds: 7 * 86400 }, false, now);
  assert.equal(pace.tone, "danger");
  assert.equal(pace.label, "按当前速度可能提前用尽");
});

test("no reset estimate falls back to the remaining level", () => {
  const pace = windowPace({ kind: "weekly", used: 85, resetsAt: "", durationSeconds: 0 }, false, now);
  assert.equal(pace.tone, "warning");
  assert.equal(pace.tick, null);
});
