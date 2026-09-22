import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { EyeIcon, EyeSlashIcon } from "@phosphor-icons/react";
import { type FC, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import { Frame } from "@/components/ui/frame";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { toastManager } from "@/components/ui/toast";
import { AccountsPanel } from "@/features/accounts";
import { beginManualUpdateCheck, finishManualUpdateCheck } from "@/features/update-dialog";
import { api, errorText, type ServiceSettings } from "@/lib/desktop";

const strategies = [
  { value: "round-robin", label: "轮询" },
  { value: "weighted-round-robin", label: "加权轮询" },
  { value: "fill-first", label: "填满优先" },
];

const listenLabels: Record<string, string> = {
  local: "只本机",
  all: "所有网络接口",
  custom: "自定义地址",
};

const strategyLabels = Object.fromEntries(strategies.map((item) => [item.value, item.label]));

function notifyError(error: unknown) {
  toastManager.add({ title: errorText(error), type: "error" });
}

function canPersistService(next: ServiceSettings) {
  if (next.port < 1 || next.port > 65535) {
    return false;
  }
  if (next.listenMode === "custom" && next.customHost.trim() === "") {
    return false;
  }
  return next.clientApiKeys.some((key) => key.trim() !== "");
}

export const ServicePage: FC = () => {
  const query = useQuery({ queryKey: ["service"], queryFn: api.service });
  const version = useQuery({ queryKey: ["version"], queryFn: api.version });
  const update = useMutation({
    mutationFn: api.checkUpdates,
    onError: (error) => {
      finishManualUpdateCheck();
      notifyError(error);
    },
  });
  if (!query.data) {
    return <p className="text-muted-foreground text-sm">正在读取设置…</p>;
  }
  return (
    <div className="flex flex-col gap-6">
      <ServiceForm key={JSON.stringify(query.data)} initial={query.data} />
      <div className="flex items-center justify-between border-t pt-4">
        <span className="text-sm text-muted-foreground">akproxy {version.data}</span>
        <Button
          variant="outline"
          loading={update.isPending}
          onClick={() => {
            beginManualUpdateCheck();
            update.mutate();
          }}
        >
          检查更新
        </Button>
      </div>
    </div>
  );
};

function maskClientKey(key: string): string {
  if (key.length <= 8) {
    return "•".repeat(Math.max(key.length, 8));
  }
  return `${key.slice(0, 6)}${"•".repeat(12)}`;
}

function createClientKey(): string {
  const bytes = new Uint8Array(24);
  crypto.getRandomValues(bytes);
  return `sk-${[...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

const ClientKeysTable: FC<{
  keys: string[];
  onChange: (keys: string[]) => void;
}> = ({ keys, onChange }) => {
  const [visible, setVisible] = useState<Record<string, boolean>>({});
  const [flashed, setFlashed] = useState<string | null>(null);
  const rows = keys.map((key) => key.trim()).filter((key) => key !== "");

  function createKey() {
    let key = createClientKey();
    while (rows.includes(key)) {
      key = createClientKey();
    }
    onChange([...rows, key]);
    setFlashed(key);
    toastManager.add({ title: "创建成功", type: "success" });
  }

  return (
    <Frame className="w-full">
      <Table variant="card">
        <TableHeader>
          <TableRow>
            <TableHead>API key</TableHead>
            <TableHead>
              <div className="flex justify-end">
                <Button type="button" size="sm" onClick={createKey}>
                  新建 API key
                </Button>
              </div>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell className="text-muted-foreground" colSpan={2}>
                还没有客户端密钥。至少保留一把，否则不能启动。
              </TableCell>
            </TableRow>
          ) : (
            rows.map((key) => (
              <TableRow
                key={key}
                className={key === flashed ? "animate-[client-key-flash_900ms_ease-out]" : undefined}
                onAnimationEnd={() => {
                  if (key === flashed) {
                    setFlashed(null);
                  }
                }}
              >
                <TableCell className="max-w-md font-mono">
                  <div className="flex items-center gap-2">
                    <Button
                      size="icon-sm"
                      type="button"
                      variant="ghost"
                      onClick={() => setVisible((current) => ({ ...current, [key]: !current[key] }))}
                    >
                      {visible[key] ? <EyeIcon /> : <EyeSlashIcon />}
                    </Button>
                    <span className="truncate">{visible[key] ? key : maskClientKey(key)}</span>
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      void navigator.clipboard.writeText(key).then(() => {
                        toastManager.add({ title: "已复制 API key", type: "success" });
                      });
                    }}
                  >
                    复制
                  </Button>
                  <Button
                    className="ms-2"
                    type="button"
                    variant="destructive-outline"
                    onClick={() => onChange(rows.filter((item) => item !== key))}
                  >
                    删除
                  </Button>
                </TableCell>
              </TableRow>
            ))
          )}
          </TableBody>
      </Table>
    </Frame>
  );
};

const ServiceForm: FC<{ initial: ServiceSettings }> = ({ initial }) => {
  const client = useQueryClient();
  const [form, setForm] = useState(initial);
  const latest = useRef(initial);
  const seq = useRef(0);
  const timer = useRef<number | null>(null);

  function persist(next: ServiceSettings, immediate: boolean) {
    latest.current = next;
    if (timer.current !== null) {
      window.clearTimeout(timer.current);
      timer.current = null;
    }
    const run = () => {
      timer.current = null;
      const snapshot = latest.current;
      if (!canPersistService(snapshot)) {
        return;
      }
      const id = ++seq.current;
      void api
        .saveService(snapshot)
        .then(async () => {
          if (id !== seq.current) {
            return;
          }
          await client.invalidateQueries({ queryKey: ["status"] });
        })
        .catch((error: unknown) => {
          if (id === seq.current) {
            notifyError(error);
          }
        });
    };
    if (immediate) {
      run();
      return;
    }
    timer.current = window.setTimeout(run, 300);
  }

  function edit(patch: Partial<ServiceSettings>, immediate = false) {
    const next = { ...latest.current, ...patch };
    if (patch.clientApiKeys && !next.clientApiKeys.some((key) => key.trim() !== "")) {
      toastManager.add({ title: "至少保留一把客户端密钥", type: "error" });
      return;
    }
    latest.current = next;
    setForm(next);
    persist(next, immediate);
  }
  return (
    <div className="flex flex-col gap-5">
      <Tabs className="gap-6" defaultValue="listen">
        <TabsList>
          <TabsTab value="listen">监听</TabsTab>
          <TabsTab value="clients">密钥</TabsTab>
          <TabsTab value="advanced">高级</TabsTab>
        </TabsList>
        <TabsPanel value="listen" className="flex flex-col gap-5">
          <Field>
            <FieldLabel>监听范围</FieldLabel>
            <Select
              value={form.listenMode}
              onValueChange={(value) => {
                if (value === "local" || value === "all" || value === "custom") {
                  edit({ listenMode: value }, true);
                }
              }}
            >
              <SelectTrigger>
                <SelectValue>
                  {(value) => (typeof value === "string" ? listenLabels[value] : null)}
                </SelectValue>
              </SelectTrigger>
              <SelectPopup>
                <SelectItem value="local">只本机</SelectItem>
                <SelectItem value="all">所有网络接口</SelectItem>
                <SelectItem value="custom">自定义地址</SelectItem>
              </SelectPopup>
            </Select>
            {form.listenMode === "custom" ? (
              <Input
                value={form.customHost}
                placeholder="192.168.1.8"
                onChange={(event) => edit({ customHost: event.target.value })}
              />
            ) : null}
            <FieldDescription>
              只本机会写成 127.0.0.1。所有网络接口意味着同一网络上的其他设备也能连接。
            </FieldDescription>
          </Field>
          <Field>
            <FieldLabel>端口</FieldLabel>
            <Input
              type="number"
              min={1}
              max={65535}
              step={1}
              value={form.port || ""}
              onChange={(event) => {
                const raw = event.target.value;
                edit({ port: raw === "" ? 0 : Number(raw) });
              }}
            />
          </Field>
        </TabsPanel>
        <TabsPanel value="clients">
          <ClientKeysTable
            keys={form.clientApiKeys}
            onChange={(clientApiKeys) => edit({ clientApiKeys }, true)}
          />
        </TabsPanel>
        <TabsPanel value="advanced" className="flex flex-col gap-5">
          <Field>
            <FieldLabel>出站代理</FieldLabel>
            <Input
              value={form.proxyUrl}
              placeholder="http://127.0.0.1:7890"
              onChange={(event) => edit({ proxyUrl: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel>路由</FieldLabel>
            <Select
              value={form.routingStrategy}
              onValueChange={(value) => {
                if (typeof value === "string") {
                  edit({ routingStrategy: value }, true);
                }
              }}
            >
              <SelectTrigger>
                <SelectValue>
                  {(value) => (typeof value === "string" ? strategyLabels[value] : null)}
                </SelectValue>
              </SelectTrigger>
              <SelectPopup>
                {strategies.map((item) => (
                  <SelectItem key={item.value} value={item.value}>
                    {item.label}
                  </SelectItem>
                ))}
              </SelectPopup>
            </Select>
          </Field>
          <Field>
            <div className="flex items-center gap-3">
              <Switch checked={form.debug} onCheckedChange={(checked) => edit({ debug: checked }, true)} />
              <FieldLabel>调试日志</FieldLabel>
            </div>
          </Field>
        </TabsPanel>
      </Tabs>
    </div>
  );
};

export const NativeProviderPage: FC<{
  provider: "codex" | "grok" | "claude" | "gemini";
  loginLabel: string;
}> = ({ provider, loginLabel }) => {
  return <AccountsPanel page={provider} logins={[{ id: provider, label: loginLabel }]} />;
};

export const KimiPage: FC = () => {
  return (
    <AccountsPanel
      page="kimi"
      logins={[
        { id: "kimi", label: "添加 Kimi 中文站账号" },
        { id: "kimi-ai", label: "添加 Kimi 国际站账号" },
      ]}
    />
  );
};

export const DevinPage: FC = () => {
  return (
    <AccountsPanel
      page="devin"
      logins={[{ id: "devin", label: "添加 Devin 账号" }]}
      filledExtra={<p className="text-muted-foreground text-sm">登录会直接写入账号目录。</p>}
    />
  );
};
