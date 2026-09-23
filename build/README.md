# 构建资源

- `config.yml`：Wails v3 开发模式，启动 Vite 并监听 Go 代码变化。
- `akproxy.icon`：Icon Composer 源文件。替换图标时改这一份。
- `appicon.png`、`icon-round.png`：从 `akproxy.icon` 导出的 1024×1024 macOS Default 图，圆角外透明。Dock 运行时图标、README、关于页（`frontend/src/assets/images/appicon.png`，直接复制）、macOS `.icns` 和 Windows `.ico` 都用这一张。GitHub 会去掉 HTML 的样式，圆角做在图片上。
- 导出并生成各平台图标：

  ```sh
  "/Applications/Icon Composer.app/Contents/Executables/ictool" build/akproxy.icon \
    --export-image --output-file build/appicon.png \
    --platform macOS --rendition Default --width 1024 --height 1024 --scale 1
  cp build/appicon.png build/icon-round.png
  cp build/appicon.png frontend/src/assets/images/appicon.png
  wails3 generate icons -input build/appicon.png -macfilename build/bin/iconfile.icns -windowsfilename build/windows/icon.ico
  ```
- `dmg-background.png`：macOS 拖放安装窗口的背景。用 `python3 scripts/render-dmg-background.py` 从 `appicon.png` 重画。图标坐标在这个脚本和 `scripts/package-dmg.sh` 里要一致。
- `darwin/Info.plist`：macOS bundle 元数据，保留既有 `com.wails.akproxy` 标识。
- `windows/`：原有 Windows 图标及安装器资源，尚未接入新的发布工作流。
- `bin/`：被 Git 忽略的构建输出，包含二进制、生成图标、macOS `.app` 和 DMG。

`wails3 task build` 调用 `scripts/package-app.sh` 组装 macOS bundle；`wails3 task package` 再用 `scripts/package-dmg.sh` 生成 DMG。源码中的 plist 不再是 v2 模板，不能通过删除来重置。

正式发布由 `scripts/release-macos.sh` 生成 macOS 通用、签名并公证的更新 ZIP；Windows/Linux 发布压缩后的可执行文件，不生成原有 NSIS 安装器。详见 [发布指南](../docs/releasing.md)。
