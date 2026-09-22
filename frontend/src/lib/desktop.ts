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
  kind: "5h" | "weekly" | "weekly_opus" | "weekly_sonnet";
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

type DesktopApp = {
  Status(): Promise<Status>;
  ServiceSettings(): Promise<ServiceSettings>;
  SaveService(input: ServiceSettings): Promise<void>;
  OpenAIProviders(): Promise<OpenAIDraft[] | null>;
  SaveOpenAI(drafts: OpenAIDraft[]): Promise<void>;
  Start(): Promise<void>;
  Stop(): Promise<void>;
  Restart(): Promise<void>;
  Login(provider: string): Promise<void>;
  CancelLogin(): Promise<void>;
  Accounts(page: string): Promise<Account[] | null>;
  AccountUsage(page: string): Promise<AccountUsage[] | null>;
  DeleteAccount(id: string): Promise<void>;
};

function desktop(): DesktopApp {
  const app = window.go?.main?.App;
  if (!app) {
    throw new Error("桌面接口还没有准备好");
  }
  return app;
}

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
  status: () => desktop().Status(),
  service: async () => normalizeService(await desktop().ServiceSettings()),
  saveService: (input: ServiceSettings) => desktop().SaveService(input),
  openai: async () => (await desktop().OpenAIProviders()) ?? [],
  saveOpenAI: (drafts: OpenAIDraft[]) => desktop().SaveOpenAI(drafts),
  start: () => desktop().Start(),
  stop: () => desktop().Stop(),
  restart: () => desktop().Restart(),
  login: (provider: string) => desktop().Login(provider),
  cancelLogin: () => desktop().CancelLogin(),
  accounts: async (page: string) => (await desktop().Accounts(page)) ?? [],
  accountUsage: async (page: string) => (await desktop().AccountUsage(page)) ?? [],
  deleteAccount: (id: string) => desktop().DeleteAccount(id),
};

function normalizeService(input: ServiceSettings): ServiceSettings {
  return {
    ...input,
    listenMode: input.listenMode || "local",
    customHost: input.customHost ?? "",
    clientApiKeys: input.clientApiKeys ?? [],
    proxyUrl: input.proxyUrl ?? "",
    routingStrategy: input.routingStrategy || "round-robin",
  };
}

declare global {
  interface Window {
    go?: {
      main?: {
        App?: DesktopApp;
      };
    };
    runtime?: unknown;
  }
}
