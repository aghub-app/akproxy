//go:build e2e && windows

package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// guardUserPath skips unless AKPROXY_E2E=1, because Windows keeps the user
// PATH in HKCU rather than the sandboxed home. It restores the value afterwards.
func guardUserPath(t *testing.T) {
	t.Helper()
	if os.Getenv("AKPROXY_E2E") != "1" {
		t.Skip("set AKPROXY_E2E=1 to change the real user PATH")
	}
	value, kind, err := readUserPath()
	missing := errors.Is(err, registry.ErrNotExist)
	if err != nil && !missing {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE)
		if err != nil {
			t.Error(err)
			return
		}
		defer key.Close()
		switch {
		case missing:
			err = key.DeleteValue("Path")
		case kind == registry.EXPAND_SZ:
			err = key.SetExpandStringValue("Path", value)
		default:
			err = key.SetStringValue("Path", value)
		}
		if err != nil && !errors.Is(err, registry.ErrNotExist) {
			t.Error(err)
		}
	})
}

func readUserPath() (string, uint32, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		return "", 0, err
	}
	defer key.Close()
	return key.GetStringValue("Path")
}

// freshShell runs line in a new cmd.exe whose PATH is what a newly opened
// terminal gets from the user registry, plus System32.
func freshShell(t *testing.T, sb *sandbox, extra map[string]string, line string) (string, error) {
	t.Helper()
	value, _, err := readUserPath()
	if err != nil && !errors.Is(err, registry.ErrNotExist) {
		t.Fatal(err)
	}
	var dirs []string
	for _, entry := range strings.Split(value, ";") {
		if entry == "" {
			continue
		}
		expanded, err := registry.ExpandString(entry)
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, expanded)
	}
	system := filepath.Join(os.Getenv("SystemRoot"), "System32")
	env := map[string]string{"PATH": strings.Join(append(dirs, system), ";")}
	for key, value := range extra {
		env[key] = value
	}
	cmd := exec.Command(filepath.Join(system, "cmd.exe"), "/d", "/c", line)
	cmd.Env = sb.environ(env)
	cmd.Dir = sb.home
	out, err := cmd.CombinedOutput()
	return trimLines(out), err
}

func userPathHasCommandDir(t *testing.T, sb *sandbox) bool {
	t.Helper()
	value, _, err := readUserPath()
	if err != nil {
		return false
	}
	dir := filepath.Join(sb.home, ".local", "bin")
	for _, entry := range strings.Split(value, ";") {
		if strings.EqualFold(filepath.Clean(strings.TrimSpace(entry)), dir) {
			return true
		}
	}
	return false
}
