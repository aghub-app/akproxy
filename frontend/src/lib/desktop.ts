import * as app from "../../bindings/akproxy/app";
import type {
  AppPrefs as WireAppPrefs,
  ServiceSettings as WireServiceSettings,
} from "../../bindings/akproxy/internal/desktop/models";
import type { UpdatePrefsStatus as WireUpdatePrefsStatus } from "../../bindings/akproxy/models";
import { currentLocale, localizeErrorMessage, type Locale } from "@/lib/i18n";

export type AppPrefs = WireAppPrefs;

export type UpdatePrefsStatus = WireUpdatePrefsStatus;

export type ServiceSettings = {
  listenMode: "local" | "all" | "custom";
  customHost: string;
  port: number;
  clientApiKeys: string[];
  proxyUrl: string;
  routingStrategy: string;
  debug: boolean;
};

export type Account = {
  id: string;
  provider: string;
  label: string;
  needsReauthorization: boolean;
};

export type UsageWindow = {
  kind: string;
  used: number;
  resetsAt: string;
  durationSeconds: number;
};

export type AccountUsage = {
  id: string;
  plan: string;
  windows: UsageWindow[] | null;
  extra?: { amount: number; currency: string; kind: string } | null;
  resetCredits?: number | null;
  error: string;
};

export type Status = {
  running: boolean;
  action: "start" | "stop" | "restart";
  address: string;
  allInterfaces: boolean;
  restartRequired: boolean;
  savedAddress: string;
  error: string;
  loginActive: boolean;
};

export function rawErrorText(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  if (typeof error === "string" && error) {
    return error;
  }
  return "操作失败";
}

export function errorText(error: unknown, locale: Locale = currentLocale()): string {
  return localizeErrorMessage(rawErrorText(error), locale);
}

export const api = {
  home: () => app.Home(),
  homeModels: async () => (await app.HomeModels()) ?? [],
  status: async (): Promise<Status> => {
    const status = await app.Status();
    if (status.action !== "start" && status.action !== "stop" && status.action !== "restart") {
      throw new Error("服务返回了未知的控制动作");
    }
    return { ...status, action: status.action };
  },
  service: async () => normalizeService(await app.ServiceSettings()),
  saveService: (input: ServiceSettings) => app.SaveService(input),
  start: () => app.Start(),
  stop: () => app.Stop(),
  restart: () => app.Restart(),
  login: (provider: string) => app.Login(provider),
  reauthorizeAccount: (id: string) => app.ReauthorizeAccount(id),
  cancelLogin: () => app.CancelLogin(),
  accounts: async (page: string) => (await app.Accounts(page)) ?? [],
  accountUsage: async (page: string) => (await app.AccountUsage(page)) ?? [],
  deleteAccount: (id: string) => app.DeleteAccount(id),
  version: () => app.Version(),
  checkUpdates: () => app.CheckUpdates(),
  applyUpdate: () => app.ApplyUpdate(),
  updatePrefStatus: () => app.UpdatePrefStatus(),
  saveUpdatePrefs: (input: AppPrefs) => app.SaveUpdatePrefs(input),
  setPresentation: (language: "zh-CN" | "en", dark: boolean) => app.SetPresentation(language, dark),
  downloadPendingUpdate: () => app.DownloadPendingUpdate(),
};

function normalizeService(input: WireServiceSettings): ServiceSettings {
  return {
    ...input,
    listenMode: input.listenMode === "all" || input.listenMode === "custom" ? input.listenMode : "local",
    customHost: input.customHost ?? "",
    clientApiKeys: input.clientApiKeys ?? [],
    proxyUrl: input.proxyUrl ?? "",
    routingStrategy: input.routingStrategy || "round-robin",
  };
}
