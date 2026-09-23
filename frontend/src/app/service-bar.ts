import type { Status } from "@/lib/desktop";
import { localizeErrorMessage, translate, type Locale } from "../lib/i18n.ts";

export function controlSuccessToast(action: Status["action"], locale: Locale = "zh-CN"): { title: string; description: string } | null {
  if (action !== "start") {
    return null;
  }
  return { title: translate("服务已启动", locale), description: translate("可以到首页看接入方式", locale) };
}

// The navbar's second line is one message: a start failure, or the two
// addresses the user must compare before restarting.
export function serviceBarNotice(status: Status | undefined, locale: Locale = "zh-CN"): string {
  if (!status) {
    return "";
  }
  if (status.error) {
    return localizeErrorMessage(status.error, locale);
  }
  if (status.restartRequired) {
    return translate("客户端仍使用 {address}。重启后改为 {savedAddress}。", locale, { address: status.address, savedAddress: status.savedAddress });
  }
  return "";
}
