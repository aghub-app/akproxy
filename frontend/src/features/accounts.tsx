import { DotsThreeIcon, TrashIcon } from "@phosphor-icons/react";
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
import { Card, CardAction, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Menu, MenuItem, MenuPopup, MenuTrigger } from "@/components/ui/menu";
import { Progress, ProgressIndicator, ProgressTrack } from "@/components/ui/progress";
import { toastManager } from "@/components/ui/toast";
import { ProviderEmpty } from "@/features/provider-empty";
import { api, errorText, type Account, type AccountUsage, type UsageWindow } from "@/lib/desktop";

const windowLabel: Record<UsageWindow["kind"], string> = {
  "5h": "5 小时",
  weekly: "每周",
  weekly_opus: "每周 Opus",
  weekly_sonnet: "每周 Sonnet",
};

const sessionLimitProviders = new Set(["codex", "claude"]);

function resetLabel(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const sameDay = date.toDateString() === new Date().toDateString();
  const time = new Intl.DateTimeFormat("zh-CN", { hour: "2-digit", minute: "2-digit" }).format(date);
  if (sameDay) {
    return `${time} 重置`;
  }
  const day = new Intl.DateTimeFormat("zh-CN", { month: "numeric", day: "numeric" }).format(date);
  return `${day} ${time} 重置`;
}

const UsageBars: FC<{ usage?: AccountUsage; loading: boolean; failed: boolean }> = ({
  usage,
  loading,
  failed,
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
  if (usage.error && !(usage.windows && usage.windows.length > 0)) {
    return <p className="px-4 pb-4 text-muted-foreground text-xs">{usage.error}</p>;
  }
  return (
    <div className="flex flex-col gap-3 px-4 pb-4">
      {(usage.windows ?? []).map((window) => (
        <div className="flex flex-col gap-1" key={window.kind}>
          <div className="flex items-center justify-between gap-2 text-xs">
            <span className="text-muted-foreground">{windowLabel[window.kind]}</span>
            <span className="tabular-nums">
              {Math.round(window.used)}%
              {window.resetsAt ? <span className="text-muted-foreground"> · {resetLabel(window.resetsAt)}</span> : null}
            </span>
          </div>
          <Progress value={window.used}>
            <ProgressTrack>
              <ProgressIndicator />
            </ProgressTrack>
          </Progress>
        </div>
      ))}
    </div>
  );
};

const providerLabel: Record<string, string> = {
  codex: "Codex",
  xai: "Grok",
  claude: "Claude",
  antigravity: "Gemini",
  kimi: "Kimi 中文站",
  "kimi-ai": "Kimi 国际站",
  "kimi.ai": "Kimi 国际站",
  devin: "Devin",
};

export const AccountsPanel: FC<{
  page: string;
  logins: { id: string; label: string }[];
  filledExtra?: ReactNode;
}> = ({ page, logins, filledExtra }) => {
  const client = useQueryClient();
  const [pendingDelete, setPendingDelete] = useState<Account | null>(null);
  const status = useQuery({ queryKey: ["status"], queryFn: api.status, refetchInterval: 2000 });
  const loginActive = status.data?.loginActive ?? false;
  const accounts = useQuery({
    queryKey: ["accounts", page],
    queryFn: () => api.accounts(page),
  });
  const hasSessionLimit = (accounts.data ?? []).some((account) =>
    sessionLimitProviders.has(account.provider),
  );
  const usage = useQuery({
    queryKey: ["account-usage", page],
    queryFn: () => api.accountUsage(page),
    enabled: hasSessionLimit,
    refetchInterval: 60_000,
  });
  const usageByID = new Map((usage.data ?? []).map((item) => [item.id, item]));

  async function login(id: string) {
    try {
      await api.login(id);
      await client.invalidateQueries({ queryKey: ["accounts", page] });
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
        <Button variant="outline" onClick={() => void api.cancelLogin()}>
          取消登录
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
      <ul className="grid grid-cols-2 gap-3">
        {accounts.data.map((account) => (
            <li className="min-w-0" key={account.id}>
              <Card className="h-full">
                <CardHeader className="p-4">
                  <CardTitle className="break-all text-base leading-snug">
                    {account.label || account.id}
                  </CardTitle>
                  <CardDescription>
                    {providerLabel[account.provider] ?? account.provider}
                  </CardDescription>
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
