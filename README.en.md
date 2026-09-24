<p align="center">
  <img src="build/icon-round.png" width="128" height="128" alt="akproxy icon">
</p>

<h1 align="center">akproxy</h1>

<p align="center">
  <strong>Stop paying twice for AI.</strong><br>
  Use the Codex, Claude, Gemini, Kimi, Grok, and Devin subscriptions you already pay for with coding tools that only speak the OpenAI API.
</p>

<p align="center">
  <a href="https://github.com/aghub-app/akproxy/releases">Download</a> · <a href="#installation">Installation</a> · <a href="README.md">中文</a>
</p>

<p align="center">
  <a href="https://github.com/aghub-app/akproxy/releases"><img src="https://img.shields.io/github/v/release/aghub-app/akproxy" alt="Latest release"></a>
  <a href="https://github.com/aghub-app/akproxy/actions/workflows/macos-dmg.yml"><img src="https://github.com/aghub-app/akproxy/actions/workflows/macos-dmg.yml/badge.svg" alt="Release build status"></a>
  <a href="https://github.com/aghub-app/akproxy/releases"><img src="https://img.shields.io/github/downloads/aghub-app/akproxy/total" alt="Downloads across all releases"></a>
  <a href="https://github.com/aghub-app/akproxy/stargazers"><img src="https://img.shields.io/github/stars/aghub-app/akproxy" alt="GitHub Stars"></a>
</p>

akproxy is built on [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). Sign in through the browser, run the proxy on your machine. The tool only sees one address and one client key.

## Quick start

1. Follow the [installation instructions](#installation) to download the build for your system.
2. Open akproxy, pick a provider in the sidebar, and finish sign-in in your browser.
3. Press Start, then copy the `.env` or an SDK example from Home → Connect into your coding tool.

First launch creates a client key. The default address is `http://127.0.0.1:8317`; Home fills the current address and key into the connection examples.

## Features

- 🖥️ **Desktop app** — macOS, Windows, and Linux. The interface follows the system light or dark theme
- 🚀 **Start and stop** — one button. The service starts and stops when you say so
- 🔐 **Browser sign-in** — Codex, Grok, Claude, Gemini, Kimi (kimi.com and kimi.ai), and Devin
- 👥 **Multiple accounts** — sign in more than once per provider. Requests go out round-robin by default
- ⚡ **Live config** — client keys, the outbound proxy, and routing apply as soon as you change them
- 🔌 **Point your tool at it** — Home gives you a `.env` and examples for the OpenAI SDKs, the AI SDK, and LangChain
- 🧪 **Try a request** — the Test tab has curl examples for the model list and two generation endpoints. Test them with a model that supports each endpoint
- 💾 **Its own data** — config and accounts stay in this app's directory

## Installation

### Download a release (recommended)

1. Open [**Releases**](https://github.com/aghub-app/akproxy/releases)
2. Grab the build for your machine:
   - **macOS**: universal (Apple silicon and Intel)
   - **Windows / Linux**: amd64
3. On a Mac, move `akproxy.app` into Applications and open it

The macOS release workflow signs and notarizes packages. A build you make yourself is ad-hoc signed.

### Build from source

See [Development](#development) below.

## Usage

### Point a coding tool at it

While the service is running, open Connect on the home page:

- Copy the `.env`, or switch examples on the SDK card and copy one
- `OPENAI_BASE_URL` includes `/v1`
- The key is hidden on screen. What you copy is the real key

To send a request yourself, open Test and copy a curl command.

### The service

- **Start / stop / restart**: one button, and a dot for whether it is running
- **Listen address or port**: restart the service to apply a change
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

The development and build tasks above generate bindings, install frontend dependencies, and build the frontend before Go embeds it. Local builds are version `dev` and do not check for updates.

Pushing a `vMAJOR.MINOR.PATCH` tag builds the macOS universal app and the Windows and Linux amd64 packages, then opens a draft Release. Signing and notarization are in [docs/releasing.md](docs/releasing.md).

## Credits

akproxy sits on [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). Sign-in, forwarding, and multiple accounts are that project. This repo is the desktop app around it.

## Support

- [GitHub Issues](https://github.com/aghub-app/akproxy/issues)
