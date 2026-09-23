# macOS 拖放安装

## 问题与用户

第一次在 Mac 上安装的人打开的是 DMG，不是应用窗口。现在的 DMG 只是一个装着 `akproxy.app` 的磁盘，没有说明要把它拖进「应用程序」。

## 流程

挂载 `akproxy.dmg` 后，Finder 打开一个固定大小的窗口。背景是品牌图：上方是应用图标和 akproxy，下面一句英文说明 `To install, drag into "Applications folder"`，再下面是一块浅色圆角区域，中间有向右的箭头。

箭头左边是 `akproxy.app`，右边是指向本机 `/Applications` 的替身，名称是 Applications。把应用拖到这个替身上，就装进「应用程序」。

这张背景是安装图，不跟随应用里的语言或外观设置。

## 非目标

不改变应用内的更新安装。不在 DMG 里放许可证、背景音乐或其他文件。不把 Windows 或 Linux 的压缩包改成安装器。

## 验收

1. 打开 DMG 能看到上述背景，而不是默认的白底图标视图。
2. 窗口里只有应用和 Applications 替身，分别在箭头两侧，图标落在浅色区域内。
3. 把 `akproxy.app` 拖到 Applications 上，会放进本机的「应用程序」文件夹。
