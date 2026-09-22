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
- **上游**: Codex、Grok、Claude、Gemini、Kimi、Devin，或一个自定义 OpenAI 兼容端点。
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

- 侧边栏只有 Codex、Grok、Claude、Gemini、Kimi、Devin、「自定义 OpenAI」和「设置」。「设置」在最下面。没有原始配置编辑页。
- 导航栏右侧只有一个服务控制。未运行时为「启动」；运行中且已写入监听等于已绑定监听时为「停止」；运行中且二者不同时为「重启」，并提示可以考虑重启。
- 设置和自定义 OpenAI 在可以写入时立即写入。没有保存按钮，也没有未保存离开拦截。启动、停止和重启只使用已经写入的配置。
- 成功写入的配置至少包含一把客户端密钥，且账号目录是应用自己的账号目录。违反任一条的写入不改文件，也不改变正在运行的服务。
- 写入不得删除该页不编辑的已有配置。Codex、Grok、Claude、Gemini、Kimi、Devin 没有上游 API key 表单。名称为 `kimi` 或 `kimi-ai` 的 OpenAI 兼容项只存在于配置文件；写入「自定义 OpenAI」不得删除它们。
- 监听配置只在启动或重启成功后成为已绑定监听。其他已写入配置在服务运行中热重载。
- 浏览器登录写入应用自己的账号目录。界面不展示令牌。
- Codex 和 Claude 账号卡片显示该账号的厂商会话额度百分比。额度读取失败只影响这一张卡片。其他平台不显示额度。
- 应用进程结束时，服务不再监听。

## System-wide constraints

- Repository agent entrypoint is root `AGENTS.md` (`CLAUDE.md` is a symlink to it).
- Feature development workflow skill lives at `.agents/skills/feature-dev/` (also linked from `.claude/skills/`).
- Commit attempts should re-check the working tree against this specification and relevant PRDs/ADRs before landing.
- 配置与账号的事实来源、进程内 SDK，以及监听重启和热重载的分界，分别见 `docs/adr/config-file-source-of-truth.md`、`docs/adr/embed-go-sdk.md` 和 `docs/adr/listen-restart.md`。
- 窗口渲染使用 React 和 coss ui，见 `docs/adr/coss-renderer.md`。

## Current implementation status

- Documentation harness directories and agent workflow files were installed by `hnm init`.
- 本地代理的产品契约和上述架构决定已经写入文档。
- 窗口、配置读写和内嵌服务的代码已经按该契约实现。`go test ./internal/desktop/` 覆盖配置不变量；渲染层可以由 `pnpm --dir frontend run build` 构建。
