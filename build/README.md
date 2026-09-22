# 构建资源

- `config.yml`：Wails v3 开发模式，启动 Vite 并监听 Go 代码变化。
- `appicon.png`：应用图标源文件。
- `darwin/Info.plist`：macOS bundle 元数据，保留既有 `com.wails.akproxy` 标识。
- `windows/`：原有 Windows 图标及安装器资源，尚未接入新的发布工作流。
- `bin/`：被 Git 忽略的构建输出，包含二进制、生成图标、macOS `.app` 和 DMG。

`wails3 task build` 调用 `scripts/package-app.sh` 组装 macOS bundle；`wails3 task package` 再用 `scripts/package-dmg.sh` 生成 DMG。源码中的 plist 不再是 v2 模板，不能通过删除来重置。
