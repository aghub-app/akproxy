import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useEffect, useRef, useState } from "react";
import { ClientKeyDisplay } from "@/components/client-key-display";
import { useSearchParams } from "react-router";
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
import { AboutTab } from "@/features/about";
import { api, errorText, type AppPrefs, type ServiceSettings } from "@/lib/desktop";

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

const AboutTabContainer: FC = () => {
  const version = useQuery({ queryKey: ["version"], queryFn: api.version });
  return <AboutTab version={version.data} />;
};

const UsagePrefField: FC = () => {
  const client = useQueryClient();
  const status = useQuery({ queryKey: ["update-prefs"], queryFn: api.updatePrefStatus });
  const save = useMutation({
    mutationFn: (prefs: AppPrefs) => api.saveUpdatePrefs(prefs),
    onSuccess: (next) => {
      client.setQueryData(["update-prefs"], next);
      void client.invalidateQueries({ queryKey: ["account-usage"] });
    },
    onError: notifyError,
  });
  if (status.isError) {
    return (
      <div role="alert" className="flex items-center gap-3 text-sm">
        <span>用量设置暂时读不到</span>
        <Button variant="outline" onClick={() => void status.refetch()}>重试</Button>
      </div>
    );
  }
  if (!status.data) {
    return <p className="text-muted-foreground text-sm">正在读取用量设置…</p>;
  }
  const prefs = status.data.prefs;
  return (
    <div className="flex max-w-xl flex-col gap-6">
      <Field>
      <div className="flex items-center gap-3">
        <Switch
          disabled={save.isPending}
          checked={prefs.usageEnabled}
          onCheckedChange={(checked) =>
            save.mutate({ ...prefs, usageEnabled: checked })
          }
        />
        <FieldLabel>额度显示</FieldLabel>
      </div>
      <FieldDescription>关闭后停止读取厂商额度。</FieldDescription>
      </Field>
      <Field>
        <FieldLabel>百分比显示</FieldLabel>
        <Select disabled={save.isPending} value={prefs.usagePercentMode} onValueChange={(value) => {
          if (value === "left" || value === "used") save.mutate({ ...prefs, usagePercentMode: value });
        }}>
          <SelectTrigger><SelectValue>{(value) => value === "used" ? "已用" : "剩余"}</SelectValue></SelectTrigger>
          <SelectPopup>
            <SelectItem value="left">剩余</SelectItem>
            <SelectItem value="used">已用</SelectItem>
          </SelectPopup>
        </Select>
      </Field>
      <Field>
        <FieldLabel>重置时间</FieldLabel>
        <Select disabled={save.isPending} value={prefs.usageResetMode} onValueChange={(value) => {
          if (value === "countdown" || value === "exact") save.mutate({ ...prefs, usageResetMode: value });
        }}>
          <SelectTrigger><SelectValue>{(value) => value === "exact" ? "具体时间" : "倒计时"}</SelectValue></SelectTrigger>
          <SelectPopup>
            <SelectItem value="countdown">倒计时</SelectItem>
            <SelectItem value="exact">具体时间</SelectItem>
          </SelectPopup>
        </Select>
      </Field>
      <Field>
        <div className="flex items-center gap-3">
          <Switch disabled={save.isPending} checked={prefs.usageAlwaysShowPacing} onCheckedChange={(checked) => save.mutate({ ...prefs, usageAlwaysShowPacing: checked })} />
          <FieldLabel>始终显示使用节奏</FieldLabel>
        </div>
        <FieldDescription>有足够窗口数据后，显示按当前速度推算的剩余额度；关闭时只提醒接近限额的窗口。</FieldDescription>
      </Field>
      <Field>
        <div className="flex items-center gap-3">
          <Switch disabled={save.isPending} checked={prefs.usageShowExtra} onCheckedChange={(checked) => save.mutate({ ...prefs, usageShowExtra: checked })} />
          <FieldLabel>显示额外额度</FieldLabel>
        </div>
        <FieldDescription>仅在厂商明确返回金额或积分时显示。</FieldDescription>
      </Field>
      <Field>
        <div className="flex items-center gap-3">
          <Switch disabled={save.isPending} checked={prefs.usageShowResets} onCheckedChange={(checked) => save.mutate({ ...prefs, usageShowResets: checked })} />
          <FieldLabel>显示重置次数</FieldLabel>
        </div>
        <FieldDescription>Codex 返回可用重置券数量时显示。</FieldDescription>
      </Field>
    </div>
  );
};

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
  if (!query.data) {
    return <p className="text-muted-foreground text-sm">正在读取设置…</p>;
  }
  return (
    <div className="flex flex-col gap-6">
      <ServiceForm key={JSON.stringify(query.data)} initial={query.data} />
    </div>
  );
};

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
                  <ClientKeyDisplay value={key} visible={!!visible[key]} onToggle={() => setVisible((current) => ({ ...current, [key]: !current[key] }))} />
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
  const writes = useRef(Promise.resolve());
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get("tab");
  const tab = tabParam === "clients" || tabParam === "advanced" || tabParam === "usage" || tabParam === "about" ? tabParam : "listen";

  function write(snapshot: ServiceSettings) {
    if (!canPersistService(snapshot)) {
      return;
    }
    const id = ++seq.current;
    writes.current = writes.current
      .then(() => api.saveService(snapshot))
      .then(async () => {
        if (id === seq.current) {
          await client.invalidateQueries({ queryKey: ["status"] });
        }
      })
      .catch(notifyError);
  }

  useEffect(() => () => {
    if (timer.current !== null) {
      window.clearTimeout(timer.current);
      timer.current = null;
      write(latest.current);
    }
  }, []);
  function persist(next: ServiceSettings, immediate: boolean) {
    latest.current = next;
    if (timer.current !== null) {
      window.clearTimeout(timer.current);
      timer.current = null;
    }
    const run = () => {
      timer.current = null;
      write(latest.current);
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
      <Tabs
        className="gap-6"
        value={tab}
        onValueChange={(value) => setSearchParams(value ? { tab: String(value) } : {})}
      >
        <TabsList>
          <TabsTab value="listen">监听</TabsTab>
          <TabsTab value="clients">密钥</TabsTab>
          <TabsTab value="advanced">高级</TabsTab>
          <TabsTab value="usage">用量</TabsTab>
          <TabsTab value="about">关于</TabsTab>
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
        <TabsPanel value="usage">
          <UsagePrefField />
        </TabsPanel>
        <TabsPanel value="about">
          <AboutTabContainer />
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
