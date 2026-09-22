# 迁移到固定版本 Wails v3

状态：已接受

## 背景与决定

Wails 官方应用内更新能力位于 v3，用户已接受框架迁移并要求优先独立提交。固定 Go 模块与 CLI 为 `v3.0.0-beta.24`，前端 runtime 为 `3.0.0-beta.24`。

使用 v3 application、service 生命周期及生成的 TypeScript 绑定，替换 v2 context runtime 和 `window.go`。保留 `internal/desktop` 的业务边界与数据路径。前端事件改用 v3 runtime，取消订阅使用注册时返回的清理函数。

构建采用精简 Taskfile 与现有 DMG 打包脚本，macOS `.app` 保持 `com.wails.akproxy` 标识。最低系统版本按 v3 官方构建模板设为 macOS 12，编译与 plist 使用同一目标。本地开发打包仅使用 ad-hoc 签名；独立 Developer ID 材料与正式发布由后续更新功能处理。

## 取舍

保留 v2 无法直接采用 v3 官方更新服务。迁移 beta 框架扩大回归范围，因此固定版本，并验证原有服务、绑定、窗口和构建。此次不混入更新器代码或发布流程扩展。

## 验证

`go test ./...`、`pnpm --dir frontend run build`、`wails3 task package`；macOS 真实窗口与主窗口关闭的退出路径。
