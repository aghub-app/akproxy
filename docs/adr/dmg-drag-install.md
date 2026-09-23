# DMG 用 Finder 窗口排成拖放安装

状态：已接受

## 背景

`scripts/package-dmg.sh` 原先用 `hdiutil create -srcfolder` 直接压成只读 DMG，卷里只有 `akproxy.app`。Finder 因此用默认图标视图打开，没有安装说明。

安装窗口的样子已经由一张品牌稿定下来：深色背景、图标和名称、英文拖放说明、浅色圆角区域和居中的箭头。应用图标和 Applications 必须是真实的 Finder 图标，不能画进背景，否则无法拖放。

## 决定

背景图提交在 `build/dmg-background.png`，由 `scripts/render-dmg-background.py` 按 720×680 点、2× 重新绘制。打包时先做可写 APFS 镜像，放入 `akproxy.app`、指向 `/Applications` 的替身，以及隐藏的 `.background/background.png`。再用 Finder 把窗口设成这个尺寸的图标视图，背景用这张图，两个图标的中心放在箭头两侧（196,430 和 524,430）。窗口框架先关闭再打开，然后才写入图标位置；写在关闭前的位置会被 Finder 存低大约 45 点。

最后 `hdiutil convert` 成 UDZO。CI 仍通过 `hdiutil` 和 `scripts/package-dmg.sh build/bin/akproxy.dmg` 识别这个产物。

## 备选与后果

继续用纯目录镜像最简单，但没有拖放说明。把箭头和两个图标都画进一张图会看起来像安装窗口，却不能拖。`bless --openfolder` 可以让磁盘自动打开，但在 Apple 芯片上不受支持，所以不使用；双击 DMG 时 Finder 仍会打开带有 `.DS_Store` 的窗口。

图标坐标和背景箭头必须一起改。打包时如果已经挂着名为 akproxy 的卷，脚本停止，避免排版写到别的盘上。
