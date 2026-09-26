# 命令行启动编程代理的环境

状态：已接受

## 背景

`akproxy claude`、`akproxy codex`、`akproxy opencode` 和 `akproxy pi` 要把对应程序指到本机代理。做法参考 Ollama 的 `cmd/launch`，但不把整份 Ollama 放进仓库。产品要求不改 `~/.claude`、`~/.codex`、`~/.pi` 和 OpenCode 自己的配置文件，参数原样往后传，全局只用一个模型 id。请求用配置里的第一把客户端密钥。专用密钥还没做。本机没有对应程序时只报错，不替用户安装。

## 决定

- 所选模型只放在应用数据目录的 `cli.json`：`{ "model": { "id": "..." } }`。没有 id 时，这四条命令失败，提示先运行 `akproxy model`。
- 数据目录里还没有应用配置时失败，提示先打开 akproxy。它们不启动代理。
- Claude 直接执行 `claude`。环境设置 `ANTHROPIC_BASE_URL` 为本机代理原点（不含 `/v1`）、`ANTHROPIC_AUTH_TOKEN` 为客户端密钥，并把 `ANTHROPIC_MODEL` 和 Opus、Sonnet、Haiku、子代理模型都设为所选 id。清掉 `ANTHROPIC_API_KEY`。`--model` 放在用户参数前面。
- Codex 直接执行 `codex`。在用户参数之前插入 `-c`：provider 名为 `akproxy`，`base_url` 是本机代理地址加 `/v1`，`wire_api` 为 `responses`，`env_key` 为 `AKPROXY_CLI_KEY`。密钥只放在该环境变量里。`model` 同样用 `-c` 设为所选 id。不写 `~/.codex`。
- OpenCode 直接执行 `opencode`。配置放在本次进程的 `OPENCODE_CONFIG_CONTENT` 里，provider 用 `@ai-sdk/openai-compatible`，地址带 `/v1`，模型是 `akproxy/<id>`。不写 OpenCode 的状态文件。
- Pi 直接执行 `pi`。本次进程用一个临时目录作为 `PI_CODING_AGENT_DIR`，里面写 `models.json` 和 `settings.json`。`apiKey` 写成 `$AKPROXY_CLI_KEY`，密钥只在环境变量里。进程结束后删掉临时目录。不改用户的 `~/.pi`。
- 查找顺序：先 `PATH`。Claude 再看 `~/.local/bin` 和 `~/.claude/local`。OpenCode 再看 `~/.opencode/bin`。找不到就失败。
- `akproxy model` 的选择界面见 `cli-model-picker.md`。

## 备选

- 把密钥和模型写进 `~/.claude/settings.json` 或 `~/.codex/config.toml`。能压过用户原来的配置，但会改他们的文件，退出后还留着。放弃。
- 只导出 `OPENAI_BASE_URL`。Codex 在已有 provider 配置时会忽略它。放弃。
- Claude 使用 `ANTHROPIC_API_KEY`。那会变成 `x-api-key`，本机代理的客户端密钥走 Bearer。放弃。

## 后果

- 用户 shell 或 Claude 的 `settings.json` 仍可能在真正执行时盖过这次环境。真正启动时要单独处理，不在这一步改用户文件。
- 模型 id 是全局一个。选了只被一边认识的模型时，另一边会在真正启动后失败。

## 验证

- 解析、缺模型、缺应用配置，以及四条命令的参数和环境，由 `go test ./internal/cli/` 覆盖。测试不真正执行外部程序。
