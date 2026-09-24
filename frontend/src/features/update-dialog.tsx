import { Events } from "@wailsio/runtime";
import { motion } from "motion/react";
import { type FC, useEffect, useRef, useState } from "react";
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
import { api, rawErrorText } from "@/lib/desktop";
import { currentLocale, localizeErrorMessage, translate } from "@/lib/i18n";
import { usePresentation } from "@/presentation";
import { useUIMotion } from "@/lib/ui-motion";

type Phase = "closed" | "available" | "downloading" | "verifying" | "installing" | "ready" | "failed";

type ReleaseNotice = { version?: string; notes?: string };
type ProgressNotice = { written?: number; total?: number };
type FailureNotice = { stage?: string; message?: string };

let manualCheck = false;
let reshowAvailable: ((notice: ReleaseNotice) => void) | null = null;

export function beginManualUpdateCheck() {
  manualCheck = true;
}

export function showNewRelease(notice: ReleaseNotice) {
  reshowAvailable?.(notice);
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
  const { t, locale } = usePresentation();
  const [phase, setPhase] = useState<Phase>("closed");
  const [version, setVersion] = useState("");
  const [notes, setNotes] = useState("");
  const [progress, setProgress] = useState(0);
  const [failure, setFailure] = useState("");
  const [applying, setApplying] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [retryAction, setRetryAction] = useState<"check" | "download" | "restart">("check");
  const phaseRef = useRef<Phase>("closed");
  const renderedPhase = useRef<Phase>("closed");
  const { transition: phaseTransition } = useUIMotion(0.15);
  const lastError = useRef("");

  function changePhase(next: Phase) {
    phaseRef.current = next;
    setPhase(next);
  }

  function showFailure(message: string) {
    if (lastError.current !== message) {
      toastManager.add({ title: localizeErrorMessage(message, currentLocale()), type: "error" });
      lastError.current = message;
    }
    setApplying(false);
    setFailure(message);
    setDismissed(false);
    changePhase("failed");
  }

  useEffect(() => {
    function showAvailable(body: unknown) {
      const notice = (body ?? {}) as ReleaseNotice;
      setVersion(notice.version ?? "");
      setNotes(notice.notes ?? "");
      setFailure("");
      setProgress(0);
      setDismissed(false);
      setApplying(false);
      lastError.current = "";
      changePhase("available");
    }
    reshowAvailable = showAvailable;

    const offNewRelease = Events.On("updates:new-release", (event) => {
      showAvailable(eventBody(event));
    });
    function showRelease(body: unknown) {
      manualCheck = false;
      const release = (body ?? {}) as ReleaseNotice;
      setVersion(release.version ?? "");
      setNotes(release.notes ?? "");
      setFailure("");
      setProgress(0);
      setDismissed(false);
      setRetryAction("download");
      lastError.current = "";
      changePhase("downloading");
    }

    const offDownload = Events.On("wails:updater:download-started", (event) => {
      showRelease(eventBody(event));
    });
    const offProgress = Events.On("wails:updater:download-progress", (event) => {
      const notice = (eventBody(event) ?? {}) as ProgressNotice;
      setProgress(percent(notice.written ?? 0, notice.total ?? 0));
      changePhase("downloading");
    });
    const offVerifying = Events.On("wails:updater:verifying", () => {
      changePhase("verifying");
    });
    const offInstalling = Events.On("wails:updater:installing", () => {
      changePhase("installing");
    });
    const offReady = Events.On("wails:updater:update-ready", (event) => {
      const release = (eventBody(event) ?? {}) as ReleaseNotice;
      if (release.version) {
        setVersion(release.version);
      }
      setDismissed(false);
      setApplying(false);
      changePhase("ready");
    });
    const offNone = Events.On("wails:updater:no-update", () => {
      if (!manualCheck) {
        return;
      }
      manualCheck = false;
      setApplying(false);
      toastManager.add({ title: translate("已是最新版本", currentLocale()), type: "success" });
    });
    const offError = Events.On("wails:updater:error", (event) => {
      const notice = (eventBody(event) ?? {}) as FailureNotice;
      if (notice.stage === "check") {
        return;
      }
      manualCheck = false;
      setRetryAction(notice.stage === "download" || notice.stage === "verify" || notice.stage === "install"
        || phaseRef.current === "downloading" || phaseRef.current === "verifying" || phaseRef.current === "installing"
        ? "download" : "check");
      showFailure(notice.message || "更新失败");
    });
    return () => {
      reshowAvailable = null;
      offNewRelease();
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
  const fadePhase = open && renderedPhase.current !== "closed" && renderedPhase.current !== phase;
  renderedPhase.current = phase;
  const title =
    phase === "available"
      ? t("发现新版本")
      : phase === "ready"
        ? t("可以重启以完成更新")
        : phase === "failed"
          ? t("更新失败")
          : t("正在更新");
  const detail =
    phase === "available"
      ? t("有新版本{version}可以更新。自动下载已关闭，可以现在手动下载。", { version: version ? ` v${version}` : "" })
      : phase === "ready"
        ? t("{version}已准备好。重启后会换成新版本，代理会停止。", { version: version ? `v${version} ` : "" })
        : phase === "failed"
          ? localizeErrorMessage(failure, locale)
          : phase === "verifying"
            ? t("正在校验下载的文件。")
            : phase === "installing"
              ? t("正在准备安装。")
              : t("正在下载{version}。", { version: version ? ` v${version}` : "" });

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
        <motion.div key={phase} initial={fadePhase ? { opacity: 0 } : false} animate={{ opacity: 1 }} transition={phaseTransition}>
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
            <p className="max-h-40 overflow-auto px-6 pb-2 text-sm whitespace-pre-wrap">{notes}</p>
          ) : null}
        </motion.div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setDismissed(true)}>
            {phase === "ready" ? t("稍后") : phase === "available" ? t("忽略") : t("关闭")}
          </Button>
          {phase === "available" ? (
            <Button
              loading={applying}
              onClick={() => {
                setApplying(true);
                setRetryAction("download");
                lastError.current = "";
                changePhase("downloading");
                void api.downloadPendingUpdate().catch((error: unknown) => {
                  showFailure(rawErrorText(error));
                });
              }}
            >
              {t("立即下载")}
            </Button>
          ) : null}
          {phase === "failed" ? (
            <Button
              loading={applying}
              onClick={() => {
                setApplying(true);
                setFailure("");
                lastError.current = "";
                if (retryAction === "download") {
                  changePhase("downloading");
                  void api.downloadPendingUpdate().catch((error: unknown) => {
                    showFailure(rawErrorText(error));
                  });
                } else if (retryAction === "restart") {
                  void api.applyUpdate().catch((error: unknown) => {
                    showFailure(rawErrorText(error));
                  });
                } else {
                  beginManualUpdateCheck();
                  changePhase("downloading");
                  void api.checkUpdates().catch((error: unknown) => {
                    finishManualUpdateCheck();
                    showFailure(rawErrorText(error));
                  });
                }
              }}
            >
              {t("重试")}
            </Button>
          ) : null}
          {phase === "ready" ? (
            <Button
              loading={applying}
              onClick={() => {
                setApplying(true);
                setRetryAction("restart");
                lastError.current = "";
                void api.applyUpdate().catch((error: unknown) => {
                  showFailure(rawErrorText(error));
                });
              }}
            >
              {t("立即重启")}
            </Button>
          ) : null}
        </DialogFooter>
      </DialogPopup>
    </Dialog>
  );
};
