package desktop

import (
	"os"
	"path/filepath"
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
