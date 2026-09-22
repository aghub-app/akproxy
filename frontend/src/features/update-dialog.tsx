import { Events } from "@wailsio/runtime";
import { type FC, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogPopup,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress, ProgressIndicator, ProgressTrack } from "@/components/ui/progress";
import { toastManager } from "@/components/ui/toast";
import { api, errorText } from "@/lib/desktop";

type Phase = "closed" | "downloading" | "verifying" | "installing" | "ready" | "failed";

type ReleaseNotice = { version?: string; notes?: string };
type ProgressNotice = { written?: number; total?: number };
type FailureNotice = { stage?: string; message?: string };

let manualCheck = false;

export function beginManualUpdateCheck() {
  manualCheck = true;
}

export function finishManualUpdateCheck() {
  manualCheck = false;
}

function eventBody(event: { data?: unknown }): unknown {
  const data = event?.data;
  return Array.isArray(data) ? data[0] : data;
}

function percent(written: number, total: number) {
  if (total <= 0) {
    return 0;
  }
  return Math.max(0, Math.min(100, Math.round((written / total) * 100)));
}

export const UpdateDialog: FC = () => {
  const [phase, setPhase] = useState<Phase>("closed");
  const [version, setVersion] = useState("");
  const [notes, setNotes] = useState("");
  const [progress, setProgress] = useState(0);
  const [failure, setFailure] = useState("");
  const [applying, setApplying] = useState(false);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    function showRelease(body: unknown) {
      manualCheck = false;
      const release = (body ?? {}) as ReleaseNotice;
      setVersion(release.version ?? "");
      setNotes(release.notes ?? "");
      setFailure("");
      setProgress(0);
      setDismissed(false);
      setPhase("downloading");
    }

    const offAvailable = Events.On("wails:updater:update-available", (event) => {
      showRelease(eventBody(event));
    });
    const offDownload = Events.On("wails:updater:download-started", (event) => {
      showRelease(eventBody(event));
    });
    const offProgress = Events.On("wails:updater:download-progress", (event) => {
      const notice = (eventBody(event) ?? {}) as ProgressNotice;
      setProgress(percent(notice.written ?? 0, notice.total ?? 0));
      setPhase("downloading");
    });
    const offVerifying = Events.On("wails:updater:verifying", () => {
      setPhase("verifying");
    });
    const offInstalling = Events.On("wails:updater:installing", () => {
      setPhase("installing");
    });
    const offReady = Events.On("wails:updater:update-ready", (event) => {
      const release = (eventBody(event) ?? {}) as ReleaseNotice;
      if (release.version) {
        setVersion(release.version);
      }
      setDismissed(false);
      setPhase("ready");
    });
    const offNone = Events.On("wails:updater:no-update", () => {
      if (!manualCheck) {
        return;
      }
      manualCheck = false;
      toastManager.add({ title: "已是最新版本", type: "success" });
    });
    const offError = Events.On("wails:updater:error", (event) => {
      const notice = (eventBody(event) ?? {}) as FailureNotice;
      if (notice.stage === "check") {
        return;
      }
      manualCheck = false;
      setFailure(notice.message || "更新失败");
      setDismissed(false);
      setPhase("failed");
    });
    return () => {
      offAvailable();
      offDownload();
      offProgress();
      offVerifying();
      offInstalling();
      offReady();
      offNone();
      offError();
    };
  }, []);

  const open = phase !== "closed" && !dismissed;
  const title =
    phase === "ready" ? "可以重启以完成更新" : phase === "failed" ? "更新失败" : "正在更新";
  const detail =
    phase === "ready"
      ? `${version ? `v${version} ` : ""}已准备好。重启后会换成新版本，代理会停止。`
      : phase === "failed"
        ? failure
        : phase === "verifying"
          ? "正在校验下载的文件。"
          : phase === "installing"
            ? "正在准备安装。"
            : `正在下载${version ? ` v${version}` : ""}。`;

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) {
          setDismissed(true);
        }
      }}
    >
      <DialogPopup>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{detail}</DialogDescription>
        </DialogHeader>
        {phase === "downloading" ? (
          <div className="px-6 pb-2">
            <Progress value={progress}>
              <ProgressTrack>
                <ProgressIndicator />
              </ProgressTrack>
            </Progress>
          </div>
        ) : null}
        {phase !== "downloading" && notes ? (
          <p className="max-h-40 overflow-auto px-6 text-sm whitespace-pre-wrap">{notes}</p>
        ) : null}
        <DialogFooter>
          <Button variant="outline" onClick={() => setDismissed(true)}>
            {phase === "ready" ? "稍后" : "关闭"}
          </Button>
          {phase === "ready" ? (
            <Button
              loading={applying}
              onClick={() => {
                setApplying(true);
                void api.applyUpdate().catch((error: unknown) => {
                  setApplying(false);
                  setFailure(errorText(error));
                  setPhase("failed");
                });
              }}
            >
              立即重启
            </Button>
          ) : null}
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  );
};
