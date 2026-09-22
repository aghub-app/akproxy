# akproxy

基于 Wails v3、React 和 coss UI 的本地 CLIProxyAPI 桌面应用。

## 开发

需要 Go 1.26、Node.js、pnpm，以及平台的 Wails 原生构建依赖。

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
wails3 task dev
```

开发模式使用 Vite，地址为 `http://127.0.0.1:9245`。Go/CLI 与前端 runtime 固定同一版本。

## 构建和验证

```sh
wails3 task build
go test ./...
wails3 task package
```

`build` 会生成 TypeScript 绑定、用 pnpm 构建前端，并编译应用。macOS 输出 `build/bin/akproxy.app`；`package` 另生成 `build/bin/akproxy.dmg`。本地默认版本为 `dev`，使用 ad-hoc 签名，不执行自动更新。全新 checkout 要先构建前端，Go 才能嵌入 `frontend/dist`。

单独生成绑定：`wails3 task bindings`。单独验证前端：`pnpm --dir frontend run build`。

推送 `vMAJOR.MINOR.PATCH` tag 会构建 macOS 通用包及 Windows/Linux amd64 包，全部成功后生成草稿 Release，手动发布后才提供更新。macOS 发布要求专用 Developer ID 和公证凭据；配置步骤与测试方法见 [发布指南](docs/releasing.md)。

正式版启动及每 24 小时检查稳定更新，设置页也可手动检查。采用 Wails 默认窗口自动下载、校验，用户点击重启后安装；配置和账号保留，代理不自动恢复。

产品约定见 [本地代理 PRD](docs/prd/local-proxy.md)、[应用更新 PRD](docs/prd/app-updates.md)，框架迁移见 [Wails v3 ADR](docs/adr/wails-v3.md)。
