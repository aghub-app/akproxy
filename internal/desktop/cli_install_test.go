package desktop

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMergeAndDropPathDir(t *testing.T) {
	dir := filepath.Join(string(os.PathSeparator), "Users", "me", ".local", "bin")
	merged, changed := MergePathDir("/usr/bin", dir)
	if !changed || !strings.HasPrefix(merged, dir) {
		t.Fatalf("merge = %q %v", merged, changed)
	}
	again, changed := MergePathDir(merged, dir)
	if changed || again != merged {
		t.Fatalf("second merge = %q %v", again, changed)
	}
	dropped, changed := DropPathDir(merged, dir)
	if !changed || dropped != "/usr/bin" {
		t.Fatalf("drop = %q %v", dropped, changed)
	}
}

func cliBytes(body string) CLISource {
	return func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(body)), nil
	}
}

func assertInstalled(t *testing.T, home, want string) {
	t.Helper()
	path := cliCommandPath(home)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("command should be a regular file: %v %v", info, err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("command is not executable: %v", info.Mode())
	}
	body, err := os.ReadFile(path)
	if err != nil || string(body) != want {
		t.Fatalf("command = %q %v, want %q", body, err, want)
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".akproxy-new-*"))
	if len(leftovers) > 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}
}

func TestGzipCLI(t *testing.T) {
	var packed bytes.Buffer
	zw := gzip.NewWriter(&packed)
	zw.Write([]byte("cli"))
	zw.Close()
	home := t.TempDir()
	if err := syncCLIInstall(home, GzipCLI(packed.Bytes()), true, "dev"); err != nil {
		t.Fatal(err)
	}
	assertInstalled(t, home, "cli")
}

func TestCLIInstallFailureKeepsExistingCommand(t *testing.T) {
	home := t.TempDir()
	if err := syncCLIInstall(home, cliBytes("old"), true, "dev"); err != nil {
		t.Fatal(err)
	}
	if err := syncCLIInstall(home, GzipCLI([]byte("not gzip")), true, "dev"); err == nil {
		t.Fatal("a damaged bundle should fail")
	}
	assertInstalled(t, home, "old")
}
