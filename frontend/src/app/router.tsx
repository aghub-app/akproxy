import { createHashRouter, Navigate } from "react-router";
import {
  DevinPage,
  KimiPage,
  NativeProviderPage,
  ServicePage,
} from "@/features/pages";
import { HomePage } from "@/features/home";
import { AppLayout } from "@/app/shell";
import { usePresentation } from "@/presentation";

function CodexRoute() {
  const { t } = usePresentation();
  return <NativeProviderPage provider="codex" loginLabel={t("添加 Codex 账号")} />;
}

function GrokRoute() {
  const { t } = usePresentation();
  return <NativeProviderPage provider="grok" loginLabel={t("添加 Grok 账号")} />;
}

function ClaudeRoute() {
  const { t } = usePresentation();
  return <NativeProviderPage provider="claude" loginLabel={t("添加 Claude 账号")} />;
}

function GeminiRoute() {
  const { t } = usePresentation();
  return <NativeProviderPage provider="gemini" loginLabel={t("添加 Gemini 账号")} />;
}

export const router = createHashRouter([
  {
    path: "/",
    element: <AppLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: "service", element: <Navigate to="/settings" replace /> },
      { path: "settings", element: <ServicePage /> },
      { path: "codex", element: <CodexRoute /> },
      { path: "grok", element: <GrokRoute /> },
      { path: "claude", element: <ClaudeRoute /> },
      { path: "gemini", element: <GeminiRoute /> },
      { path: "kimi", element: <KimiPage /> },
      { path: "devin", element: <DevinPage /> },
      { path: "openai", element: <Navigate to="/" replace /> },
      { path: "yaml", element: <Navigate to="/settings" replace /> },
    ],
  },
]);
