# 构建资源

- `config.yml`：Wails v3 开发模式，启动 Vite 并监听 Go 代码变化。
- `appicon.png`：应用图标源文件（直角方形）。替换图标时更新这一个文件。
- `icon-round.png`：`appicon.png` 的透明圆角版本，四角按应用图标比例（约 22.37%）切好。它是 README、关于页展示图（`frontend/src/assets/images/appicon.png`，直接复制）、macOS `.icns` 和 Windows `.ico` 的共同输入。GitHub 会去掉 HTML 的样式，圆角做在图片上。
- 替换图标后运行 `wails3 generate icons -input build/icon-round.png -macfilename build/bin/iconfile.icns -windowsfilename build/windows/icon.ico`，再把 `icon-round.png` 复制到 `frontend/src/assets/images/appicon.png`。
- `darwin/Info.plist`：macOS bundle 元数据，保留既有 `com.wails.akproxy` 标识。
- `windows/`：原有 Windows 图标及安装器资源，尚未接入新的发布工作流。
- `bin/`：被 Git 忽略的构建输出，包含二进制、生成图标、macOS `.app` 和 DMG。

`wails3 task build` 调用 `scripts/package-app.sh` 组装 macOS bundle；`wails3 task package` 再用 `scripts/package-dmg.sh` 生成 DMG。源码中的 plist 不再是 v2 模板，不能通过删除来重置。

正式发布由 `scripts/release-macos.sh` 生成 macOS 通用、签名并公证的更新 ZIP；Windows/Linux 发布压缩后的可执行文件，不生成原有 NSIS 安装器。详见 [发布指南](../docs/releasing.md)。
