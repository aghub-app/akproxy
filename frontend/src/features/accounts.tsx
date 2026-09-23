import { ArrowsClockwiseIcon, DotsThreeIcon, TrashIcon } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, type ReactNode, useState } from "react";
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardAction, CardHeader, CardTitle } from "@/components/ui/card";
import { Menu, MenuItem, MenuPopup, MenuTrigger } from "@/components/ui/menu";
import { Progress, ProgressIndicator, ProgressTrack } from "@/components/ui/progress";
import { toastManager } from "@/components/ui/toast";
import { ProviderEmpty } from "@/features/provider-empty";
import { windowPace } from "@/features/usage-pace";
import { api, errorText, type Account, type AccountUsage, type AppPrefs, type UsageWindow } from "@/lib/desktop";

const windowLabel: Record<UsageWindow["kind"], string> = {
  "5h": "5 小时",
  daily: "每日",
  weekly: "每周",
  monthly: "每月",
  weekly_opus: "每周 Opus",
  weekly_sonnet: "每周 Sonnet",
};

const sessionLimitProviders = new Set(["codex", "claude", "xai", "devin"]);

type UsageOptions = Pick<AppPrefs, "usagePercentMode" | "usageResetMode" | "usageShowExtra" | "usageShowResets" | "usageAlwaysShowPacing">;

const defaultUsageOptions: UsageOptions = {
  usagePercentMode: "left",
  usageResetMode: "countdown",
  usageShowExtra: true,
  usageShowResets: true,
  usageAlwaysShowPacing: false,
};

function resetLabel(value: string, mode: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  if (mode === "countdown") {
    const minutes = Math.max(0, Math.ceil((date.getTime() - Date.now()) / 60_000));
    if (minutes === 0) return "即将重置";
    const days = Math.floor(minutes / 1440);
    const hours = Math.floor((minutes % 1440) / 60);
    const rest = minutes % 60;
    return `约 ${days ? `${days} 天 ` : ""}${hours ? `${hours} 小时 ` : ""}${!days && rest ? `${rest} 分钟` : ""}后重置`;
  }
  const sameDay = date.toDateString() === new Date().toDateString();
  const time = new Intl.DateTimeFormat("zh-CN", { hour: "2-digit", minute: "2-digit" }).format(date);
  if (sameDay) {
    return `${time} 重置`;
  }
  const day = new Intl.DateTimeFormat("zh-CN", { month: "numeric", day: "numeric" }).format(date);
  return `${day} ${time} 重置`;
}

function extraLabel(extra: NonNullable<AccountUsage["extra"]>) {
  if (extra.kind === "disabled") return "未启用";
  const amount = extra.currency === "credits"
    ? `${extra.amount.toLocaleString("zh-CN", { maximumFractionDigits: 1 })} 积分`
    : `${extra.currency === "USD" ? "$" : `${extra.currency} `}${extra.amount.toFixed(2)}`;
  if (extra.kind === "cap") return `${amount} 上限`;
  return `${amount} ${extra.kind === "used" ? "已用" : "余额"}`;
}

