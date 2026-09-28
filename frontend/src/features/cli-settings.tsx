import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { FC } from "react";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import { Switch } from "@/components/ui/switch";
import { toastManager } from "@/components/ui/toast";
import { api, errorText, type UpdatePrefsStatus } from "@/lib/desktop";
import { usePresentation } from "@/presentation";

export const CLISettings: FC = () => {
  const { t } = usePresentation();
  const client = useQueryClient();
  const status = useQuery({ queryKey: ["cli-status"], queryFn: api.cliStatus });
  const save = useMutation({
    mutationFn: (enabled: boolean) => api.setCLIEnabled(enabled),
    onSuccess: (next) => {
      client.setQueryData(["cli-status"], next);
      client.setQueryData<UpdatePrefsStatus>(["update-prefs"], (current) =>
        current ? { ...current, prefs: { ...current.prefs, cliEnabled: next.enabled } } : current,
      );
    },
    onError: (error) => {
      toastManager.add({ title: errorText(error), type: "error" });
      void client.invalidateQueries({ queryKey: ["cli-status"] });
      void client.invalidateQueries({ queryKey: ["update-prefs"] });
    },
  });
  if (status.isError && !status.data) {
    return (
      <div role="alert" className="flex items-center gap-3 text-sm">
        <span>{t("命令行暂时读不到")}</span>
        <Button variant="outline" onClick={() => void status.refetch()}>{t("重试")}</Button>
      </div>
    );
  }
  if (!status.data) {
    return <p className="text-muted-foreground text-sm">{t("正在读取命令行设置…")}</p>;
  }
  return (
    <div className="flex max-w-xl flex-col gap-6">
      <Field>
        <div className="flex items-center gap-3">
          <Switch
            disabled={save.isPending}
            checked={status.data.enabled}
            onCheckedChange={(checked) => save.mutate(checked)}
          />
          <FieldLabel>{t("终端命令")}</FieldLabel>
        </div>
        <FieldDescription>{t("打开后，终端里可以直接使用 akproxy。关闭后不再安装，并移除已经加上的命令。")}</FieldDescription>
      </Field>
      {status.data.error ? <p className="text-sm text-destructive-foreground">{errorText(status.data.error)}</p> : null}
    </div>
  );
};
