# Specification

## Product scope

akproxy 是本机上的一个窗口。它保管自己的一份 CLIProxyAPI 配置，用表单编辑这份配置，并启动、停止或重启内嵌的 HTTP 服务。应用里没有原始配置编辑器。产品要求见 `docs/prd/local-proxy.md`。

Out of scope until explicitly specified: anything not yet accepted in a PRD.

## Terminology

- **Feature 质问**: the mandatory product-then-technical clarification loop driven by `$feature-dev` before implementation.
- **PRD**: a product requirements document under `docs/prd/` describing problem, users, goals, non-goals, flows, failure behavior, and acceptance criteria.
- **ADR**: an architecture decision record under `docs/adr/` capturing one material technical choice, alternatives, and consequences.
- **Spec**: this file — the single source of truth for shared terminology, observable contracts, and system-wide invariants.
- **服务**: 本应用启动的 HTTP 代理进程内服务。
- **上游**: Codex、Grok、Claude、Gemini、Kimi、Devin。
- **客户端密钥**: 调用本服务的调用方凭证。它不是上游账号的密钥。
- **监听配置**: host、端口和 TLS。
- **已绑定的监听**: 当前这次成功启动实际在听的 host、端口和 TLS。
- **热重载**: 服务运行期间，监听配置以外的已保存变更无需重启即可生效。

## Observable contracts

### Documentation harness

- New product behavior is defined in `docs/prd/` before feature code lands.
- Material technical choices are recorded in `docs/adr/` before or with the code that depends on them.
- Stable, implementation-independent rules merge into this `docs/spec.md`.
- Feature work that changes terminology, contracts, invariants, or failure behavior updates this file in the same change set.
- Agents must not implement feature code during 质问; the accepted PRD/ADR/spec set is the source of truth for implementation.

### 本地代理

- 默认入口为 `/` 首页。无上游登录账号或非空上游 API key 时展示 provider 导航卡片；存在任意上游凭据时用分段切换「接入」和「测试」，默认是「接入」。客户端密钥不算上游凭据。
- 「接入」只在服务运行时展示当前绑定地址、第一把客户端密钥、一份不含 `export` 的 .env（`OPENAI_API_KEY` 和 `OPENAI_BASE_URL`），以及一张可切换 SDK 的示例卡片。密钥默认遮挡。未启动时不展示这些信息。「测试」展示 models、chat completions、responses 三个 curl 示例；未启动时不能复制。复制内容是纯文本，包含真实客户端密钥。模型读取失败或服务停止时使用明确的模型占位符，不自动启动服务。命令展示按 shell 语法高亮。详见 `docs/prd/home.md`。

- 侧边栏依次为「首页」、Codex、Grok、Claude、Gemini、Kimi、Devin。「设置」固定在侧边栏底部，不随上面的导航滚动。没有原始配置编辑页，也没有自定义 OpenAI 页。
- 导航栏左侧是 akproxy。右侧用圆点表示服务是否在运行，并只有一个服务控制。未运行时为「启动」；运行中且已写入监听等于已绑定监听时为「停止」；运行中且二者不同时为「重启」，下一行写明当前地址和重启后将使用的地址。启动或重启失败后服务停止，该行显示失败原因。启动成功后提示服务已启动，并说明可以到首页看接入方式；人已经在首页，或首页还没有接入命令时，也同样提示。停止和重启成功不出现这条提示。
- 设置在可以写入时立即写入。没有保存按钮，也没有未保存离开拦截。启动、停止和重启只使用已经写入的配置。
- 成功写入的配置至少包含一把客户端密钥，且账号目录是应用自己的账号目录。违反任一条的写入不改文件，也不改变正在运行的服务。
- 写入不得删除该页不编辑的已有配置。Codex、Grok、Claude、Gemini、Kimi、Devin 没有上游 API key 表单。配置文件里的 OpenAI 兼容项没有编辑页，保存其他页面时不得删除它们。
- 监听配置只在启动或重启成功后成为已绑定监听。其他已写入配置在服务运行中热重载。
- 浏览器登录写入应用自己的账号目录。界面不展示令牌。
- 没有已登录账号的 Codex、Grok、Claude、Gemini、Kimi、Devin 页，在内容区中央显示该页图标、名称和添加操作。读取失败时不显示成空列表。
- Codex 和 Claude 账号卡片显示该账号的厂商会话额度百分比。额度读取失败只影响这一张卡片。其他平台不显示额度。
- 应用进程结束时，服务不再监听。

## System-wide constraints

### 应用更新

- 正式版本按更新偏好检查正式更新，忽略预发布和草稿；开发版本不更新自身。默认启动及每 24 小时检查一次。偏好默认值为自动更新开、自动检查开、间隔 24 小时。
- 更新偏好（自动更新、自动检查更新、检查间隔）存在应用自己的偏好文件，不进 CLIProxyAPI 的 config.yaml；改动立即生效，重启后保持。
- 自动检查更新关闭时，启动与周期检查都不发生，只剩手动检查。自动更新关闭时，自动检查发现新版本只提示并提供手动下载，不自动下载。手动检查始终检查并直接下载。
- 设置页提供「关于」标签：头部信息区含应用图标、名称、版本、平台架构、上次检查结果与时间、手动检查入口；发现新版本时头部显示可点开对话框的提示。另含更新偏好与链接。
- 采用 Wails 更新器下载和校验。检查中和已是最新不弹窗；有新版本、失败或可以重启时才提示。失败弹窗可「重试」重放失败前的动作。用户点击重启后安装，代理不自动恢复。
- 更新不得改变配置和账号。校验信息缺失或不匹配时禁止安装。
- tag 构建生成草稿 Release，维护者手动发布后才向客户端提供更新。
- macOS 使用 akproxy 独立的证书、私钥和公证凭据。产品及架构见 `docs/prd/app-updates.md`、`docs/prd/about.md`、`docs/adr/wails-updater.md`、`docs/adr/update-prefs.md`。

### 工程约束

- 桌面壳使用 Wails v3，迁移范围与兼容要求见 `docs/prd/wails-v3.md` 和 `docs/adr/wails-v3.md`。
- Repository agent entrypoint is root `AGENTS.md` (`CLAUDE.md` is a symlink to it).
- Feature development workflow skill lives at `.agents/skills/feature-dev/` (also linked from `.claude/skills/`).
- Commit attempts should re-check the working tree against this specification and relevant PRDs/ADRs before landing.
- 配置与账号的事实来源、进程内 SDK，以及监听重启和热重载的分界，分别见 `docs/adr/config-file-source-of-truth.md`、`docs/adr/embed-go-sdk.md` 和 `docs/adr/listen-restart.md`。
- 窗口渲染使用 React 和 coss ui，见 `docs/adr/coss-renderer.md`。首页命令高亮见 `docs/adr/home-command-highlight.md`。

## Current implementation status

- Documentation harness directories and agent workflow files were installed by `hnm init`.
- 本地代理的产品契约和上述架构决定已经写入文档。
- 窗口、配置读写和内嵌服务的代码已经按该契约实现。`go test ./internal/desktop/` 覆盖配置不变量；渲染层可以由 `pnpm --dir frontend run build` 构建。
