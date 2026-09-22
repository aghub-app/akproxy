import * as app from "../../bindings/akproxy/app";
import type { ServiceSettings as WireServiceSettings } from "../../bindings/akproxy/internal/desktop/models";

export type ServiceSettings = {
  listenMode: "local" | "all" | "custom";
  customHost: string;
  port: number;
  clientApiKeys: string[];
  proxyUrl: string;
  routingStrategy: string;
  debug: boolean;
};

export type ModelDraft = {
  name: string;
  alias: string;
};

export type OpenAIDraft = {
  name: string;
  baseUrl: string;
  apiKeys: string[];
  models: ModelDraft[];
};

export type Account = {
  id: string;
  provider: string;
  label: string;
};

export type UsageWindow = {
  kind: string;
  used: number;
  resetsAt: string;
};

export type AccountUsage = {
  id: string;
  windows: UsageWindow[] | null;
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
};

export function errorText(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message;
  }
  if (typeof error === "string" && error) {
    return error;
  }
  return "操作失败";
}

export const api = {
  status: async (): Promise<Status> => {
    const status = await app.Status();
    if (status.action !== "start" && status.action !== "stop" && status.action !== "restart") {
      throw new Error("服务返回了未知的控制动作");
    }
    return { ...status, action: status.action };
  },
  service: async () => normalizeService(await app.ServiceSettings()),
  saveService: (input: ServiceSettings) => app.SaveService(input),
  openai: async (): Promise<OpenAIDraft[]> => ((await app.OpenAIProviders()) ?? []).map(
    (row) => ({ ...row, apiKeys: row.apiKeys ?? [], models: row.models ?? [] }),
  ),
  saveOpenAI: (drafts: OpenAIDraft[]) => app.SaveOpenAI(drafts),
  start: () => app.Start(),
  stop: () => app.Stop(),
  restart: () => app.Restart(),
  login: (provider: string) => app.Login(provider),
  cancelLogin: () => app.CancelLogin(),
  accounts: async (page: string) => (await app.Accounts(page)) ?? [],
  accountUsage: async (page: string) => (await app.AccountUsage(page)) ?? [],
  deleteAccount: (id: string) => app.DeleteAccount(id),
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
