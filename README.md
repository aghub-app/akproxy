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
go test ./...
wails3 task build
wails3 task package
```

`build` 会生成 TypeScript 绑定、用 pnpm 构建前端，并编译应用。macOS 输出 `build/bin/akproxy.app`；`package` 另生成 `build/bin/akproxy.dmg`。当前本地包为 ad-hoc 签名，尚未接入正式分发签名与公证。

单独生成绑定：`wails3 task bindings`。单独验证前端：`pnpm --dir frontend run build`。

推送 tag 会触发 macOS DMG 构建，产物位于 GitHub Actions artifacts。产品约定见 [本地代理 PRD](docs/prd/local-proxy.md)，框架迁移见 [Wails v3 ADR](docs/adr/wails-v3.md)。
