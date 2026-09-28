import { providerPages } from "@/lib/providers";
import { HardDrivesIcon, HouseIcon, PlayIcon, StopIcon } from "@phosphor-icons/react";
import { useMutation, useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC, useEffect } from "react";
import { NavLink, Outlet, useNavigate } from "react-router";
import { Events } from "@wailsio/runtime";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { toastManager } from "@/components/ui/toast";
import { api, errorText, type Status } from "@/lib/desktop";
import { cn } from "@/lib/utils";
import { controlSuccessToast, serviceBarNotice } from "./service-bar";
import { usePresentation } from "@/presentation";

const HomeIcon: FC<{ size?: number }> = ({ size }) => (
  <HouseIcon size={size} weight="duotone" />
);

const SettingsIcon: FC<{ size?: number }> = ({ size }) => (
  <HardDrivesIcon size={size} weight="duotone" />
);

const mainPages = [
  { to: "/", label: "首页", icon: HomeIcon },
  ...providerPages,
];

const settingsPage = { to: "/settings", label: "设置", icon: SettingsIcon };
let cliInstallToastShown = false;

function SideLink({ item }: { item: typeof settingsPage }) {
  const Icon = item.icon;
  return (
    <NavLink
      to={item.to}
      end={item.to === "/"}
      className={({ isActive }) =>
        cn(
          "flex items-center gap-2 rounded-lg px-3 py-2 text-left text-sm",
          isActive ? "bg-accent text-accent-foreground" : "hover:bg-accent/60",
        )
      }
    >
      <Icon size={16} />
      {item.label}
    </NavLink>
  );
}

const actionLabel = { start: "启动", stop: "停止", restart: "重启" } as const;
const usagePages = ["codex", "grok", "claude", "gemini", "kimi", "devin", "meta"] as const;

export const AppLayout: FC = () => {
  const { t, locale } = usePresentation();
  const client = useQueryClient();
  const navigate = useNavigate();
  const usagePrefs = useQuery({ queryKey: ["update-prefs"], queryFn: api.updatePrefStatus });
  const accountLists = useQueries({ queries: usagePages.map((page) => ({
    queryKey: ["accounts", page],
    queryFn: ({ signal }) => api.accounts(page, signal),
    staleTime: 60_000,
    refetchInterval: 60_000,
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
  })) });
  useQueries({ queries: usagePages.map((page, index) => ({
    queryKey: ["account-usage", page],
    queryFn: () => api.accountUsage(page),
    enabled: usagePrefs.data?.prefs.usageEnabled === true && (accountLists[index].data?.length ?? 0) > 0,
    staleTime: 60_000,
    refetchInterval: 60_000,
    refetchIntervalInBackground: true,
    refetchOnWindowFocus: true,
  })) });
  useEffect(
    () => Events.On("app:about", () => navigate("/settings?tab=about")),
    [navigate],
  );
  useEffect(() => {
    if (cliInstallToastShown) {
      return;
    }
    cliInstallToastShown = true;
    void api.cliStatus().then((status) => {
      if (!status.error) {
        return;
      }
      toastManager.add({
        title: errorText(status.error),
        type: "error",
        actionProps: {
          children: t("打开设置"),
          onClick: () => navigate("/settings?tab=cli"),
        },
      });
    }).catch(() => undefined);
  }, [navigate, t]);
  const status = useQuery({
    queryKey: ["status"],
    queryFn: api.status,
    refetchInterval: 2000,
  });

  const control = useMutation({
    mutationFn: async (action: Status["action"]) => {
      if (action === "stop") {
        await api.stop();
        return;
      }
      if (action === "restart") {
        await api.restart();
        return;
      }
      await api.start();
    },
    onSuccess: async (_data, ran) => {
      await client.invalidateQueries({ queryKey: ["status"] });
      const started = controlSuccessToast(ran, locale);
      if (started) {
        toastManager.add({ ...started, type: "success" });
      }
    },
    onError: (error) => {
      toastManager.add({ title: errorText(error), type: "error" });
    },
  });

  function requestControl() {
    control.mutate(status.data?.action ?? "start");
  }

  const current = status.data;
  const action = current?.action ?? "start";
  const notice = serviceBarNotice(current, locale);

  return (
    <div className="flex h-full flex-col bg-background text-foreground">
      <header className="shrink-0 border-b">
        <div className="flex h-12 items-center gap-3 px-4">
          <div className="min-w-0 flex-1 truncate text-sm font-medium">akproxy</div>
          <span
            className={cn(
              "size-2 shrink-0 rounded-full",
              current?.running ? "bg-success" : "bg-muted-foreground/60",
            )}
            role="img"
            aria-label={current?.running ? t("运行中") : t("已停止")}
            title={current?.running ? t("运行中") : t("已停止")}
          />
          <Button
            variant={action === "stop" ? "destructive" : "default"}
            loading={control.isPending}
            onClick={requestControl}
          >
            {action === "stop" ? <StopIcon weight="duotone" /> : null}
            {action === "start" ? <PlayIcon weight="duotone" /> : null}
            {t(actionLabel[action])}
          </Button>
        </div>
        {notice ? (
          <p
            className={cn(
              "border-t px-4 py-2 text-xs",
              current?.error ? "text-destructive-foreground" : "text-warning-foreground",
            )}
            role={current?.error ? "alert" : "status"}
          >
            {notice}
          </p>
        ) : null}
      </header>
      <div className="flex min-h-0 flex-1">
        <nav className="flex w-56 shrink-0 flex-col border-r">
          <div className="flex min-h-0 flex-1 flex-col gap-1 overflow-y-auto p-3">
            {mainPages.map((item) => (
              <SideLink key={item.to} item={{ ...item, label: t(item.label) }} />
            ))}
          </div>
          <div className="shrink-0 p-3 pt-0">
            <SideLink item={{ ...settingsPage, label: t(settingsPage.label) }} />
          </div>
        </nav>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <ScrollArea className="h-full" stretch>
            <div className="mx-auto flex w-full max-w-3xl flex-1 flex-col px-6 py-6">
              <Outlet />
            </div>
          </ScrollArea>
        </div>
      </div>
    </div>
  );
};
