import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { GithubLogoIcon } from "@phosphor-icons/react";
import { Browser } from "@wailsio/runtime";
import { AnimatePresence } from "motion/react";
import { type FC, type MouseEvent, useState } from "react";
import { Button } from "@/components/ui/button";
import { AnimatedField } from "@/components/animated-field";
import { Field, FieldDescription, FieldLabel } from "@/components/ui/field";
import {
  NumberField,
  NumberFieldDecrement,
  NumberFieldGroup,
  NumberFieldIncrement,
  NumberFieldInput,
} from "@/components/ui/number-field";
import {
  Select,
  SelectItem,
  SelectPopup,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { toastManager } from "@/components/ui/toast";
import appIcon from "@/assets/images/appicon.png";
import { beginManualUpdateCheck, finishManualUpdateCheck, showNewRelease } from "@/features/update-dialog";
import { api, errorText, type BuildInfo } from "@/lib/desktop";
import { usePresentation } from "@/presentation";

const intervalPresets = [
  { value: "1", label: "每 1 小时" },
  { value: "6", label: "每 6 小时" },
  { value: "24", label: "每 24 小时" },
  { value: "168", label: "每 7 天" },
  { value: "custom", label: "自定义" },
];

const presetLabels = Object.fromEntries(intervalPresets.map((item) => [item.value, item.label]));

function notifyError(error: unknown) {
  toastManager.add({ title: errorText(error), type: "error" });
}

function openExternalLink(event: MouseEvent<HTMLAnchorElement>) {
  event.preventDefault();
  void Browser.OpenURL(event.currentTarget.href).catch(notifyError);
}

export const AboutTab: FC<{ build: BuildInfo | undefined }> = ({ build }) => {
  const { t, locale } = usePresentation();
  const client = useQueryClient();
  const status = useQuery({ queryKey: ["update-prefs"], queryFn: api.updatePrefStatus });
  const check = useMutation({
    mutationFn: api.checkUpdates,
    onError: (error) => {
      finishManualUpdateCheck();
      notifyError(error);
    },
  });
  const save = useMutation({
    mutationFn: api.saveUpdatePrefs,
    onSuccess: (next) => {
      client.setQueryData(["update-prefs"], next);
    },
    onError: (error) => {
      notifyError(error);
      void client.invalidateQueries({ queryKey: ["update-prefs"] });
    },
  });

  if (!status.data) {
    return <p className="text-muted-foreground text-sm">{t("正在读取关于信息…")}</p>;
  }

  const current = status.data;
  const dev = build?.version === "dev";
  return (
    <div className="flex flex-col gap-10">
      <div className="flex items-center gap-4">
        <img src={appIcon} alt={t("akproxy 图标")} className="size-16" />
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex items-baseline gap-2">
            <span className="text-lg font-medium">akproxy</span>
            {build ? <span className="text-muted-foreground text-sm" title={dev ? build.revision || undefined : undefined}>{build.version}{dev ? ` · ${build.revision ? build.revision.slice(0, 8) : t("不可用")}` : ""}</span> : null}
          </div>
          <span className="text-muted-foreground text-sm">{current.platform}</span>
          <div className="flex items-center gap-3">
            {current.latestVersion ? (
              <button
                type="button"
                className="text-primary text-sm underline-offset-4 hover:underline"
                onClick={() => showNewRelease({ version: current.latestVersion })}
              >
                {t("发现新版本 v{version}，查看详情", { version: current.latestVersion })}
              </button>
            ) : current.lastCheckAt ? (
              <span className="text-muted-foreground text-xs tabular-nums">
                {t(current.lastCheckResult || "检查完成")} · {t("上次检查 {time}", { time: new Intl.DateTimeFormat(locale, { dateStyle: "short", timeStyle: "short" }).format(new Date(current.lastCheckAt.replace(" ", "T"))) })}
              </span>
            ) : null}
            <Button
              variant="outline"
              size="sm"
              loading={check.isPending}
              onClick={() => {
                beginManualUpdateCheck();
                check.mutate();
              }}
            >
              {t("检查更新")}
            </Button>
          </div>
        </div>
      </div>
      <div className="flex flex-col gap-5">
        <PrefSwitch
          label={t("自动更新")}
          description={t("自动检查发现新版本后直接下载，下载完成再询问是否重启。关闭时只提示，不下载。")}
          checked={current.prefs.autoUpdate}
          onChange={(autoUpdate) =>
            save.mutate({ ...current.prefs, autoUpdate })
          }
        />
        <PrefSwitch
          label={t("自动检查更新")}
          description={t("启动时检查一次，并按下面的间隔持续检查。关闭后只能手动检查。")}
          checked={current.prefs.autoCheck}
          onChange={(autoCheck) =>
            save.mutate({ ...current.prefs, autoCheck })
          }
        />
        <IntervalField
          hours={current.prefs.checkIntervalHours}
          disabled={!current.prefs.autoCheck}
          onChange={(hours) => save.mutate({ ...current.prefs, checkIntervalHours: hours })}
        />
      </div>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-3 border-t pt-5 text-sm">
        <span className="flex items-center gap-2 font-medium text-muted-foreground">
          <GithubLogoIcon size={18} weight="duotone" aria-hidden />
          MIT
        </span>
        <div className="flex items-center gap-5">
          <a className="underline underline-offset-4 hover:text-primary" href="https://github.com/aghub-app/akproxy" onClick={openExternalLink}>
            {t("源代码")}
          </a>
          <a className="underline underline-offset-4 hover:text-primary" href="https://github.com/aghub-app/akproxy/issues" onClick={openExternalLink}>
            {t("问题反馈")}
          </a>
        </div>
      </div>
    </div>
  );
};

const PrefSwitch: FC<{
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}> = ({ label, description, checked, onChange }) => (
  <Field>
    <div className="flex items-center gap-3">
      <Switch checked={checked} onCheckedChange={onChange} />
      <FieldLabel>{label}</FieldLabel>
    </div>
    <FieldDescription>{description}</FieldDescription>
  </Field>
);

const IntervalField: FC<{
  hours: number;
  disabled: boolean;
  onChange: (hours: number) => void;
}> = ({ hours, disabled, onChange }) => {
  const { t } = usePresentation();
  const matched = intervalPresets.some((item) => item.value === String(hours));
  const [customMode, setCustomMode] = useState(matched ? false : true);
  const preset = matched && !customMode ? String(hours) : "custom";
  return (
    <Field className={disabled ? "opacity-50" : undefined}>
      <FieldLabel>{t("检查间隔")}</FieldLabel>
      <div className="flex items-center gap-2">
        <Select
          value={preset}
          onValueChange={(value) => {
            if (value === "custom") {
              setCustomMode(true);
              return;
            }
            setCustomMode(false);
            const next = Number(value);
            if (Number.isFinite(next)) {
              onChange(next);
            }
          }}
        >
          <SelectTrigger disabled={disabled} className="w-auto min-w-28">
            <SelectValue>
              {(value) => (typeof value === "string" ? t(presetLabels[value] ?? value) : null)}
            </SelectValue>
          </SelectTrigger>
          <SelectPopup>
            {intervalPresets.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {t(item.label)}
              </SelectItem>
            ))}
          </SelectPopup>
        </Select>
        <AnimatePresence initial={false}>
          {preset === "custom" ? (
            <AnimatedField key="custom-interval">{(present) => (
              <NumberField
                value={hours}
                min={1}
                max={720}
                step={1}
                disabled={disabled || !present}
                onValueChange={(value) => {
                  if (value !== null && Number.isInteger(value)) onChange(value);
                }}
              >
                <NumberFieldGroup className="w-24">
                  <NumberFieldDecrement className="px-1.5" />
                  <NumberFieldInput className="px-1" />
                  <NumberFieldIncrement className="px-1.5" />
                </NumberFieldGroup>
              </NumberField>
            )}</AnimatedField>
          ) : null}
        </AnimatePresence>
      </div>
      <FieldDescription>
        {preset === "custom"
          ? t("填写 1 到 720 之间的小时数。")
          : t("自动检查发现新版本的节奏，改动立即生效。")}
      </FieldDescription>
    </Field>
  );
};
