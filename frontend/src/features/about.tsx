import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ArrowSquareOutIcon } from "@phosphor-icons/react";
import { type FC, useState } from "react";
import { Button } from "@/components/ui/button";
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
import { toastManager } from "@/components/ui/toast";
import appIcon from "@/assets/images/appicon.png";
import { beginManualUpdateCheck, finishManualUpdateCheck } from "@/features/update-dialog";
import { api, errorText } from "@/lib/desktop";

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

const GithubMark: FC<{ size?: number }> = ({ size = 16 }) => (
  <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
    <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
  </svg>
);

export const AboutTab: FC<{ version: string | undefined }> = ({ version }) => {
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
    return <p className="text-muted-foreground text-sm">正在读取关于信息…</p>;
  }

  const current = status.data;
  return (
    <div className="flex flex-col gap-8">
      <div className="flex items-center gap-4">
        <img src={appIcon} alt="akproxy 图标" className="size-16" />
        <div className="flex flex-col gap-1">
          <div className="flex items-baseline gap-2">
            <span className="text-lg font-medium">akproxy</span>
            {version ? <span className="text-muted-foreground text-sm">{version}</span> : null}
          </div>
          <span className="text-muted-foreground text-sm">{current.platform}</span>
        </div>
      </div>
      <div className="flex flex-col gap-5">
        <PrefSwitch
          label="自动更新"
          description="自动检查发现新版本后直接下载，下载完成再询问是否重启。关闭时只提示，不下载。"
          checked={current.prefs.autoUpdate}
          onChange={(autoUpdate) =>
            save.mutate({ ...current.prefs, autoUpdate })
          }
        />
        <PrefSwitch
          label="自动检查更新"
          description="启动时检查一次，并按下面的间隔持续检查。关闭后只能手动检查。"
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
        <div className="flex items-center justify-between">
          <span className="text-muted-foreground text-sm">
            {current.lastCheckAt
              ? `上次检查 ${current.lastCheckAt} · ${current.lastCheckResult || "检查完成"}`
              : "还没有检查记录"}
          </span>
          <Button
            variant="outline"
            loading={check.isPending}
            onClick={() => {
              beginManualUpdateCheck();
              check.mutate();
            }}
          >
            检查更新
          </Button>
        </div>
      </div>
      <div className="text-muted-foreground flex flex-col gap-2 text-sm">
        <a
          className="flex items-center gap-2 underline-offset-4 hover:underline"
          href="https://github.com/aghub-app/akproxy"
          target="_blank"
          rel="noreferrer"
        >
          <GithubMark size={16} />
          github.com/aghub-app/akproxy
          <ArrowSquareOutIcon size={14} />
        </a>
        <div className="flex flex-wrap gap-x-4 gap-y-1">
          <a className="underline-offset-4 hover:underline" href="https://github.com/aghub-app/akproxy#readme" target="_blank" rel="noreferrer">
            README
          </a>
          <a className="underline-offset-4 hover:underline" href="https://github.com/aghub-app/akproxy/blob/main/LICENSE" target="_blank" rel="noreferrer">
            LICENSE
          </a>
          <a className="underline-offset-4 hover:underline" href="https://github.com/aghub-app/akproxy/issues" target="_blank" rel="noreferrer">
            问题反馈
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
  const matched = intervalPresets.some((item) => item.value === String(hours));
  const [customMode, setCustomMode] = useState(matched ? false : true);
  const [custom, setCustom] = useState(matched ? "" : String(hours));
  const preset = matched && !customMode ? String(hours) : "custom";
  return (
    <Field>
      <FieldLabel>检查间隔</FieldLabel>
      <Select
        value={preset}
        onValueChange={(value) => {
          if (value === "custom") {
            setCustomMode(true);
            setCustom(String(hours));
            return;
          }
          setCustomMode(false);
          const next = Number(value);
          if (Number.isFinite(next)) {
            onChange(next);
          }
        }}
      >
        <SelectTrigger disabled={disabled}>
          <SelectValue>
            {(value) => (typeof value === "string" ? presetLabels[value] ?? value : null)}
          </SelectValue>
        </SelectTrigger>
        <SelectPopup>
          {intervalPresets.map((item) => (
            <SelectItem key={item.value} value={item.value}>
              {item.label}
            </SelectItem>
          ))}
        </SelectPopup>
      </Select>
      {preset === "custom" ? (
        <Input
          type="number"
          min={1}
          max={720}
          step={1}
          placeholder="24"
          value={custom}
          onChange={(event) => {
            const raw = event.target.value;
            setCustom(raw);
            const hours = Number(raw);
            if (raw !== "" && Number.isInteger(hours) && hours >= 1 && hours <= 720) {
              onChange(hours);
            }
          }}
        />
      ) : null}
      <FieldDescription>
        {preset === "custom"
          ? "填写 1 到 720 之间的小时数。"
          : "自动检查发现新版本的节奏，改动立即生效。"}
      </FieldDescription>
    </Field>
  );
};
