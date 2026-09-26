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
- **上游**: Codex、Grok、Claude、Gemini、Kimi、Devin、Meta。
- **客户端密钥**: 调用本服务的调用方凭证。它不是上游账号的密钥。
- **监听配置**: host、端口和 TLS。
- **已绑定的监听**: 当前这次成功启动实际在听的 host、端口和 TLS。
- **热重载**: 服务运行期间，监听配置以外的已保存变更无需重启即可生效。
- **需要重新授权**: 运行中的 SDK 将某账号判为 `unauthorized` 或 `invalid_grant` 的认证错误。
- **界面语言**: 应用自有文案的简体中文或英语呈现，不改变配置值及上游返回内容。

## Observable contracts

### 界面状态动效

- 首页主要状态、新增账号卡片、两项自定义设置字段，以及更新对话框的阶段说明可以用短暂动效提示状态变化；相同状态的后台刷新不得重播。
- 动效不延迟操作。隐藏中的设置字段立即不可操作或聚焦。系统要求减少动态效果时，不移动或缩放内容，仅保留简短淡入。详见 `docs/prd/interface-state-motion.md` 和 `docs/adr/interface-state-motion.md`。

### Documentation harness

- New product behavior is defined in `docs/prd/` before feature code lands.
- Material technical choices are recorded in `docs/adr/` before or with the code that depends on them.
- Stable, implementation-independent rules merge into this `docs/spec.md`.
- Feature work that changes terminology, contracts, invariants, or failure behavior updates this file in the same change set.
- Agents must not implement feature code during 质问; the accepted PRD/ADR/spec set is the source of truth for implementation.

### 本地代理

- 默认入口为 `/` 首页。无上游登录账号或非空上游 API key 时展示 provider 导航卡片。已有上游凭据但服务未运行时，内容区只有居中空状态：播放图标、标题「启动服务」、一句只提示去右上角启动的描述，以及描述旁指向窗口右上角的手绘箭头。箭头不在文字里。没有分段。服务运行且存在任意上游凭据时，用分段切换「接入」和「测试」，默认是「接入」。客户端密钥不算上游凭据。
- 「接入」展示当前绑定地址、可选择的客户端密钥、一份不含 `export` 的 .env（`OPENAI_API_KEY` 和 `OPENAI_BASE_URL`），以及一张可切换 SDK 的示例卡片。默认选第一把密钥；所选密钥失效时回退到首项。固定窗口内，地址卡片比密钥卡片窄，显示时能看完整把生成的客户端密钥。「接入」的密钥卡片、.env 和 SDK 示例默认用八个圆点遮挡；密钥卡片标题区的显示/隐藏按钮统一控制这三处可见值，切换所选密钥后重新遮挡。「测试」展示 models、chat completions、responses 三个 curl 示例。复制内容是纯文本；其中包含密钥的内容使用当前所选的真实客户端密钥。模型读取失败时使用明确的模型占位符，不自动启动服务。命令展示按 shell 语法高亮。详见 `docs/prd/home.md`。
- 主窗口固定为 1120×760，不支持拖动调整大小、最大化或进入全屏；仍可移动、最小化和关闭。

