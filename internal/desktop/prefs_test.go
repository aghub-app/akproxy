package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppPrefsDefaults(t *testing.T) {
	root := t.TempDir()
	prefs := ReadAppPrefs(root)
	if prefs != (AppPrefs{AutoUpdate: true, AutoCheck: true, CheckIntervalHours: 24}) {
		t.Fatalf("missing file should yield defaults, got %+v", prefs)
	}
}

func TestAppPrefsRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := AppPrefs{AutoUpdate: false, AutoCheck: true, CheckIntervalHours: 6}
	if err := WriteAppPrefs(root, in); err != nil {
		t.Fatal(err)
	}
	if got := ReadAppPrefs(root); got != in {
		t.Fatalf("round trip mismatch: got %+v want %+v", got, in)
	}
	body, err := os.ReadFile(filepath.Join(root, "app.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		t.Fatalf("app.json should end with a newline")
	}
}

func TestAppPrefsDamagedFileFallsBack(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := ReadAppPrefs(root); got != DefaultAppPrefs() {
		t.Fatalf("damaged file should fall back to defaults, got %+v", got)
	}
}

func TestAppPrefsNormalizeInterval(t *testing.T) {
	root := t.TempDir()
	if err := WriteAppPrefs(root, AppPrefs{AutoUpdate: true, AutoCheck: true, CheckIntervalHours: 0}); err != nil {
		t.Fatal(err)
	}
	if got := ReadAppPrefs(root); got.CheckIntervalHours != DefaultCheckIntervalHours {
		t.Fatalf("out-of-range interval should normalize to default, got %d", got.CheckIntervalHours)
	}
	if err := WriteAppPrefs(root, AppPrefs{CheckIntervalHours: MaxCheckIntervalHours}); err != nil {
		t.Fatal(err)
	}
	if got := ReadAppPrefs(root); got.CheckIntervalHours != MaxCheckIntervalHours {
		t.Fatalf("boundary interval should be kept, got %d", got.CheckIntervalHours)
	}
}
