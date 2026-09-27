//go:build !windows

package desktop

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIInstallAndRemove(t *testing.T) {
	home := t.TempDir()
	if err := syncCLIInstall(home, cliBytes("cli"), true, "dev"); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, cliBytes("cli"), true, "dev"); err != nil {
		t.Fatal(err)
	}
	assertInstalled(t, home, "cli")
	files := shellStartupFiles(home)
	if len(files) == 0 {
		t.Fatal("expected shell startup files")
	}
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Count(string(body), cliPathBegin) != 1 {
			t.Fatalf("%s = %q", path, body)
		}
	}
	if err := syncCLIInstall(home, cliBytes("next"), true, "dev"); err != nil {
		t.Fatal(err)
	}
	assertInstalled(t, home, "next")
	body, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(files[0], []byte("export KEEP=1\n"+string(body)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, nil, false, "dev"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(cliCommandPath(home)); !os.IsNotExist(err) {
		t.Fatalf("command should be removed, err=%v", err)
	}
	left, err := os.ReadFile(files[0])
	if err != nil || strings.Contains(string(left), cliPathBegin) || !strings.Contains(string(left), "KEEP=1") {
		t.Fatalf("startup file after removal = %q %v", left, err)
	}
	if _, err := os.Lstat(files[1]); !os.IsNotExist(err) {
		t.Fatalf("empty shell file should be removed, err=%v", err)
	}
}

func writeVersionCommand(t *testing.T, path, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "#!/bin/sh\nprintf '%s\\n' '" + version + "'\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestCLIInstallKeepsMatchingVersion(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	writeVersionCommand(t, dest, "1.2.3")
	if err := syncCLIInstall(home, cliBytes("next"), true, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(body), "1.2.3") {
		t.Fatalf("matching version should stay: %q %v", body, err)
	}
	profile, err := os.ReadFile(shellStartupFiles(home)[0])
	if err != nil || !strings.Contains(string(profile), cliPathBegin) {
		t.Fatalf("PATH should still be ensured: %q %v", profile, err)
	}
}

func TestCLIInstallReplacesMismatchedVersion(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	writeVersionCommand(t, dest, "1.0.0")
	if err := syncCLIInstall(home, cliBytes("next"), true, "1.0.1"); err != nil {
		t.Fatal(err)
	}
	assertInstalled(t, home, "next")
}

func TestCLIInstallDevAlwaysReplaces(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	writeVersionCommand(t, dest, "dev")
	if err := syncCLIInstall(home, cliBytes("next"), true, "dev"); err != nil {
		t.Fatal(err)
	}
	assertInstalled(t, home, "next")
}

func TestCLIInstallLeavesForeignCommand(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("foreign"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, nil, false, "dev"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || string(body) != "foreign" {
		t.Fatalf("foreign command = %q %v", body, err)
	}
}
