# 在 Wails 进程内嵌入 CLIProxyAPI SDK

状态：已接受

## 背景

应用要编辑 CLIProxyAPI 的配置并启动它的 HTTP 服务，还要打开系统浏览器完成上游登录。CLIProxyAPI 把这些能力放在 Go SDK 里。桌面壳曾经在 Wails 和 Tauri 之间选择。

## 决定

用 Wails 做窗口。在同一个进程里嵌入 `github.com/router-for-me/CLIProxyAPI/v7` 的公开 SDK，由这个进程启动、热重载和停止 HTTP 服务，并调用 SDK 的浏览器登录。

登录只注册这份产品要用的认证器：Codex、xAI（Grok）、Claude、Antigravity（Gemini 页）、Kimi 的 kimi.com 与 kimi.ai、Devin。不注册 Meta。

## 备选

- Tauri 窗口加一个 Go 侧车进程。服务和登录要跨进程传配置、状态和取消，窗口退出时还要另管侧车生命周期。
- 下载并拉起 CLIProxyAPI 官方二进制。应用会依赖另一份发布物，也绕开了 SDK 的嵌入方式。
- 固定在文档示例里的 v6。当前登录和上游类型以 v7 为准，v6 对不上这份产品选中的提供商。

## 后果

- 服务的生命周期等于应用进程。退出应用必须取消服务。这满足「退出后不再监听」。
- SDK 要求 Go 1.26 工具链。本机若是更早的 Go，由工具链自动下载，不把模块降到 v6。
- 外部模块使用 `sdk/config`、`sdk/cliproxy` 和 `sdk/auth`。不把 `internal/` 当作本应用的导入路径。
- 官方 SDK 文档仍写 v6 和 `internal/config`。以仓库 `main` 上的 v7 公开包为准。
- 管理面板资源不作为本应用的界面。默认配置关闭面板下载。管理密钥为空时，SDK 不挂管理接口。

## 验证

- 服务启动、停止和登录都走进程内 SDK，不产生第二个代理进程。
- 构建使用 v7 模块。
