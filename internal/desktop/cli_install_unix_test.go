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
	bundled := filepath.Join(t.TempDir(), "akproxy-cli")
	if err := os.WriteFile(bundled, []byte("cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, bundled, true, "dev"); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, bundled, true, "dev"); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(cliCommandPath(home))
	if err != nil || link != bundled {
		t.Fatalf("link = %q %v", link, err)
	}
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
	other := filepath.Join(t.TempDir(), "akproxy-cli")
	if err := os.WriteFile(other, []byte("next"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, other, true, "dev"); err != nil {
		t.Fatal(err)
	}
	if link, err = os.Readlink(cliCommandPath(home)); err != nil || link != other {
		t.Fatalf("replaced link = %q %v", link, err)
	}
	body, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(files[0], []byte("export KEEP=1\n"+string(body)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, "", false, "dev"); err != nil {
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
	bundled := filepath.Join(t.TempDir(), "akproxy-cli")
	if err := os.WriteFile(bundled, []byte("next"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, bundled, true, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || !strings.Contains(string(body), "1.2.3") {
		t.Fatalf("matching version should stay: %q %v", body, err)
	}
	profile, err := os.ReadFile(filepath.Join(home, ".zprofile"))
	if err != nil || !strings.Contains(string(profile), cliPathBegin) {
		t.Fatalf("PATH should still be ensured: %q %v", profile, err)
	}
}

func TestCLIInstallReplacesMismatchedVersion(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	writeVersionCommand(t, dest, "1.0.0")
	bundled := filepath.Join(t.TempDir(), "akproxy-cli")
	if err := os.WriteFile(bundled, []byte("next"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, bundled, true, "1.0.1"); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(dest)
	if err != nil || link != bundled {
		t.Fatalf("mismatched version link = %q %v", link, err)
	}
}

func TestCLIInstallDevAlwaysReplaces(t *testing.T) {
	home := t.TempDir()
	dest := cliCommandPath(home)
	writeVersionCommand(t, dest, "dev")
	bundled := filepath.Join(t.TempDir(), "akproxy-cli")
	if err := os.WriteFile(bundled, []byte("next"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, bundled, true, "dev"); err != nil {
		t.Fatal(err)
	}
	link, err := os.Readlink(dest)
	if err != nil || link != bundled {
		t.Fatalf("dev link = %q %v", link, err)
	}
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
	if err := syncCLIInstall(home, "", false, "dev"); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(dest)
	if err != nil || string(body) != "foreign" {
		t.Fatalf("foreign command = %q %v", body, err)
	}
}
