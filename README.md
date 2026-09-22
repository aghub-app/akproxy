# akproxy

[English](README.en.md)

把 Codex、Grok、Claude、Gemini、Kimi、Devin 的登录放进一个窗口，在本机开一个 OpenAI 兼容的 HTTP 服务。编程工具指到这个地址，用应用自己的客户端密钥调用。

基于 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)。账号和配置属于这个应用自己的数据目录，不接管你已经在用的 CLI 配置。

## 功能

- **一个窗口**：登录、改设置、启动、停止、重启，都在这里。界面跟随系统浅色或深色。
- **浏览器登录**：Codex、Grok、Claude、Gemini、Kimi（中文站和国际站）、Devin。登录在系统浏览器里完成，令牌不出现在界面上。
- **本机服务**：默认监听 `127.0.0.1:8317`。第一次打开会生成一把客户端密钥。
- **接入**：服务启动后，首页给出地址、密钥、`.env`（`OPENAI_API_KEY`、`OPENAI_BASE_URL`），以及 OpenAI、AI SDK、LangChain 的示例。
- **试一次**：首页的「测试」里有三个 curl。服务没启动时不能复制。
- **改完即写**：密钥、出站代理、路由马上生效。改监听地址或端口要重启后才换绑。
- **更新**：正式版会检查更新。下载和校验在后台进行，你确认后再重启安装。

## 安装

到 [Releases](https://github.com/aghub-app/akproxy/releases) 下载对应系统的包。macOS 是通用包；Windows 和 Linux 为 amd64。带签名和公证的 macOS 包只来自已发布的 Release，本地自己构建的是 ad-hoc 签名。

## 使用

1. 打开 akproxy。侧边栏选一个上游，点登录，在浏览器里完成授权。
2. 点右上角「启动」。
3. 回到首页「接入」。把 `.env` 或 SDK 示例里的地址和密钥交给编程工具。`OPENAI_BASE_URL` 带 `/v1`。
4. 要自己打一发请求，切到「测试」，复制 curl。

数据在系统的用户配置目录下的 `akproxy/`：`config.yaml` 是配置，`auths/` 是登录账号。macOS 上是 `~/Library/Application Support/akproxy`。

退出应用会停掉服务。应用不驻留菜单栏。

## 开发

需要 Go 1.26、Node.js、pnpm，以及当前系统的 Wails 原生构建依赖。Go 和前端 runtime 固定在 Wails `v3.0.0-beta.24`。

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
wails3 task dev
```

开发时页面由 Vite 提供，地址是 `http://127.0.0.1:9245`。

```sh
wails3 task build          # 绑定、前端、应用。macOS 得到 build/bin/akproxy.app
go test ./...
pnpm --dir frontend run build
wails3 task package        # macOS 另生成 build/bin/akproxy.dmg
wails3 task bindings       # 只生成 TypeScript 绑定
```

全新 checkout 要先构建前端，Go 才能嵌入 `frontend/dist`。本地默认版本是 `dev`，不检查更新。

推送 `vMAJOR.MINOR.PATCH` tag 会构建 macOS 通用包以及 Windows、Linux 的 amd64 包，成功后开一个草稿 Release。发布和公证见 [docs/releasing.md](docs/releasing.md)。

产品约定见 [docs/prd/local-proxy.md](docs/prd/local-proxy.md) 和 [docs/spec.md](docs/spec.md)。

## 致谢

核心代理是 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)。akproxy 只做本机窗口、登录和这份配置的编辑。
