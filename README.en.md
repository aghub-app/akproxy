# akproxy

<p align="center">
  <a href="README.md">中文</a>
</p>

<p align="center">
  <img src="build/icon-round.png" width="128" height="128" alt="akproxy">
</p>

<p align="center">
  <a href="https://github.com/aghub-app/akproxy"><img alt="Star this repo" src="https://img.shields.io/github/stars/aghub-app/akproxy.svg?style=social&label=Star%20this%20repo"></a>
</p>

**Stop paying twice for AI.** akproxy lets you use the Codex, Claude, Gemini, Kimi, Grok, and Devin subscriptions you already pay for with coding tools that only speak the OpenAI API.

Built on [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). Sign in through the browser, run the proxy on your machine. The tool only sees one address and one client key.

## Features

- 🖥️ **Desktop app** — macOS, Windows, and Linux. The interface follows the system light or dark theme
- 🚀 **Start and stop** — one button in the window. Change the listen address or port, and that button becomes Restart
- 🔐 **Browser sign-in** — Codex, Grok, Claude, Gemini, Kimi (kimi.com and kimi.ai), and Devin. Tokens never show up in the UI
- 👥 **Multiple accounts** — sign in more than once per provider. Requests go out round-robin
- ⚡ **Live config** — client keys, the outbound proxy, and routing apply without stopping. A new listen address waits for restart
- 🔌 **Point your tool at it** — once the service is running, Home gives you a `.env` (`OPENAI_API_KEY`, `OPENAI_BASE_URL`) and examples for the OpenAI SDKs, the AI SDK, and LangChain
- 🧪 **Try a request** — the Test tab has three curl commands. Copy stays off until the service is running
- 🔄 **Updates** — release builds check, download, and verify in the background, then install when you restart
- 💾 **Its own data** — config and accounts stay in this app's directory. It does not take over a CLI config you already use

## Installation

### Download a release (recommended)

1. Open [**Releases**](https://github.com/aghub-app/akproxy/releases)
2. Grab the build for your machine:
   - **macOS**: universal (Apple silicon and Intel)
   - **Windows / Linux**: amd64
3. On a Mac, move `akproxy.app` into Applications and open it

Published macOS builds are signed and notarized. A build you make yourself is ad-hoc signed.

### Build from source

See [Development](#development) below.

## Usage

### First launch

1. Open akproxy
2. Pick a provider in the sidebar and sign in
3. Finish login in the browser. The account shows up on that page
4. Press Start

The first launch creates a client key. The default address is `http://127.0.0.1:8317`.

### Sign-in

1. The system browser opens the provider's login page
2. You sign in there
3. Back in the app, the account is in the list
4. Add more accounts for the same provider if you want

### Point a coding tool at it

While the service is running, open Connect on the home page:

- Copy the `.env`, or copy the example on an SDK card
- `OPENAI_BASE_URL` includes `/v1`
- The key is hidden on screen. What you copy is the real key

To send a request yourself, open Test and copy a curl command.

### The service

- **Start / stop / restart**: one button, and a dot for whether it is running
- **Quit the app**: the service stops with it. Nothing stays in the menu bar

Data lives in `akproxy/` under the OS user config directory. On macOS that is `~/Library/Application Support/akproxy`: `config.yaml` for settings, `auths/` for signed-in accounts.

## Development

You need Go 1.26, Node.js, pnpm, and the Wails build dependencies for your OS. The Go module and the frontend runtime are pinned to Wails `v3.0.0-beta.24`.

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
wails3 task dev
```

Vite serves the UI at `http://127.0.0.1:9245`.

```sh
wails3 task build       # bindings, frontend, and the app. On macOS: build/bin/akproxy.app
go test ./...
wails3 task package     # also writes a DMG on macOS
```

A fresh clone has to build the frontend before Go can embed `frontend/dist`. Local builds are version `dev` and do not check for updates.

Pushing a `vMAJOR.MINOR.PATCH` tag builds the macOS universal app and the Windows and Linux amd64 packages, then opens a draft Release. Signing and notarization are in [docs/releasing.md](docs/releasing.md).

## Credits

akproxy sits on [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). Sign-in, forwarding, and multiple accounts are that project. This repo is the desktop app around it.

## Support

- [GitHub Issues](https://github.com/aghub-app/akproxy/issues)
