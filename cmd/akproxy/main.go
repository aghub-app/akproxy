package main

import (
	"fmt"
	"os"

	"akproxy/internal/cli"
	"akproxy/internal/desktop"
)

func main() {
	paths, err := desktop.Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if err := cli.Execute(os.Stdout, paths, os.Args[1:]); err != nil {
		if code, ok := cli.ExitCode(err); ok {
			os.Exit(code)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
