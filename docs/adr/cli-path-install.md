# 把随附的命令行装进用户 PATH

状态：已接受

## 背景

`akproxy claude` 这类命令要在终端里能敲到。桌面进程从访达打开时，看不到用户终端的 PATH。`/usr/local/bin` 在 macOS 上经常需要管理员权限。产品要求无感安装、失败只 toast、用户关掉功能后移除，并且直接覆盖已有的同名命令。

命令行是单独的程序。macOS 包把它放在 `akproxy.app/Contents/MacOS/akproxy-cli`。Windows 安装包把它放在应用目录的 `akproxy-cli.exe`。开发构建放在 GUI 可执行文件旁边。

## 决定

- 偏好 `cliEnabled` 存在 `app.json`，默认 true。旧文件缺这个字段时视为开。
- 每次应用启动，开着就检查已安装的 `akproxy --version`。和当前应用版本一致时只补 PATH，不换二进制。对不上、命令不在或版本读不出来时删掉再装。开发版本（`dev`）每次启动都覆盖安装。设置里的开关立刻做同样的动作。
- 安装位置是用户主目录的 `.local/bin/akproxy`（Windows 为 `akproxy.exe`）。不请求管理员权限。需要重装时直接替换该位置已有的命令。
- macOS 在 `~/.zprofile` 和 `~/.zshrc` 写入同一段带标记的 PATH。Linux 写 `~/.profile` 和 `~/.bashrc`。Windows 改当前用户的 PATH 注册表。标记不存在才补，避免重复。
- 用 `.local/bin/.akproxy-cli` 标记这次安装。移除时只删标记指向的命令和我们写入的 PATH 段。没有标记时不删用户自己的文件。
- 找不到随附二进制、目录写不了或 PATH 写不了时，启动不中断。界面 toast 错误，并提供进入设置「命令行」页的按钮。该页用同一句错误和开关让用户关掉或再试。

## 备选

- 链接到 `/usr/local/bin`。终端默认就能找到，但经常要管理员密码，放弃。
- 只在应用自己的进程 PATH 里放命令。访达启动的进程改不了已经打开的终端，放弃。
- 安装失败就弹系统对话框。产品要求 toast，放弃。

## 后果

- 新开的终端才能看到 PATH 变化。已经开着的终端要重新加载 shell 配置。
- 覆盖会换掉 `.local/bin` 里原来的 `akproxy`。关掉功能只能恢复我们装上的那一个，不能找回被覆盖前的文件。
- Windows 安装包必须带上 `akproxy-cli.exe`，否则每次启动都会报找不到随附程序。

## 验证

- 临时主目录上的安装、重复安装、覆盖、版本一致时保留、版本不一致和开发版本时替换、关闭后移除，以及没有标记时保留外来文件，由 `go test ./internal/desktop/` 覆盖。
