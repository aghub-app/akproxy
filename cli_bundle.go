//go:build production

package main

import (
	_ "embed"

	"akproxy/internal/desktop"
)

// The CLI travels inside the desktop executable, because the updater replaces
// only that one file on Windows and Linux. `wails3 task build` writes it first.
//
//go:embed build/bin/akproxy-cli.gz
var bundledCLIGzip []byte

var bundledCLI = desktop.GzipCLI(bundledCLIGzip)
