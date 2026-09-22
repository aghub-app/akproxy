import { createHashRouter, Navigate } from "react-router";
import {
  DevinPage,
  KimiPage,
  NativeProviderPage,
  OpenAIPage,
  ServicePage,
} from "@/features/pages";
import { AppLayout } from "@/app/shell";

function CodexRoute() {
  return <NativeProviderPage provider="codex" loginLabel="添加 Codex 账号" />;
}

function GrokRoute() {
  return <NativeProviderPage provider="grok" loginLabel="添加 Grok 账号" />;
}

function ClaudeRoute() {
  return <NativeProviderPage provider="claude" loginLabel="添加 Claude 账号" />;
}

function GeminiRoute() {
  return <NativeProviderPage provider="gemini" loginLabel="添加 Gemini 账号" />;
}

export const router = createHashRouter([
  {
    path: "/",
    element: <AppLayout />,
    children: [
      { index: true, element: <Navigate to="/settings" replace /> },
      { path: "service", element: <Navigate to="/settings" replace /> },
      { path: "settings", element: <ServicePage /> },
      { path: "codex", element: <CodexRoute /> },
      { path: "grok", element: <GrokRoute /> },
      { path: "claude", element: <ClaudeRoute /> },
      { path: "gemini", element: <GeminiRoute /> },
      { path: "kimi", element: <KimiPage /> },
      { path: "devin", element: <DevinPage /> },
      { path: "openai", element: <OpenAIPage /> },
      { path: "yaml", element: <Navigate to="/settings" replace /> },
    ],
  },
]);
