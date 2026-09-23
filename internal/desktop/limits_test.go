package desktop

import (
	"context"
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

func TestParseClaudeLimitsDoesNotInventZeroUsage(t *testing.T) {
	windows, err := parseClaudeLimits([]byte(`{"five_hour":{"resets_at":"2026-09-23T12:00:00Z"},"seven_day":{"utilization":0}}`))
	if err != nil || len(windows) != 1 || windows[0].Kind != "weekly" || windows[0].Used != 0 {
		t.Fatalf("missing utilization must be skipped, explicit zero kept: %+v, %v", windows, err)
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
	if len(windows) != 2 || windows[0].Kind != "monthly" || windows[0].Used != 5 || windows[0].ResetsAt == "" || windows[0].DurationSeconds != 2592000 {
		t.Fatalf("primary = %+v", windows)
	}
	if windows[1].Kind != "5h" || windows[1].Used != 20 {
		t.Fatalf("secondary = %+v", windows[1])
	}
	weekly, ok := codexWindow(map[string]any{"used_percent": float64(12), "limit_window_seconds": float64(604800)}, "5h", now)
	if !ok || weekly.Kind != "weekly" {
		t.Fatalf("seven-day window = %+v, %v", weekly, ok)
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

func TestCodexPlanName(t *testing.T) {
	cases := map[string]string{
		"prolite":                     "Pro 5x",
		"pro":                         "Pro 20x",
		"self_serve_business_prolite": "Business Premium",
		"team":                        "Team",
		"self_serve_business":         "Self Serve Business",
	}
	for raw, want := range cases {
		if got := codexPlanName(raw); got != want {
			t.Fatalf("codexPlanName(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := codexPlanName(""); got != "" {
		t.Fatalf("codexPlanName(empty) = %q", got)
	}
}

func TestCodexMissingTokenFailsBeforeRequest(t *testing.T) {
	_, err := fetchCodexLimits(context.Background(), "", "")
	if err == nil || err.Error() != "这个账号没有可用的登录令牌" {
		t.Fatalf("missing token error = %v", err)
	}
}

func TestParseGrokLimitsWeeklyOnly(t *testing.T) {
	windows, err := parseGrokLimits([]byte(`{
		"period_type": "weekly",
		"used_percent": 42.5,
		"period_end_unix": 1785339666,
		"period_duration_seconds": 604800
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 1 || windows[0].Kind != "weekly" || windows[0].Used != 42.5 || windows[0].ResetsAt != "2026-07-29T15:41:06Z" {
		t.Fatalf("windows = %+v", windows)
	}
}

func TestParseGrokLegacyWindowDoesNotInventZeroUsage(t *testing.T) {
	windows, err := parseGrokLimits([]byte(`{"period_type":"weekly","period_end_unix":1785339666}`))
	if err != nil || len(windows) != 0 {
		t.Fatalf("missing legacy usage must be skipped: %+v, %v", windows, err)
	}
}

func TestParseGrokLimitsMonthlyAccountHasNoWindow(t *testing.T) {
	windows, err := parseGrokLimits([]byte(`{"period_type":"monthly","used_percent":10,"period_end_unix":1785339666}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 0 {
		t.Fatalf("windows = %+v", windows)
	}
}

func TestParseDevinLimitsReadsQuotaAndPlan(t *testing.T) {
	windows, plan, err := parseDevinLimits([]byte(`{
		"userStatus": {
			"planStatus": {
				"planInfo": {"planName": "Team"},
				"dailyQuotaRemainingPercent": 80,
				"weeklyQuotaRemainingPercent": 55,
				"dailyQuotaResetAtUnix": 1785339666,
				"weeklyQuotaResetAtUnix": 1785944466
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if plan != "Team" {
		t.Fatalf("plan = %q", plan)
	}
	if len(windows) != 2 {
		t.Fatalf("windows = %+v", windows)
	}
	if windows[0].Kind != "daily" || windows[0].Used != 20 || windows[0].ResetsAt != "2026-07-29T15:41:06Z" {
		t.Fatalf("daily = %+v", windows[0])
	}
	if windows[1].Kind != "weekly" || windows[1].Used != 45 || windows[1].ResetsAt != "2026-08-05T15:41:06Z" {
		t.Fatalf("weekly = %+v", windows[1])
	}
}

func TestParseDevinLimitsHidesDailyQuota(t *testing.T) {
	windows, _, err := parseDevinLimits([]byte(`{
		"userStatus": {
			"planStatus": {
				"planInfo": {"planName": "Team", "hideDailyQuota": true},
				"weeklyQuotaRemainingPercent": 55,
				"weeklyQuotaResetAtUnix": 1785944466
			}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 1 || windows[0].Kind != "weekly" {
		t.Fatalf("windows = %+v", windows)
	}
}

func TestParseDevinLimitsRejectsMissingQuota(t *testing.T) {
	for _, body := range []string{
		`{}`, `null`, `{"userStatus":{}}`,
		`{"userStatus":{"planStatus":{}}}`,
		`{"userStatus":{"planStatus":{"planInfo":{"planName":"Team"}}}}`,
	} {
		windows, _, err := parseDevinLimits([]byte(body))
		if err == nil || len(windows) != 0 {
			t.Errorf("response %s produced windows %+v, error %v", body, windows, err)
		}
	}
}

func TestParseDevinLimitsDistinguishesAbsentAndExhaustedWindows(t *testing.T) {
	for _, test := range []struct {
		body string
		used float64
	}{
		{`{"weeklyQuotaRemainingPercent":55}`, 45},
		{`{"weeklyQuotaRemainingPercent":0}`, 100},
		{`{"weeklyQuotaResetAtUnix":1785944466}`, 100},
	} {
		windows, _, err := parseDevinLimits([]byte(`{"userStatus":{"planStatus":` + test.body + `}}`))
		if err != nil || len(windows) != 1 || windows[0].Kind != "weekly" || windows[0].Used != test.used {
			t.Errorf("response %s produced windows %+v, error %v", test.body, windows, err)
		}
	}
}

func TestFormatPlanName(t *testing.T) {
	if got := formatPlanName("claude_max", "5x"); got != "max 5x" {
		t.Fatalf("plan = %q", got)
	}
	if got := formatPlanName("claude_pro", ""); got != "pro" {
		t.Fatalf("plan = %q", got)
	}
	if got := formatPlanName("", "5x"); got != "" {
		t.Fatalf("plan = %q", got)
	}
}

func TestGrokPlanName(t *testing.T) {
	if got := grokPlanName([]byte(`{"subscription_tier_display":"SuperGrok Heavy"}`)); got != "SuperGrok Heavy" {
		t.Fatalf("plan = %q", got)
	}
	if got := grokPlanName([]byte(`{}`)); got != "" {
		t.Fatalf("plan = %q", got)
	}
}