- 侧边栏依次为「首页」、Codex、Grok、Claude、Gemini、Kimi、Devin、Meta。「设置」固定在侧边栏底部，不随上面的导航滚动。没有原始配置编辑页，也没有自定义 OpenAI 页。
- 导航栏左侧是 akproxy。右侧用圆点表示服务是否在运行，并只有一个服务控制。未运行时为「启动」；运行中且已写入监听等于已绑定监听时为「停止」；运行中且二者不同时为「重启」，下一行写明当前地址和重启后将使用的地址。启动或重启失败后服务停止，该行显示失败原因。启动成功后提示服务已启动，并说明可以到首页看接入方式；人已经在首页，或首页还没有接入命令时，也同样提示。停止和重启成功不出现这条提示。
- 设置在可以写入时立即写入。没有保存按钮，也没有未保存离开拦截。启动、停止和重启只使用已经写入的配置。
- 成功写入的配置至少包含一把客户端密钥，且账号目录是应用自己的账号目录。违反任一条的写入不改文件，也不改变正在运行的服务。
- 写入不得删除该页不编辑的已有配置。Codex、Grok、Claude、Gemini、Kimi、Devin、Meta 没有上游 API key 表单。配置文件里的 OpenAI 兼容项没有编辑页，保存其他页面时不得删除它们。
- 监听配置只在启动或重启成功后成为已绑定监听。其他已写入配置在服务运行中热重载。
- 浏览器登录写入应用自己的账号目录。界面不展示令牌。登录或重新授权成功后，主窗口会显示并成为当前窗口，包括它被最小化或挡在其他窗口后面的情况；这次成功会刷新对应页面的账号列表和额度。失败或取消不改变窗口层级。详见 `docs/prd/local-proxy.md` 和 `docs/adr/login-success-window.md`。
- Meta 登录和重新授权先在应用弹窗显示验证码并自动复制；弹窗保留手动复制及用户主动打开系统浏览器的链接。取消或失败不保存账号；成功后沿用其他上游的账号和窗口行为。详见 `docs/prd/meta-oauth.md`。
- 服务运行时，账号列表按 SDK 实时认证状态为需要重新授权的账号显示红色提醒和对应卡片的重新授权操作。重新授权仅能更新被选中的同一账号；登录到不同账号时不写入。服务未运行时不显示历史提醒。其他 SDK 错误或额度读取失败不等于需要重新授权。详见 `docs/prd/account-reauthorization.md` 和 `docs/adr/account-auth-status.md`。
- 没有已登录账号的 Codex、Grok、Claude、Gemini、Kimi、Devin、Meta 页，在内容区中央显示该页图标、名称和添加操作。读取账号失败时不显示成空列表。
- Codex、Claude、Grok、Gemini、Kimi、Devin、Meta 账号卡片显示该账号的套餐名（有数据时）和厂商会话额度百分比窗口：Codex 按 ChatGPT 返回的周期显示 5 小时、每周或每月窗口，Claude 来自 Anthropic 的 5 小时、每周，以及有数据时的每周 Opus 和每周 Sonnet，Grok 来自 xAI 的周窗口，Gemini 来自 Antigravity 的 Gemini 池与非 Gemini 池的 5 小时和每周窗口，Kimi 的中国站和国际站来自 Coding Plan 的 5 小时、每周和有数据时的每月窗口，没有可用窗口时套餐名显示为「免费版」，其下说明没有 5 小时和每周额度，Devin 来自 GetUserStatus 的每日和每周窗口，Meta 来自 Muse 订阅接口的滚动和每周窗口。卡片标题空间不足时可省略账号邮箱，套餐名须完整显示。额度读取用该账号自己的登录令牌，令牌不出现在界面上。读取失败只影响这一张卡片。开发构建和生产构建均不注入示例数据。设置 → 用量的「额度显示」开关关闭后停止额度读取与刷新、隐藏「刷新额度」按钮；开关默认开，改动立即生效，重启后保持。详见 `docs/prd/provider-usage.md`、`docs/adr/provider-usage.md`、`docs/adr/gemini-quota.md` 和 `docs/adr/kimi-quota.md`。
- 后端操作和查询失败用 toast 展示原因；同一查询连续失败期间只提示一次，成功后再次失败可以重新提示。服务意外退出时用 toast 报告原因。
- 应用进程结束时，服务不再监听。
- 额度偏好升级时补齐缺失的默认值，保留明确的用户选择。缺失额度数据不得显示为零用量或额度耗尽；无可用窗口时按平台约定显示无窗口或读取失败。
- 账号用量卡片可显示厂商明确返回的额外额度与重置次数。缺失附加指标不显示假零值。设置的「用量」页可选择已用/剩余百分比、倒计时/具体时间、使用节奏，并分别控制附加行可见性；关闭额度显示时子选项禁用并保留原值，偏好即时生效、重启保持。详见 `docs/prd/provider-usage.md`。
- 额度显示开启时，有已保存账号的 Codex、Claude、Grok、Gemini、Kimi、Devin、Meta 额度在应用运行期间独立于页面导航定期读取；切换页面立即显示已读到的结果，首次请求未完成时才显示读取状态。关闭开关后停止后台额度请求。
- 设置页的语言可选跟随系统、简体中文、英语；外观可选跟随系统、浅色、深色。两项默认跟随系统、立即生效、重启保持。系统语言不受支持时回退简体中文。只翻译应用自有文案。外观里的直播模式默认关闭；打开后服务商账号卡片和删除确认中的邮箱显示为固定打码，不改账号数据。详情见 `docs/prd/language-appearance.md`。
- 设置页有「命令行」标签。随附的命令行程序默认装进当前用户的 PATH，不请求管理员权限。应用每次启动比对已安装命令的 `--version` 和当前应用版本；一致则保留，对不上就删掉再装。开发版本每次启动都覆盖安装。安装或移除失败时 toast 报错，并有按钮打开这个标签。关掉开关后不再安装，并移除这次加上的命令。详见 `docs/prd/cli.md` 和 `docs/adr/cli-path-install.md`。
- `akproxy model` 用配置里的代理地址和第一把客户端密钥读取模型列表，并用 gum choose 的默认样式让用户选一个。确认后写入数据目录 `cli.json` 的 `model.id`。取消或失败不改已有选择，也不启动代理。`akproxy claude`、`codex`、`opencode`、`pi` 在没有模型 id 时失败并提示先运行 `akproxy model`。没有应用配置时提示先打开 akproxy。这四条命令用所选模型和第一把客户端密钥启动对应程序，参数原样往后传，不改用户原来的配置，也不在找不到程序时自动安装。详见 `docs/adr/cli-launch-env.md` 和 `docs/adr/cli-model-picker.md`。

