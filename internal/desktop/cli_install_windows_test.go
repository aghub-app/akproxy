//go:build windows

package desktop

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// These tests write the real HKCU PATH, so they only run when AKPROXY_E2E=1.
func requireWindowsE2E(t *testing.T) {
	t.Helper()
	if os.Getenv("AKPROXY_E2E") != "1" {
		t.Skip("set AKPROXY_E2E=1 to change the real user PATH")
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	value, kind, err := key.GetStringValue("Path")
	missing := errors.Is(err, registry.ErrNotExist)
	if err != nil && !missing {
		t.Fatal(err)
	}
	key.Close()
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

func buildCLI(t *testing.T, version string) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "akproxy-cli.exe")
	cmd := exec.Command("go", "build", "-ldflags", "-X akproxy/internal/cli.Version="+version, "-o", out, "akproxy/cmd/akproxy")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	if body, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build cli: %v\n%s", err, body)
	}
	return out
}

func fileCLI(path string) CLISource {
	return func() (io.ReadCloser, error) { return os.Open(path) }
}

func userPathEntries(t *testing.T) []string {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE)
	if err != nil {
		t.Fatal(err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(value, ";")
}

func countDir(entries []string, dir string) int {
	count := 0
	for _, entry := range entries {
		if pathEqual(entry, dir) {
			count++
		}
	}
	return count
}

// freshShellVersion runs `akproxy --version` in a new cmd.exe whose PATH is
// what a newly opened terminal gets: the user registry PATH plus System32.
func freshShellVersion(t *testing.T) (string, error) {
	t.Helper()
	var dirs []string
	for _, entry := range userPathEntries(t) {
		expanded, err := registry.ExpandString(entry)
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, expanded)
	}
	system := filepath.Join(os.Getenv("SystemRoot"), "System32")
	dirs = append(dirs, system)
	cmd := exec.Command(filepath.Join(system, "cmd.exe"), "/d", "/c", "akproxy --version")
	cmd.Env = []string{
		"PATH=" + strings.Join(dirs, ";"),
		"PATHEXT=.COM;.EXE;.BAT;.CMD",
		"SystemRoot=" + os.Getenv("SystemRoot"),
		"APPDATA=" + os.Getenv("APPDATA"),
		"USERPROFILE=" + os.Getenv("USERPROFILE"),
	}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func TestWindowsCLIInstallE2E(t *testing.T) {
	requireWindowsE2E(t)
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "bin")
	bundled := buildCLI(t, "1.2.3")

	if err := syncCLIInstall(home, fileCLI(bundled), true, "1.2.3"); err != nil {
		t.Fatalf("install: %v", err)
	}
	if got := installedCLIVersion(cliCommandPath(home)); got != "1.2.3" {
		t.Fatalf("installed version = %q", got)
	}
	if n := countDir(userPathEntries(t), dir); n != 1 {
		t.Fatalf("user PATH lists %s %d times", dir, n)
	}
	if got, err := freshShellVersion(t); err != nil || got != "1.2.3" {
		t.Fatalf("new terminal `akproxy --version` = %q %v", got, err)
	}

	// Relaunching the same version keeps the command and does not repeat PATH.
	if err := syncCLIInstall(home, fileCLI(bundled), true, "1.2.3"); err != nil {
		t.Fatalf("second install: %v", err)
	}
	if n := countDir(userPathEntries(t), dir); n != 1 {
		t.Fatalf("user PATH lists %s %d times after relaunch", dir, n)
	}

	// An app update replaces the older command.
	next := buildCLI(t, "1.2.4")
	if err := syncCLIInstall(home, fileCLI(next), true, "1.2.4"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if got, err := freshShellVersion(t); err != nil || got != "1.2.4" {
		t.Fatalf("new terminal after upgrade = %q %v", got, err)
	}

	if err := syncCLIInstall(home, nil, false, "1.2.4"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(cliCommandPath(home)); !os.IsNotExist(err) {
		t.Fatalf("command should be removed, err=%v", err)
	}
	if n := countDir(userPathEntries(t), dir); n != 0 {
		t.Fatalf("user PATH still lists %s", dir)
	}
}

// A terminal still running the old `akproxy` must not block the update:
// Windows refuses to overwrite a running executable.
func TestWindowsCLIUpgradeWhileOldCommandRuns(t *testing.T) {
	requireWindowsE2E(t)
	home := t.TempDir()
	dest := cliCommandPath(home)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	// ping.exe stands in for a long-running older akproxy session.
	ping, err := os.ReadFile(filepath.Join(os.Getenv("SystemRoot"), "System32", "PING.EXE"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, ping, 0o755); err != nil {
		t.Fatal(err)
	}
	running := exec.Command(dest, "-n", "60", "127.0.0.1")
	if err := running.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = running.Process.Kill()
		_ = running.Wait()
	})

	bundled := buildCLI(t, "2.0.0")
	if err := syncCLIInstall(home, fileCLI(bundled), true, "2.0.0"); err != nil {
		t.Fatalf("upgrade while old command runs: %v", err)
	}
	if got := installedCLIVersion(dest); got != "2.0.0" {
		t.Fatalf("installed version = %q", got)
	}

	// Turning the switch off while the old copy still runs removes the command.
	if err := syncCLIInstall(home, nil, false, "2.0.0"); err != nil {
		t.Fatalf("remove while old command runs: %v", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatalf("command should be removed, err=%v", err)
	}
}
