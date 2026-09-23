import assert from "node:assert/strict";
import test from "node:test";
import { localizeErrorMessage, resolveDark, resolveLocale, translate } from "./i18n.ts";

test("language and theme choices follow the system with Chinese fallback", () => {
  assert.equal(resolveLocale("system", "en-US"), "en");
  assert.equal(resolveLocale("system", "fr-FR"), "zh-CN");
  assert.equal(resolveLocale("zh-CN", "en-US"), "zh-CN");
  assert.equal(resolveDark("system", true), true);
  assert.equal(resolveDark("light", true), false);
  assert.equal(resolveDark("dark", false), true);
});

test("translated text interpolates values and preserves external error details", () => {
  assert.equal(translate("客户端仍使用 {address}。重启后改为 {savedAddress}。", "en", { address: "old", savedAddress: "new" }), "Clients still use old. Restart to use new.");
  assert.equal(localizeErrorMessage("登录失败: upstream rejected token", "en"), "Sign-in failed: upstream rejected token");
  assert.equal(localizeErrorMessage("读取模型列表失败（HTTP 503）", "en"), "Model list request failed (HTTP 503)");
  assert.equal(localizeErrorMessage("登录失败: upstream rejected token", "zh-CN"), "登录失败: upstream rejected token");
});
