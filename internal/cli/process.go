package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var start = startProcess

func startProcess(launch Launch) error {
	if launch.Dir != "" {
		defer os.RemoveAll(launch.Dir)
	}
	bin, err := findBinary(launch.Argv[0])
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, launch.Argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergeEnv(os.Environ(), launch)
	return cmd.Run()
}

func mergeEnv(base []string, launch Launch) []string {
	drop := map[string]bool{}
	for _, key := range launch.Clear {
		drop[key] = true
	}
	for key := range launch.Env {
		drop[key] = true
	}
	out := make([]string, 0, len(base)+len(launch.Env))
	for _, item := range base {
		key, _, ok := strings.Cut(item, "=")
		if ok && drop[key] {
			continue
		}
		out = append(out, item)
	}
	for key, value := range launch.Env {
		out = append(out, key+"="+value)
	}
	return out
}

func findBinary(name string) (string, error) {
	file := name
	if runtime.GOOS == "windows" {
		file += ".exe"
	}
	if path, err := exec.LookPath(file); err == nil {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("找不到 %s", name)
	}
	var candidates []string
	switch name {
	case "claude":
		candidates = []string{
			filepath.Join(home, ".local", "bin", file),
			filepath.Join(home, ".claude", "local", file),
		}
	case "opencode":
		candidates = []string{filepath.Join(home, ".opencode", "bin", file)}
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("找不到 %s", name)
}

// ExitCode reports a launched program's own status.
func ExitCode(err error) (int, bool) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), true
	}
	return 0, false
}
