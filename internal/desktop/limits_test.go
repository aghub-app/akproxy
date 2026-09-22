package desktop

import (
	"testing"
	"time"
)

func TestParseClaudeLimitsClampsAndSkipsMissing(t *testing.T) {
	windows, err := parseClaudeLimits([]byte(`{
		"five_hour": {"utilization": 42, "resets_at": "2026-06-09T12:00:00+00:00"},
		"seven_day": {"utilization": 150}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 2 {
		t.Fatalf("windows = %+v", windows)
	}
	if windows[0].Kind != "5h" || windows[0].Used != 42 || windows[0].ResetsAt == "" {
		t.Fatalf("five hour = %+v", windows[0])
	}
	if windows[1].Kind != "weekly" || windows[1].Used != 100 {
		t.Fatalf("weekly = %+v", windows[1])
	}
}

func TestParseCodexLimitsReadsCurrentAndLegacyShapes(t *testing.T) {
	now := time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)
	windows, err := parseCodexLimits([]byte(`{
		"rate_limit": {
			"primary_window": {"used_percent": 5, "limit_window_seconds": 2592000, "reset_at": 1785339666},
			"secondary_window": {"used_percent": 20, "limit_window_seconds": 18000}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 2 || windows[0].Kind != "weekly" || windows[0].Used != 5 || windows[0].ResetsAt == "" {
		t.Fatalf("primary = %+v", windows)
	}
	if windows[1].Kind != "5h" || windows[1].Used != 20 {
		t.Fatalf("secondary = %+v", windows[1])
	}

	legacy, err := parseCodexLimits([]byte(`{"rate_limits":{"primary":{"percent_left":30},"secondary":{"used_percent":250}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if legacy[0].Used != 70 || legacy[1].Used != 100 {
		t.Fatalf("legacy = %+v", legacy)
	}
	reset := codexReset(map[string]any{"resets_in_seconds": float64(3600)}, now)
	if reset != "2026-09-22T01:00:00Z" {
		t.Fatalf("reset = %s", reset)
	}
}

func TestParseCodexLimitsRejectsUnknownWindow(t *testing.T) {
	windows, err := parseCodexLimits([]byte(`{"rate_limit":{"primary_window":{"foo":1}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 0 {
		t.Fatalf("windows = %+v", windows)
	}
}
