# akproxy

<p align="center">
  <a href="README.en.md">English</a>
</p>

<p align="center">
  <img src="build/icon-round.png" width="128" height="128" alt="akproxy">
</p>

<p align="center">
  <a href="https://github.com/aghub-app/akproxy/releases">下载连接</a>
</p>

<p align="center">
  <a href="https://github.com/aghub-app/akproxy"><img alt="Star this repo" src="https://img.shields.io/github/stars/aghub-app/akproxy.svg?style=social&label=Star%20this%20repo"></a>
</p>

**别为同一件事付两次钱。** akproxy 把你已经有的 Codex、Claude、Gemini、Kimi、Grok、Devin 订阅，接到只认 OpenAI 接口的编程工具上。

它基于 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)。浏览器里登录，代理在本机跑起来。工具只看到一个地址和一把客户端密钥。

## 功能

- 🖥️ **桌面应用** — macOS、Windows、Linux，界面跟着系统浅色或深色走
- 🚀 **启动和停止** — 一个按钮，服务说开就开，说停就停
- 🔐 **浏览器登录** — Codex、Grok、Claude、Gemini、Kimi（中文站和国际站）、Devin
- 👥 **多个账号** — 同一个上游可以登多个号，请求按轮询分出去
- ⚡ **改完就生效** — 客户端密钥、出站代理、路由，改完即用
- 🔌 **把工具接进来** — 首页给出 `.env` 和 OpenAI、AI SDK、LangChain 的示例
- 🧪 **当场试一次** — 首页「测试」里有三个 curl，你自己试试就知道有多丝滑
- 💾 **自己的一份数据** — 配置和账号都在这个应用自己的目录里

## 安装

### 下载 Release（推荐）

1. 打开 [**Releases**](https://github.com/aghub-app/akproxy/releases)
2. 下载对应系统的包：
   - **macOS**：通用包（Apple 芯片和 Intel）
   - **Windows / Linux**：amd64
3. macOS 把 `akproxy.app` 放进「应用程序」再打开

已发布的 macOS 包会签名并公证。自己在本地构建的是 ad-hoc 签名。

### 从源码构建

见下面的[开发](#开发)。

## 使用

### 第一次打开

1. 打开 akproxy
2. 侧边栏选一个上游，点登录
3. 浏览器里完成授权，账号出现在这一页
4. 点右上角「启动」

第一次打开会生成一把客户端密钥。默认地址是 `http://127.0.0.1:8317`。

### 登录

1. 系统浏览器打开上游的登录页
2. 你在浏览器里登录
3. 回到应用，这个账号已经在列表里
4. 可以继续加同一个上游的其他账号

### 把编程工具指过来

服务在跑的时候，打开首页的「接入」：

- 复制 `.env`，或复制某一张 SDK 卡片里的示例
- `OPENAI_BASE_URL` 带 `/v1`
- 密钥默认遮住。复制出去的是真实密钥

要自己发一条请求，切到「测试」，复制 curl。

### 服务

- **启动 / 停止 / 重启**：右上角只有一个按钮，圆点表示是否在跑
- **退出应用**：服务一起停。没有菜单栏常驻

数据在系统用户配置目录的 `akproxy/` 里。macOS 是 `~/Library/Application Support/akproxy`：`config.yaml` 是配置，`auths/` 是登录账号。

## 开发

需要 Go 1.26、Node.js、pnpm，以及本机的 Wails 构建依赖。Go 和前端 runtime 固定为 Wails `v3.0.0-beta.24`。

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
wails3 task dev
```

开发时界面由 Vite 提供：`http://127.0.0.1:9245`。

```sh
wails3 task build       # 绑定、前端、应用。macOS 得到 build/bin/akproxy.app
go test ./...
wails3 task package     # macOS 另生成 DMG
```

全新 clone 要先构建前端，Go 才能嵌入 `frontend/dist`。本地版本是 `dev`，不检查更新。

推送 `vMAJOR.MINOR.PATCH` 会打 macOS 通用包和 Windows、Linux 的 amd64 包，并开一个草稿 Release。签名和公证见 [docs/releasing.md](docs/releasing.md)。

## 致谢

akproxy 建在 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 上。登录、转发和多账号是它在做。这个仓库是包住它的桌面应用。

## 支持

- [GitHub Issues](https://github.com/aghub-app/akproxy/issues)
