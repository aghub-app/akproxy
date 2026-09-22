# akproxy

[中文](README.md)

A desktop window for signing in to Codex, Grok, Claude, Gemini, Kimi, and Devin, then exposing them on your machine as an OpenAI-compatible HTTP service. Point a coding tool at that address and call it with the app's own client key.

Built on [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). Accounts and config live in this app's own data directory. It does not take over a CLI config you already use.

## Features

- **One window.** Sign in, change settings, start, stop, and restart. The interface follows the system light or dark theme.
- **Browser sign-in.** Codex, Grok, Claude, Gemini, Kimi (kimi.com and kimi.ai), and Devin. The system browser handles login. Tokens stay off the screen.
- **Local service.** Listens on `127.0.0.1:8317` by default. The first launch creates a client key.
- **Connect.** After the service is running, Home shows the address, the key, a `.env` (`OPENAI_API_KEY`, `OPENAI_BASE_URL`), and examples for the OpenAI SDKs, the AI SDK, and LangChain.
- **Try a request.** The Test tab has three curl commands. Copy stays disabled until the service is running.
- **Writes immediately.** Client keys, the outbound proxy, and routing apply while the service is up. A new listen address or port takes effect only after restart.
- **Updates.** Release builds check for updates, download and verify in the background, and install when you restart.

## Install

Download a build from [Releases](https://github.com/aghub-app/akproxy/releases). macOS is a universal app. Windows and Linux builds are amd64. Signed and notarized macOS builds come from a published release. A build you make locally is ad-hoc signed.

## Use

1. Open akproxy. Pick a provider in the sidebar and finish sign-in in the browser.
2. Press Start.
3. On Home, open Connect. Give your tool the address and key from the `.env` or an SDK example. `OPENAI_BASE_URL` includes `/v1`.
4. To send a request yourself, open Test and copy a curl command.

Data lives in `akproxy/` under the OS user config directory: `config.yaml` for settings, `auths/` for signed-in accounts. On macOS that is `~/Library/Application Support/akproxy`.

Quitting the app stops the service. There is no menu-bar resident.

## Development

You need Go 1.26, Node.js, pnpm, and the Wails native build dependencies for your OS. The Go module and the frontend runtime are pinned to Wails `v3.0.0-beta.24`.

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
wails3 task dev
```

In development, Vite serves the UI at `http://127.0.0.1:9245`.

```sh
wails3 task build          # bindings, frontend, and the app. On macOS: build/bin/akproxy.app
go test ./...
pnpm --dir frontend run build
wails3 task package        # also writes build/bin/akproxy.dmg on macOS
wails3 task bindings       # regenerate TypeScript bindings only
```

A fresh checkout has to build the frontend before Go can embed `frontend/dist`. Local builds use the version `dev` and do not check for updates.

Pushing a `vMAJOR.MINOR.PATCH` tag builds the macOS universal app and the Windows and Linux amd64 packages, then opens a draft Release. Signing and notarization are described in [docs/releasing.md](docs/releasing.md).

Product rules are in [docs/prd/local-proxy.md](docs/prd/local-proxy.md) and [docs/spec.md](docs/spec.md).

## Credits

The proxy is [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). akproxy is the local window, the sign-in flow, and the editor for this app's config.
