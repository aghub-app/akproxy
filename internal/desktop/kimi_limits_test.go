package desktop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseKimiLimitsPrefersRatioPools(t *testing.T) {
	windows, plan, err := parseKimiLimits([]byte(`{
		"version": "GOODS_VERSION_V1",
		"user": {"membership": {"level": "LEVEL_INTERMEDIATE"}},
		"usages": {
			"limit_5h": {"used_ratio": 0.25, "reset_time": "2026-07-02T16:00:00Z"},
			"limit_7d": {"used_ratio": 0.1, "reset_time": "2026-07-06T07:00:00Z"},
			"limit_month_total": {"used_ratio": 0.4, "reset_time": "2026-08-01T00:00:00Z"}
		},
		"usage": {"limit": "2048", "used": "2000", "remaining": "48", "resetTime": "2026-07-06T07:00:00Z"}
	}`))
	if err != nil || plan != "Allegretto" || len(windows) != 3 {
		t.Fatalf("plan=%q windows=%+v err=%v", plan, windows, err)
	}
	if windows[0].Kind != "5h" || windows[0].Used != 25 || windows[0].DurationSeconds != kimiSessionSeconds {
		t.Fatalf("session: %+v", windows[0])
	}
	if windows[1].Kind != "weekly" || windows[1].Used != 10 || windows[2].Kind != "monthly" || windows[2].Used != 40 {
		t.Fatalf("longer windows: %+v", windows[1:])
	}
}

func TestParseKimiLimitsFallsBackFromZeroRatioPlaceholder(t *testing.T) {
	windows, _, err := parseKimiLimits([]byte(`{
		"usages": {"limit_7d": {"used_ratio": 0, "reset_time": "2026-07-06T07:00:00Z"}},
		"usage": {"limit": "2048", "used": "214", "remaining": "1834", "resetTime": "2026-07-06T07:00:01.450Z"}
	}`))
	if err != nil || len(windows) != 1 || windows[0].Kind != "weekly" || windows[0].Used != 214.0/2048.0*100 {
		t.Fatalf("windows=%+v err=%v", windows, err)
	}
}

func TestParseKimiLimitsKeepsZeroRatioWhenMonthlyExists(t *testing.T) {
	windows, _, err := parseKimiLimits([]byte(`{
		"usages": {
			"limit_7d": {"used_ratio": 0, "reset_time": "2026-07-06T07:00:00Z"},
			"limit_month_total": {"used_ratio": 0.2, "reset_time": "2026-08-01T00:00:00Z"}
		},
		"usage": {"limit": "2048", "used": "214", "resetTime": "2026-07-06T07:00:00Z"}
	}`))
	if err != nil || len(windows) != 2 || windows[0].Kind != "weekly" || windows[0].Used != 0 || windows[1].Kind != "monthly" {
		t.Fatalf("windows=%+v err=%v", windows, err)
	}
}

func TestParseKimiLimitsReadsLegacyCounts(t *testing.T) {
	windows, plan, err := parseKimiLimits([]byte(`{
		"user": {"membership": {"level": "LEVEL_UNSPECIFIED"}},
		"usage": {"limit": "2048", "used": "214", "remaining": "1834", "resetTime": "2026-01-09T15:23:13.716839300Z"},
		"limits": [{
			"window": {"duration": 300, "timeUnit": "TIME_UNIT_MINUTE"},
			"detail": {"limit": "200", "used": "139", "remaining": "61", "resetTime": "2026-01-06T13:33:02.717479433Z"}
		}]
	}`))
	if err != nil || plan != "" || len(windows) != 2 {
		t.Fatalf("plan=%q windows=%+v err=%v", plan, windows, err)
	}
	if windows[0].Kind != "5h" || windows[0].Used != 69.5 || windows[0].ResetsAt != "2026-01-06T13:33:02Z" {
		t.Fatalf("rate: %+v", windows[0])
	}
	if windows[1].Kind != "weekly" || windows[1].Used != 214.0/2048.0*100 {
		t.Fatalf("weekly: %+v", windows[1])
	}
}

func TestParseKimiLimitsOmitsMissingWindows(t *testing.T) {
	windows, plan, err := parseKimiLimits([]byte(`{
		"version": "GOODS_VERSION_V2",
		"user": {"membership": {"level": "LEVEL_BASIC"}},
		"usages": {"limit_5h": {"used_ratio": 1.4, "reset_time": "not-a-time"}}
	}`))
	if err != nil || plan != "LEVEL_BASIC" || len(windows) != 1 || windows[0].Used != 100 || windows[0].ResetsAt != "" {
		t.Fatalf("plan=%q windows=%+v err=%v", plan, windows, err)
	}
	windows, plan, err = parseKimiLimits([]byte(`{"user":{"membership":{"level":"LEVEL_BASIC"}},"usage":{"remaining":"1"}}`))
	if err != nil || plan != "免费版" || len(windows) != 0 {
		t.Fatalf("free plan: plan=%q windows=%+v err=%v", plan, windows, err)
	}
}

func TestFetchKimiLimitsUsesRegionalCodingUsage(t *testing.T) {
	var gotPath, gotAuth, gotDevice string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotDevice = r.Header.Get("X-Msh-Device-Id")
		w.Write([]byte(`{"usages":{"limit_5h":{"used_ratio":0.5,"reset_time":"2026-07-02T16:00:00Z"}}}`))
	}))
	defer server.Close()
	previous := kimiUsageHosts
	kimiUsageHosts = map[string]string{"api.kimi.ai": server.URL, "api.kimi.com": server.URL}
	t.Cleanup(func() { kimiUsageHosts = previous })

	usage, err := fetchKimiLimits(t.Context(), "kimi", map[string]any{
		"access_token": "token",
		"domain":       "kimi.ai",
		"device_id":    "device-1",
	})
	if err != nil || gotPath != "/coding/v1/usages" || gotAuth != "Bearer token" || gotDevice != "device-1" {
		t.Fatalf("usage=%+v err=%v path=%s auth=%s device=%s", usage, err, gotPath, gotAuth, gotDevice)
	}
	if len(usage.Windows) != 1 || usage.Windows[0].Used != 50 {
		t.Fatalf("windows=%+v", usage.Windows)
	}
	if got := kimiUsagesURL("kimi-ai", nil); got != server.URL+"/coding/v1/usages" {
		t.Fatalf("international url: %s", got)
	}
	if got := kimiUsagesURL("kimi", map[string]any{"domain": "kimi.com"}); got != server.URL+"/coding/v1/usages" {
		t.Fatalf("china url: %s", got)
	}
}
