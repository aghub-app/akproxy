import type { Status } from "@/lib/desktop";

export function controlSuccessToast(action: Status["action"]): { title: string; description: string } | null {
  if (action !== "start") {
    return null;
  }
  return { title: "服务已启动", description: "可以到首页看接入方式" };
}

// The navbar's second line is one message: a start failure, or the two
// addresses the user must compare before restarting.
export function serviceBarNotice(status: Status | undefined): string {
  if (!status) {
    return "";
  }
  if (status.error) {
    return status.error;
  }
  if (status.restartRequired) {
    return `客户端仍使用 ${status.address}。重启后改为 ${status.savedAddress}。`;
  }
  return "";
}
