package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AppPrefs is the 更新偏好 owned by the 关于 page. It lives in app.json and
// never touches config.yaml, whose parser drops unknown fields.
type AppPrefs struct {
	AutoUpdate         bool `json:"autoUpdate"`
	AutoCheck          bool `json:"autoCheck"`
	CheckIntervalHours int  `json:"checkIntervalHours"`
}

const (
	DefaultCheckIntervalHours = 24
	MaxCheckIntervalHours     = 720
)

// DefaultAppPrefs matches the historical built-in behavior.
func DefaultAppPrefs() AppPrefs {
	return AppPrefs{AutoUpdate: true, AutoCheck: true, CheckIntervalHours: DefaultCheckIntervalHours}
}

// Normalize clamps a parsed prefs value into the accepted range.
func NormalizeAppPrefs(in AppPrefs) AppPrefs {
	out := in
	if out.CheckIntervalHours < 1 || out.CheckIntervalHours > MaxCheckIntervalHours {
		out.CheckIntervalHours = DefaultCheckIntervalHours
	}
	return out
}

// ReadAppPrefs loads app.json, falling back to defaults when the file is
// missing or damaged.
func ReadAppPrefs(root string) AppPrefs {
	body, err := os.ReadFile(filepath.Join(root, "app.json"))
	if err != nil {
		return DefaultAppPrefs()
	}
	var prefs AppPrefs
	if err := json.Unmarshal(body, &prefs); err != nil {
		return DefaultAppPrefs()
	}
	return NormalizeAppPrefs(prefs)
}

// WriteAppPrefs saves app.json for the 关于 page.
func WriteAppPrefs(root string, in AppPrefs) error {
	prefs := NormalizeAppPrefs(in)
	body, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return fmt.Errorf("偏好无法编码: %w", err)
	}
	path := filepath.Join(root, "app.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o600); err != nil {
		return fmt.Errorf("偏好无法保存: %w", err)
	}
	return nil
}
