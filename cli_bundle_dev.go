//go:build !production

package main

import "akproxy/internal/desktop"

// Development builds use the CLI built beside the desktop executable.
var bundledCLI desktop.CLISource = desktop.SiblingCLI
