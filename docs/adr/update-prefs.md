# 应用自管的更新定时器与独立偏好文件

状态：已接受

## 背景

PRD `docs/prd/about.md` 要求自动检查可开关、间隔可调且改动立即生效。现有实现把周期任务交给 Wails 更新器的 `Config.CheckInterval`（固定 24 小时），并把偏好寄存在 CLIProxyAPI 的 `config.yaml`。

两个硬约束来自代码与依赖：

- `updater.Init` 只能调用一次（再次调用返回 `ErrAlreadyConfigured`）；`StopPeriodicCheck` 停掉内置周期任务后没有重启入口。周期开关和间隔因此不能靠更新器内置机制。
- `config.LoadConfig` 解析上游 `config.Config` 结构，未知字段被丢弃；写入时 `SaveConfigPreserveComments` 也不会保留它们。应用自己的偏好不能放进该文件。

另一个已确认的事实：更新器 `Check()` 只查询并发出 `update-available` 事件，不下载；随后调用 `DownloadAndInstall()` 才下载。这对应「自动更新关＝只提示」与「自动更新开＝直接下载」两条路径。

## 决定

- 偏好存独立文件 `app.json`，位于应用配置目录（`Paths.Root` 下，与 `config.yaml` 同目录）。更新字段为 `{ "autoUpdate": bool, "autoCheck": bool, "checkIntervalHours": int }`，默认值为 `true`、`true`、`24`。后续用量及语言与外观偏好也存于该文件，见 `provider-usage.md` 和 `language-appearance.md`。文件损坏或缺字段时按默认值处理，不报错阻断启动。
- `updater.Config.CheckInterval` 恒传 0，更新器内置周期任务不再使用。应用在 `internal/desktop` 之外新建一个定时管理器（app 包内），持有 `time.Timer`：按偏好启动、停止、改间隔；从关到开或改短间隔时立即触发一次检查。
- 定时到点后按偏好分派：自动更新开 → `Check` 后 `DownloadAndInstall`；自动更新关 → 只 `Check`（发现新版本发应用自己的事件，由窗口弹「发现新版本」对话框，「立即下载」再调 `DownloadAndInstall`）。启动检查走同一条分派路径。分开调用可保留查到的版本供关于页展示和失败后重试。
- 手动检查入口仍调 `CheckUpdates()`，执行同样的检查和下载流程；手动、定时和待下载操作串行，避免并发检查破坏已下载状态。定时触发前先看更新器状态，进行中则跳过本次。
- 偏好读写走 `internal/desktop` 新的 `AppPrefs` 类型与存取函数，JSON 编解码；`app.json` 不参与 `config.yaml` 的任何校验或保存流程。
- 关于页数据：版本沿用现有 `Version()` 绑定；平台架构用 `runtime.GOOS`/`runtime.GOARCH` 经绑定返回；仓库、链接是静态内容写在前端。
- 定时器生命周期跟随应用：启动时按偏好启动，`OnShutdown` 停止。

## 备选与后果

- 继续用更新器内置周期：无法实现开关和改间隔后立即生效（`StopPeriodicCheck` 不可逆），放弃。
- 偏好放进 `config.yaml` 自定义字段：上游结构体会丢弃未知字段，需要 fork 配置解析，放弃。
- 「发现新版本」复用 `update-available` 事件：事件通路现成，对话框新增一个 `available` 阶段即可，无需新事件协议。

## 边界与验证

- `app.json` 损坏不阻断启动，按默认值运行并在下次保存时覆写。
- 定时检查失败不打开更新对话框；关于页记录上次结果，并用 toast 提示错误原因。
- `go test ./internal/desktop/` 覆盖偏好的读写、默认值与损坏文件回退；定时器分派逻辑以状态函数测试。
- 前端 `pnpm --dir frontend run build` 通过；macOS 上运行验证开关与间隔的立即生效。
