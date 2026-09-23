# 发布与更新验收

## akproxy 独立签名材料

不要复用其他应用的证书、私钥、p12 密码或公证密码，也不要撤销其他应用正在使用的证书。Developer ID 是团队级身份，独立材料不是 Apple 强制执行的单 App 权限隔离。

1. 打开「钥匙串访问」→「证书助理」→「从证书颁发机构请求证书」。填写开发者邮箱，常用名称用 `akproxy Developer ID`，CA 邮箱留空，选择「存储到磁盘」。这次新建 CSR 和私钥，不使用旧 CSR。
2. 到 [Apple Developer Certificates](https://developer.apple.com/account/resources/certificates/list) 点 `+`，选择 **Developer ID Application**（不是 Apple Development、Apple Distribution 或 Developer ID Installer），上传新 CSR，下载 `.cer`。若达到证书数量上限，先停下来确认，不撤销现有应用证书腾位置。
3. 双击导入 `.cer`；在「我的证书」确认该证书可展开且带有这次生成的私钥。只导出这一组为 `akproxy-developer-id.p12`，设置新的强密码。证书和私钥不能放进项目目录；密码保存在密码管理器，不发到聊天或日志里。
4. 在 Apple 账户生成一个标记为 `akproxy notarization` 的新 App 专用密码，用于公证。这是独立可撤销的凭据，不是仅有 akproxy 权限的凭据。
5. 在仓库 Settings → Secrets and variables → Actions 创建以下 **repository secrets**。不要放到 organization secrets 共享给其他仓库。

| Secret | 内容 |
| --- | --- |
| `AKPROXY_CERTIFICATE_BASE64` | 专用 p12 的单行 Base64 |
| `AKPROXY_CERTIFICATE_PASSWORD` | p12 导出密码 |
| `AKPROXY_SIGNING_IDENTITY` | 新证书的 SHA-1 指纹，避免同名证书歧义 |
| `AKPROXY_APPLE_ID` | 公证用 Apple 账户邮箱 |
| `AKPROXY_APPLE_TEAM_ID` | 该证书所属 Team ID |
| `AKPROXY_APPLE_APP_PASSWORD` | 新建的 akproxy 公证 App 专用密码 |

`security find-identity -v -p codesigning` 可查看证书指纹；必须和新 `.cer` 的指纹核对，不能只凭相同的 Developer ID 名称选择。p12 可用 `base64 -i /绝对路径/akproxy-developer-id.p12 | tr -d '\n' | gh secret set AKPROXY_CERTIFICATE_BASE64 --repo aghub-app/akproxy` 直接送入指定仓库，不打印内容。其他 secret 使用 `gh secret set NAME --repo aghub-app/akproxy` 的交互输入，不把密码放进命令行历史。

CI 将这组材料及 Apple 官方 Developer ID 中间证书导入一次性钥匙串，并临时加入用户钥匙串搜索列表以构建证书链。通用包编译前检查指定指纹是否为有效签名身份；结束时恢复原搜索列表，删除临时钥匙串、p12 和中间证书。缺少材料、签名或公证失败都会阻止草稿 Release。

仅传 `codesign --keychain` 不会把该钥匙串加入证书链搜索列表。相关流程见 [GitHub 签名指南](https://docs.github.com/en/actions/how-tos/deploy/deploy-to-third-party-platforms/sign-xcode-applications)；公开中间证书来源为 [Apple PKI](https://www.apple.com/certificateauthority/)，不修改系统信任设置。

参考：[Apple CSR 流程](https://developer.apple.com/help/account/certificates/create-a-certificate-signing-request/)、[Developer ID 证书流程与数量限制](https://developer.apple.com/help/account/certificates/create-developer-id-certificates/)。

## 发布流程

1. 完成测试并提交代码。首次发布前先配置上述 Secrets。
2. 创建并推送稳定语义版本 tag，例如 `v1.0.0`。构建版本由 tag 注入；预发布 tag 会被拒绝。
3. `release` 工作流在对应系统 runner 构建。macOS 生成 arm64/amd64 通用、Developer ID 签名且公证 stapled 的应用；Windows/Linux 生成 amd64 可执行文件。
4. 所有构建成功后汇总以下文件，生成 `SHA256SUMS`，创建同名 **草稿** Release：
   - `akproxy.dmg`：macOS 首次安装。
   - `akproxy-darwin-universal.zip`：完整 `.app` 更新包。
   - `akproxy-windows-amd64.zip`：Windows 安装/更新包。
   - `akproxy-linux-amd64.tar.gz`：Linux 安装/更新包。
5. 检查签名、公证、资产和校验文件，下载并做真实升级验收，再手动发布。更新器忽略草稿、预发布和不高于当前版本的 Release。

macOS 应用应复制到可写安装目录，不要直接从只读 DMG 运行更新。Windows/Linux 当前不做本机原生验收；Windows 未接入 Authenticode。开发版 `dev` 不执行更新。

## 本地 macOS 更新测试

`go test -race ./internal/updates ./internal/ci` 覆盖真实官方 provider/更新器的版本过滤、资产选择、缺失/错误校验和、损坏 ZIP、HTTP 错误及下载后等待重启。`go test ./...` 覆盖其余后端；`wails3 task build` 覆盖绑定、前端和桌面构建。CI YAML 使用 `actionlint .github/workflows/macos-dmg.yml` 检查。

真实窗口测试使用 `scripts/update-fixture.mjs`，仅监听 `127.0.0.1`，模拟 GitHub Releases API：

```sh
node scripts/update-fixture.mjs /绝对路径/更新包目录 0.0.2
```

目录里放 `akproxy-darwin-universal.zip`，ZIP 必须只有一个顶层 `akproxy.app`。测试时不要求通用架构，但包内版本应与 fixture 一致。加上 `18765 bad-checksum` 可模拟校验失败。

先用 `wails3 task build` 生成前端；分别编译两份测试版本，构建时注入 `-X main.version=0.0.1 -X main.updateAPIBase=http://127.0.0.1:18765`（新版改为 `0.0.2`），macOS CGO 参数沿用 Taskfile；调用 `bash scripts/package-app.sh 版本号` 打包。旧版复制到临时可写目录，新版用 `ditto -c -k --norsrc --keepParent build/bin/akproxy.app 更新包目录/akproxy-darwin-universal.zip` 打包。不要拿正式安装目录做测试。

验收：启动旧版后在应用更新对话框看到下载进度；等待「可以重启以完成更新」，期间旧版仍可使用；点击「重启」后确认 PID 改变、安装路径不变、设置页版本为新版、代理停止、监听端口关闭。对比重启前后配置和账号文件哈希。若启动代理触发上游令牌刷新，应在刷新完成后取基线，或单独在代理停止状态验证文件保持不变。

再验证设置页手动检查已是最新版本时只显示 toast；错误校验和出现「更新失败」且无重启按钮，点击「重试」会重新下载；检查阶段网络失败用 toast 提示，从设置重新检查。

本地 ad-hoc 测试不等于正式签名验收。发布前还须用专用 Developer ID、公证包和实际 GitHub Release 完成升级验证。

### 本次本地验收（2026-09-22）

- macOS 真实窗口完成 `0.0.1 → 0.0.2 → 0.0.3` 两次原位升级；PID 更换，设置页版本更新，包签名完整性检查通过。
- 第一轮升级前启动代理，HTTP 返回预期的未授权 401；升级后代理停止、端口关闭。
- 第二轮升级前后配置及全部三个账号文件哈希一致。第一轮启动代理期间一份账号文件被刷新，未把这一轮误报为全文件字节不变。
- 手动无更新、错误校验和拦截、网络断开及恢复后重新检查，均在官方窗口验证。
- macOS arm64/amd64 编译并合并通过；未在本机执行 Windows/Linux 二进制。
- 尚未完成：专用 Developer ID/公证、三平台远端 tag 工作流、真实 GitHub 发布包升级。24 小时轮询使用框架配置，未进行真实 24 小时等待验收。
