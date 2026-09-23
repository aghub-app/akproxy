package desktop

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppPrefsDefaults(t *testing.T) {
	root := t.TempDir()
	prefs := ReadAppPrefs(root)
	if prefs != DefaultAppPrefs() {
		t.Fatalf("missing file should yield defaults, got %+v", prefs)
	}
}

func TestAppPrefsUpgradeKeepsDefaultsAndExplicitChoices(t *testing.T) {
	for _, test := range []struct {
		name         string
		body         string
		usageEnabled bool
	}{
		{"legacy", `{"autoUpdate":false,"autoCheck":true,"checkIntervalHours":6}`, true},
		{"disabled", `{"autoUpdate":false,"autoCheck":true,"checkIntervalHours":6,"usageEnabled":false}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "app.json"), []byte(test.body), 0o600); err != nil {
				t.Fatal(err)
			}
			want := DefaultAppPrefs()
			want.AutoUpdate = false
			want.CheckIntervalHours = 6
			want.UsageEnabled = test.usageEnabled
			if got := ReadAppPrefs(root); got != want {
				t.Fatalf("preferences = %+v, want %+v", got, want)
			}
		})
	}
}

func TestAppPrefsRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := DefaultAppPrefs()
	in.AutoUpdate = false
	in.CheckIntervalHours = 6
	in.UsagePercentMode = "used"
	in.UsageResetMode = "exact"
	in.Language = "en"
	in.Theme = "dark"
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

func TestAppPrefsPresentationDefaultsAndInvalidValues(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "app.json"), []byte(`{"autoUpdate":false,"theme":"invalid","language":"fr"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := ReadAppPrefs(root)
	if got.AutoUpdate || got.Language != "system" || got.Theme != "system" {
		t.Fatalf("presentation preference upgrade = %+v", got)
	}
	if err := WriteAppPrefs(root, AppPrefs{Language: "fr", Theme: "invalid"}); err != nil {
		t.Fatal(err)
	}
	if got := ReadAppPrefs(root); got.Language != "system" || got.Theme != "system" {
		t.Fatalf("invalid choices should be normalized: %+v", got)
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