## System-wide constraints

### 应用更新

- 正式版本按更新偏好检查正式更新，忽略预发布和草稿；开发版本不更新自身。默认启动及每 24 小时检查一次。偏好默认值为自动更新开、自动检查开、间隔 24 小时。
- 更新偏好（自动更新、自动检查更新、检查间隔）存在应用自己的偏好文件，不进 CLIProxyAPI 的 config.yaml；改动立即生效，重启后保持。
- 语言与外观偏好也存在该偏好文件，不进入 CLIProxyAPI 的 config.yaml；旧文件缺失字段时使用跟随系统。
- 自动检查更新关闭时，启动与周期检查都不发生，只剩手动检查。自动更新关闭时，自动检查发现新版本只提示并提供手动下载，不自动下载。手动检查始终检查并直接下载。
- 设置页提供「关于」标签：头部信息区含应用图标、名称、版本、平台架构、上次检查结果与时间、手动检查入口；发现新版本时头部显示可点开对话框的提示。另含更新偏好与链接；链接在用户的默认浏览器中打开。
- 关于页仅在 `dev` 版本旁显示该二进制的短提交 hash；缺失时显示不可用，不从运行时仓库推断。正式版本不显示 hash。
- 采用 Wails 更新器下载和校验。检查中和已是最新不弹窗；有新版本、失败或可以重启时才提示。失败弹窗可「重试」重放失败前的动作。用户点击重启后安装，代理不自动恢复。
- 更新不得改变配置和账号。校验信息缺失或不匹配时禁止安装。
- tag 构建生成草稿 Release，维护者手动发布后才向客户端提供更新。
- macOS 使用 akproxy 独立的证书、私钥和公证凭据。产品及架构见 `docs/prd/app-updates.md`、`docs/prd/about.md`、`docs/adr/wails-updater.md`、`docs/adr/update-prefs.md`。
- macOS 首次安装的 DMG 打开后是拖放窗口：背景为固定品牌图，箭头左侧是 `akproxy.app`，右侧是指向本机「应用程序」文件夹的替身。说明文字不随界面语言变化。见 `docs/prd/dmg-install.md` 和 `docs/adr/dmg-drag-install.md`。

### 工程约束

- 桌面壳使用 Wails v3，迁移范围与兼容要求见 `docs/prd/wails-v3.md` 和 `docs/adr/wails-v3.md`。
- Repository agent entrypoint is root `AGENTS.md` (`CLAUDE.md` is a symlink to it).
- Feature development workflow skill lives at `.agents/skills/feature-dev/` (also linked from `.claude/skills/`).
- Commit attempts should re-check the working tree against this specification and relevant PRDs/ADRs before landing.
- 配置与账号的事实来源、进程内 SDK，以及监听重启和热重载的分界，分别见 `docs/adr/config-file-source-of-truth.md`、`docs/adr/embed-go-sdk.md` 和 `docs/adr/listen-restart.md`。
- 窗口渲染使用 React 和 coss ui，见 `docs/adr/coss-renderer.md`。首页命令高亮见 `docs/adr/home-command-highlight.md`。已有凭据但未运行时的首页空状态见 `docs/adr/home-stopped-empty.md`。

## Current implementation status

- Documentation harness directories and agent workflow files were installed by `hnm init`.
- 本地代理的产品契约和上述架构决定已经写入文档。
- 窗口、配置读写和内嵌服务的代码已经按该契约实现。`go test ./internal/desktop/` 覆盖配置不变量；渲染层可以由 `pnpm --dir frontend run build` 构建。