const UsageBars: FC<{ usage?: AccountUsage; loading: boolean; failed: boolean; options: UsageOptions }> = ({
  usage,
  loading,
  failed,
  options,
}) => {
  if (!usage) {
    if (loading) {
      return <p className="px-4 pb-4 text-muted-foreground text-xs">正在读取额度…</p>;
    }
    if (failed) {
      return <p className="px-4 pb-4 text-muted-foreground text-xs">额度暂时读不到</p>;
    }
    return null;
  }
  if (usage.error && !(usage.windows && usage.windows.length > 0) && !usage.extra && usage.resetCredits == null) {
    return <p className="px-4 pb-4 text-muted-foreground text-xs">{usage.error}</p>;
  }
  return (
    <div className="flex flex-1 flex-col gap-4 px-4 pb-4">
      {usage.error ? <p className="text-xs text-muted-foreground">{usage.error}</p> : null}
      {!usage.error && !usage.windows?.length ? <p className="text-xs text-muted-foreground">暂无额度窗口</p> : null}
      {(usage.windows ?? []).map((window) => {
        const pace = windowPace(window, options.usageAlwaysShowPacing);
        const label = windowLabel[window.kind] ?? window.kind;
        return (
          <div className="flex flex-col gap-2" key={window.kind}>
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-medium">{label}</span>
              {pace.label ? <span className={`text-right text-xs ${pace.tone === "danger" ? "text-red-600 dark:text-red-400" : "text-muted-foreground"}`}>{pace.label}</span> : null}
            </div>
            <Progress aria-label={`${label}额度`} value={options.usagePercentMode === "used" ? window.used : 100 - window.used}>
              <ProgressTrack className="relative">
                <ProgressIndicator className={pace.tone === "danger" ? "bg-red-500" : pace.tone === "warning" ? "bg-amber-500" : "bg-blue-500"} />
                {pace.tick != null && (options.usageAlwaysShowPacing || pace.tone !== "normal") ? (
                  <span aria-hidden="true" className="absolute inset-y-0 w-0.5 bg-foreground/70" style={{ left: `${options.usagePercentMode === "used" ? pace.tick : 100 - pace.tick}%` }} />
                ) : null}
              </ProgressTrack>
            </Progress>
            <div className="flex items-center justify-between gap-2 text-xs tabular-nums">
              <span>{Math.round(options.usagePercentMode === "used" ? window.used : 100 - window.used)}% {options.usagePercentMode === "used" ? "已用" : "剩余"}</span>
              {window.resetsAt ? <span className="text-right text-muted-foreground">{resetLabel(window.resetsAt, options.usageResetMode)}</span> : null}
            </div>
          </div>
        );
      })}
      {options.usageShowExtra && usage.extra ? (
        <div className="flex items-center justify-between gap-2 text-sm">
          <span>{usage.extra.kind === "cap" || usage.extra.kind === "disabled" ? "按量付费" : "额外额度"}</span>
          <span className="text-right tabular-nums text-muted-foreground">{extraLabel(usage.extra)}</span>
        </div>
      ) : null}
      {options.usageShowResets && usage.resetCredits != null ? (
        <div className="flex items-center justify-between gap-2 text-sm">
          <span>额度重置次数</span>
          <span className="tabular-nums text-muted-foreground">{usage.resetCredits} 次可用</span>
        </div>
      ) : null}
    </div>
  );
};

const Badge: FC<{ children: ReactNode }> = ({ children }) => (
  <span className="rounded-sm bg-muted px-1.5 py-0.5 text-muted-foreground text-xs leading-none">
    {children}
  </span>
);

