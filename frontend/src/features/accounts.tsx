import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useState } from "react";
import {
  AlertDialog,
  AlertDialogClose,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogPopup,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress, ProgressIndicator, ProgressTrack } from "@/components/ui/progress";
import { toastManager } from "@/components/ui/toast";
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
      return <p className="px-4 text-muted-foreground text-xs">正在读取额度…</p>;
    }
    if (failed) {
      return <p className="px-4 text-muted-foreground text-xs">额度暂时读不到</p>;
    }
    return null;
  }
  if (usage.error && !(usage.windows && usage.windows.length > 0)) {
    return <p className="px-4 text-muted-foreground text-xs">{usage.error}</p>;
  }
  return (
    <div className="flex flex-col gap-3 px-4">
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
}> = ({ page, logins }) => {
  const client = useQueryClient();
  const [pendingDelete, setPendingDelete] = useState<Account | null>(null);
  const [loggingIn, setLoggingIn] = useState<string | null>(null);
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
    setLoggingIn(id);
    try {
      await api.login(id);
      await client.invalidateQueries({ queryKey: ["accounts", page] });
      toastManager.add({ title: "登录完成", type: "success" });
    } catch (error) {
      toastManager.add({ title: errorText(error), type: "error" });
    } finally {
      setLoggingIn(null);
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

  return (
    <section className="flex flex-col gap-3">
      <div className="flex flex-wrap gap-2">
        {logins.map((item) => (
          <Button
            key={item.id}
            loading={loggingIn === item.id}
            disabled={loggingIn !== null && loggingIn !== item.id}
            onClick={() => void login(item.id)}
          >
            {item.label}
          </Button>
        ))}
        {loggingIn ? (
          <Button variant="outline" onClick={() => void api.cancelLogin()}>
            取消登录
          </Button>
        ) : null}
      </div>
      {accounts.data && accounts.data.length === 0 ? (
        <p className="text-muted-foreground text-sm">还没有浏览器登录的账号。</p>
      ) : null}
      {accounts.data && accounts.data.length > 0 ? (
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
                </CardHeader>
                {sessionLimitProviders.has(account.provider) ? (
                  <UsageBars
                    failed={usage.isError}
                    loading={usage.isLoading}
                    usage={usageByID.get(account.id)}
                  />
                ) : null}
                <CardFooter className="p-4 pt-3">
                  <Button
                    variant="destructive-outline"
                    onClick={() => setPendingDelete(account)}
                  >
                    删除
                  </Button>
                </CardFooter>
              </Card>
            </li>
          ))}
        </ul>
      ) : null}
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
              删除
            </Button>
          </AlertDialogFooter>
        </AlertDialogPopup>
      </AlertDialog>
    </section>
  );
};
