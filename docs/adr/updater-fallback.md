# 更新检查在 API 限额失败时改用 Atom 备用源

状态：已接受

## 背景

更新检查走 Wails GitHub provider 请求 `api.github.com` 的 `releases/latest`。该端点对未认证请求限额 60 次/小时/IP，限额由整机所有进程共享。正式版启动和每 24 小时各检查一次，共享出口（NAT、代理池）下的用户常态化遇到 403。把 PAT 编进应用分发给所有用户等于泄露凭证；要求用户自己填 token 则给「检查更新」增加了配置负担。

## 决定

`internal/updates` 里的 provider 按顺序降级，对更新器仍表现为单一来源：

- 主路是现有 GitHub API provider，行为不变。
- 主路返回 403、429、5xx 或传输层错误时，改请求 `github.com/aghub-app/akproxy/releases.atom`。这个 feed 在 GitHub web 基础设施上，不占 API 限额，也不需要凭据。取 feed 里 semver 最大的纯三段式 tag。
- 下载地址按 CI 资产命名规则确定性拼接为 `releases/download/<tag>/<资产名>`，`SHA256SUMS` 同规则拼接。
- 主路 404 表示仓库还没有任何 release，feed 同样为空，不降级。
- 主路成功时不请求备用路。

## 备选

- 内置共享 PAT。提高限额到 5000 次/小时，但凭证进二进制即泄露，限额照样可能被耗尽。
- 要求用户填自己的 token。检查更新不该有配置和信任成本。
- 两路并行竞速。请求翻倍，发布瞬间两路可能不一致，顺序降级已足够。
- 解析 `releases/latest` 的 HTML 页。非契约格式，比 Atom 更脆弱。

## 后果

- Atom feed 不是 GitHub 承诺的契约。格式变更或条目缺失时备用路失败，呈现为可重试的检查失败。
- feed 只保留最近若干条 release。对「找最新版」足够。
- feed 按 updated 时间排序而非 semver 序，解析全部条目取 semver 最大者。
- 只接受纯 `vMAJOR.MINOR.PATCH`，预发布和草稿不参与；草稿本就不出现在 feed，符合「维护者手动发布后才可见」。
- 校验失败不换源重试。任一来源的 SHA-256 校验不过，整个更新流程终止。
- 用户网络整体到不了 `github.com` 时备用路同样不可用；那种场景下载本身也不可用，不为它设计第三条路。

## 验证

- 主路 403 后降级，能拿到新版本并通过校验，等待重启安装。
- 主路 404 不产生对 feed 的请求。
- feed 不可达、SHA256SUMS 缺失或缺少对应行时，检查失败且不返回 release。
- 既有 GitHub API 用例行为不变。