export const AccountsPanel: FC<{
  page: string;
  logins: { id: string; label: string }[];
  filledExtra?: ReactNode;
}> = ({ page, logins, filledExtra }) => {
  const client = useQueryClient();
  const [pendingDelete, setPendingDelete] = useState<Account | null>(null);
  const status = useQuery({ queryKey: ["status"], queryFn: api.status, refetchInterval: 2000 });
  const loginActive = status.data?.loginActive ?? false;
  const prefs = useQuery({ queryKey: ["update-prefs"], queryFn: api.updatePrefStatus });
  const usageEnabled = prefs.data?.prefs.usageEnabled ?? false;
  const usageOptions = prefs.data?.prefs ?? defaultUsageOptions;
  const accounts = useQuery({
    queryKey: ["accounts", page],
    queryFn: () => api.accounts(page),
    staleTime: 60_000,
    refetchOnMount: false,
  });
  const hasSessionLimit = (accounts.data ?? []).some((account) =>
    sessionLimitProviders.has(account.provider),
  );
  const usage = useQuery({
    queryKey: ["account-usage", page],
    queryFn: () => api.accountUsage(page),
    enabled: hasSessionLimit && usageEnabled,
    staleTime: 60_000,
    refetchOnMount: false,
  });
  const usageByID = new Map((usage.data ?? []).map((item) => [item.id, item]));

  const refresh = useMutation({
    mutationFn: () => api.accountUsage(page),
    onSuccess: (data) => {
      client.setQueryData(["account-usage", page], data);
    },
    onError: (error) => toastManager.add({ title: errorText(error), type: "error" }),
  });

  async function login(id: string) {
    try {
      await api.login(id);
      await client.invalidateQueries({ queryKey: ["accounts", page] });
      await client.invalidateQueries({ queryKey: ["account-usage", page] });
      toastManager.add({ title: "登录完成", type: "success" });
    } catch (error) {
      toastManager.add({ title: errorText(error), type: "error" });
    }
  }

  const remove = useMutation({
    mutationFn: (id: string) => api.deleteAccount(id),
    onSuccess: async () => {
      setPendingDelete(null);
      await client.invalidateQueries({ queryKey: ["accounts", page] });
      await client.invalidateQueries({ queryKey: ["account-usage", page] });
    },
    onError: (error) => {
      toastManager.add({ title: errorText(error), type: "error" });
    },
  });

  const actions = (
    <>
      {logins.map((item) => (
        <Button
          key={item.id}
          disabled={loginActive}
          onClick={() => void login(item.id)}
        >
          {item.label}
        </Button>
      ))}
      {loginActive ? (
        <Button variant="outline" onClick={() => void api.cancelLogin().catch((error: unknown) => {
          toastManager.add({ title: errorText(error), type: "error" });
        })}>
          取消登录
        </Button>
      ) : null}
      {prefs.isError && hasSessionLimit ? (
        <Button variant="outline" onClick={() => void prefs.refetch()}>重试读取用量设置</Button>
      ) : null}
      {hasSessionLimit && usageEnabled ? (
        <Button
          variant="outline"
          loading={refresh.isPending}
          onClick={() => refresh.mutate()}
        >
          <ArrowsClockwiseIcon weight="duotone" />
          刷新额度
        </Button>
      ) : null}
    </>
  );

  if (accounts.isError && !accounts.data) {
    return (
      <div role="alert" className="flex flex-col items-start gap-3">
        <p>{errorText(accounts.error)}</p>
        <Button variant="outline" onClick={() => void accounts.refetch()}>重试</Button>
      </div>
    );
  }
  if (!accounts.data) {
    return <p className="text-muted-foreground text-sm">正在读取账号…</p>;
  }
  if (accounts.data.length === 0) {
    return <ProviderEmpty page={page}>{actions}</ProviderEmpty>;
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">{actions}</div>
      <ul className="grid grid-cols-2 items-start gap-3">
        {accounts.data.map((account) => (
            <li className="min-w-0" key={account.id}>
              <Card className="h-full">
                <CardHeader className="p-4">
                  <CardTitle className="flex items-center gap-2 break-all text-base leading-snug">
                    <span>{account.label || account.id}</span>
                    {usageByID.get(account.id)?.plan ? (
                      <Badge>{usageByID.get(account.id)?.plan}</Badge>
                    ) : null}
                  </CardTitle>
                  <CardAction>
                    <Menu>
                      <MenuTrigger
                        aria-label="更多"
                        className={buttonVariants({ variant: "ghost", size: "icon-sm" })}
                      >
                        <DotsThreeIcon weight="bold" />
                      </MenuTrigger>
                      <MenuPopup align="end">
                        <MenuItem
                          variant="destructive"
                          onClick={() => setPendingDelete(account)}
                        >
                          <TrashIcon weight="duotone" />
                          删除
                        </MenuItem>
                      </MenuPopup>
                    </Menu>
                  </CardAction>
                </CardHeader>
                {sessionLimitProviders.has(account.provider) ? (
                  <UsageBars
                    failed={usage.isError}
                    loading={usage.isLoading}
                    usage={usageByID.get(account.id)}
                    options={usageOptions}
                  />
                ) : null}
              </Card>
            </li>
          ))}
        </ul>
      {filledExtra}
      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDelete(null);
          }
        }}
      >
        <AlertDialogPopup>
          <AlertDialogHeader>
            <AlertDialogTitle>删除这个账号？</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDelete?.label || pendingDelete?.id} 会从账号目录移除。正在运行的服务也不再使用它。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogClose render={<Button variant="outline" />}>取消</AlertDialogClose>
            <Button
              variant="destructive"
              loading={remove.isPending}
              onClick={() => {
                if (pendingDelete) {
                  remove.mutate(pendingDelete.id);
                }
              }}
            >
              <TrashIcon weight="duotone" />
              删除
            </Button>
          </AlertDialogFooter>
        </AlertDialogPopup>
      </AlertDialog>
    </section>
  );
};
