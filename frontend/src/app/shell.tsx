import { providerPages } from "@/lib/providers";
import { HardDrivesIcon, HouseIcon } from "@phosphor-icons/react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { type FC } from "react";
import { NavLink, Outlet } from "react-router";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { toastManager } from "@/components/ui/toast";
import { api, errorText, type Status } from "@/lib/desktop";
import { cn } from "@/lib/utils";

const SettingsIcon: FC<{ size?: number }> = ({ size }) => (
  <HardDrivesIcon size={size} weight="duotone" />
);

const pages = [
  { to: "/", label: "首页", icon: HouseIcon },
  ...providerPages,
  { to: "/settings", label: "设置", icon: SettingsIcon },
];

const actionLabel = { start: "启动", stop: "停止", restart: "重启" } as const;

export const AppLayout: FC = () => {
  const client = useQueryClient();
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
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: ["status"] });
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

  return (
    <div className="flex h-full flex-col bg-background text-foreground">
      <header className="flex h-14 shrink-0 items-center gap-3 border-b px-4">
        <div className="font-medium">akproxy</div>
        <Badge variant={current?.running ? "success" : "secondary"}>
          {current?.running ? "运行中" : "已停止"}
        </Badge>
        <button
          className="min-w-0 truncate text-left text-sm text-muted-foreground"
          type="button"
          onClick={() => {
            if (!current?.address) {
              return;
            }
            void navigator.clipboard.writeText(current.address).then(() => {
              toastManager.add({ title: "已复制地址", type: "success" });
            });
          }}
        >
          {current?.address || "还没有地址"}
        </button>
        {current?.allInterfaces ? (
          <span className="text-muted-foreground text-xs">同时监听所有网络接口</span>
        ) : null}
        {current?.restartRequired ? (
          <span className="text-warning-foreground text-xs">
            可以考虑重启，之后会使用 {current.savedAddress}
          </span>
        ) : null}
        {current?.error ? (
          <span className="truncate text-destructive-foreground text-xs">{current.error}</span>
        ) : null}
        <Button
          className="ms-auto"
          variant={action === "stop" ? "destructive" : "default"}
          loading={control.isPending}
          onClick={requestControl}
        >
          {actionLabel[action]}
        </Button>
      </header>
      <div className="flex min-h-0 flex-1">
        <nav className="flex w-56 shrink-0 flex-col gap-1 border-r p-3">
          {pages.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.to}
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
          })}
        </nav>
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <ScrollArea className="h-full">
            <div className="mx-auto flex w-full max-w-3xl flex-col px-6 py-6">
              <Outlet />
            </div>
          </ScrollArea>
        </div>
      </div>
    </div>
  );
};
