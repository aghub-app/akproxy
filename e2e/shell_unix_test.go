//go:build e2e && !windows

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func guardUserPath(*testing.T) {}

// freshShell runs line in a new login shell. PATH starts from the system
// default, so only the shell startup files can make `akproxy` reachable.
func freshShell(t *testing.T, sb *sandbox, extra map[string]string, line string) (string, error) {
	t.Helper()
	shell := []string{"/bin/bash", "-l", "-c", line}
	if runtime.GOOS == "darwin" {
		shell = []string{"/bin/zsh", "-l", "-c", line}
	}
	env := map[string]string{"PATH": "/usr/bin:/bin:/usr/sbin:/sbin"}
	for key, value := range extra {
		env[key] = value
	}
	cmd := exec.Command(shell[0], shell[1:]...)
	cmd.Env = sb.environ(env)
	cmd.Dir = sb.home
	out, err := cmd.CombinedOutput()
	return trimLines(out), err
}

func userPathHasCommandDir(t *testing.T, sb *sandbox) bool {
	t.Helper()
	for _, name := range []string{".zprofile", ".zshrc", ".profile", ".bashrc"} {
		body, err := os.ReadFile(filepath.Join(sb.home, name))
		if err == nil && strings.Contains(string(body), "akproxy cli") {
			return true
		}
	}
	return false
}
