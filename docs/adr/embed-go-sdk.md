# 在 Wails 进程内嵌入 CLIProxyAPI SDK

状态：已接受

## 背景

应用要编辑 CLIProxyAPI 的配置并启动它的 HTTP 服务，还要打开系统浏览器完成上游登录。CLIProxyAPI 把这些能力放在 Go SDK 里。桌面壳曾经在 Wails 和 Tauri 之间选择。

## 决定

用 Wails 做窗口。在同一个进程里嵌入 `github.com/router-for-me/CLIProxyAPI/v8` 的公开 SDK，由这个进程启动、热重载和停止 HTTP 服务，并调用 SDK 的浏览器登录。

升级到 v8.0.20，以接入 xAI OAuth 请求的 Grok CLI 版本更新。旧版 SDK 固定发送 `0.2.120`，已被上游以 HTTP 426 拒绝；更新本机 Grok CLI 不会改变内嵌 SDK 的请求头。嵌入服务负责后台获取当前版本，并在获取失败时使用 SDK 的回退版本，不在桌面应用里复制版本更新逻辑。

登录注册这份产品要用的认证器：Codex、xAI（Grok）、Claude、Antigravity（Gemini 页）、Kimi 的 kimi.com 与 kimi.ai、Devin。Meta 的设备码需要展示在桌面弹窗，按 `meta-device-oauth.md` 在应用内适配协议，仍由 SDK 管理账号与代理。

## 备选

- Tauri 窗口加一个 Go 侧车进程。服务和登录要跨进程传配置、状态和取消，窗口退出时还要另管侧车生命周期。
- 下载并拉起 CLIProxyAPI 官方二进制。应用会依赖另一份发布物，也绕开了 SDK 的嵌入方式。
- 固定在文档示例里的 v6。当前登录和上游类型以 v8 为准，v6 对不上这份产品选中的提供商。

## 后果

- 服务的生命周期等于应用进程。退出应用必须取消服务。这满足「退出后不再监听」。
- SDK 要求 Go 1.26 工具链。本机若是更早的 Go，由工具链自动下载，不把模块降到 v6。
- 外部模块使用 `sdk/config`、`sdk/cliproxy` 和 `sdk/auth`。不把 `internal/` 当作本应用的导入路径。
- 官方 SDK 文档仍有 v6 和 `internal/config` 的示例。以所锁定 v8 版本的公开包为准。
- 管理面板资源不作为本应用的界面。默认配置关闭面板下载。管理密钥为空时，SDK 不挂管理接口。

## 验证

- 服务启动、停止和登录都走进程内 SDK，不产生第二个代理进程。
- 构建使用 v8 模块。
- 用本地模拟上游经过内嵌 xAI 执行器验证 Grok CLI 版本不低于 `1.0.13`、User-Agent 版本一致，以及 chat completions 返回助手文本。
