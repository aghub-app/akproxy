import ClaudeColor from "@lobehub/icons/es/Claude/components/Color";
import CodexColor from "@lobehub/icons/es/Codex/components/Color";
import DevinColor from "@lobehub/icons/es/Devin/components/Color";
import GeminiColor from "@lobehub/icons/es/Gemini/components/Color";
import GrokMono from "@lobehub/icons/es/Grok/components/Mono";
import KimiMono from "@lobehub/icons/es/Kimi/components/Mono";
import type { FC } from "react";

export const providerPages: { to: string; label: string; icon: FC<{ size?: number }> }[] = [
  { to: "/codex", label: "Codex", icon: CodexColor },
  { to: "/grok", label: "Grok", icon: GrokMono },
  { to: "/claude", label: "Claude", icon: ClaudeColor },
  { to: "/gemini", label: "Gemini", icon: GeminiColor },
  { to: "/kimi", label: "Kimi", icon: KimiMono },
  { to: "/devin", label: "Devin", icon: DevinColor },
];
