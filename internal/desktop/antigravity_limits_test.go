package desktop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseAntigravityQuotaSummaryReadsBothPools(t *testing.T) {
	body := []byte(`{"groups":[
		{"buckets":[
			{"bucketId":"3p-weekly","remainingFraction":1,"resetTime":"2026-07-06T07:00:00Z"},
			{"bucketId":"3p-5h","remainingFraction":0.4,"resetTime":"2026-07-02T15:30:00Z"}
		]},
		{"buckets":[
			{"bucketId":"gemini-5h","remainingFraction":0.75,"resetTime":"2026-07-02T16:00:00Z"},
			{"bucketId":"gemini-weekly","remainingFraction":0.9,"resetTime":"2026-07-06T07:00:00Z"}
		]}
	]}`)
	windows, ok := parseAntigravityQuotaSummary(body)
	wrapped, wrappedOK := parseAntigravityQuotaSummary([]byte(`{"response":` + string(body) + `}`))
	if !ok || !wrappedOK {
		t.Fatal("summary did not parse")
	}
	if len(windows) != 4 || len(wrapped) != 4 {
		t.Fatalf("windows: %+v wrapped: %+v", windows, wrapped)
	}
	want := []UsageWindow{
		{Kind: "gemini_5h", Used: 25, ResetsAt: "2026-07-02T16:00:00Z", DurationSeconds: antigravitySessionSeconds},
		{Kind: "gemini_weekly", Used: 10, ResetsAt: "2026-07-06T07:00:00Z", DurationSeconds: antigravityWeekSeconds},
		{Kind: "claude_5h", Used: 60, ResetsAt: "2026-07-02T15:30:00Z", DurationSeconds: antigravitySessionSeconds},
		{Kind: "claude_weekly", Used: 0, ResetsAt: "2026-07-06T07:00:00Z", DurationSeconds: antigravityWeekSeconds},
	}
	for i := range want {
		if windows[i] != want[i] || wrapped[i] != want[i] {
			t.Fatalf("window %d: got %+v wrapped %+v", i, windows[i], wrapped[i])
		}
	}
}

func TestParseAntigravityQuotaSummaryDropsIncompleteBuckets(t *testing.T) {
	body := []byte(`{"groups":[{"buckets":[
		"junk",
		{"bucketId":"3p-5h","remainingFraction":"lots"},
		{"bucketId":"gemini-image-5h","displayName":"Session","remainingFraction":0.1},
		{"bucketId":"gemini-5h","resetTime":"2026-07-02T16:00:00Z"},
		{"bucketId":"gemini-weekly","remainingFraction":0.5,"resetTime":"2026-07-06T07:00:00Z"},
		{"bucketId":"gemini-weekly","remainingFraction":0.1}
	]}]}`)
	windows, ok := parseAntigravityQuotaSummary(body)
	if !ok || len(windows) != 1 || windows[0].Kind != "gemini_weekly" || windows[0].Used != 50 {
		t.Fatalf("incomplete summary: ok=%v windows=%+v", ok, windows)
	}
	if _, ok := parseAntigravityQuotaSummary([]byte(`{}`)); ok {
		t.Fatal("groupless body was treated as a summary")
	}
	windows, ok = parseAntigravityQuotaSummary([]byte(`{"groups":[]}`))
	if !ok || len(windows) != 0 {
		t.Fatalf("empty summary: ok=%v windows=%+v", ok, windows)
	}
}

func TestParseAntigravityModelPoolsKeepsWorstFraction(t *testing.T) {
	body := []byte(`{"models":{
		"gemini-3-pro":{"displayName":"Gemini 3 Pro (High)","quotaInfo":{"remainingFraction":0.8,"resetTime":"2026-07-02T16:00:00Z"}},
		"gemini-flash":{"label":"Gemini Flash","quotaInfo":{"remainingFraction":0.25,"resetTime":"2026-07-02T17:00:00Z"}},
		"claude":{"displayName":"Claude Opus","quotaInfo":{"remainingFraction":0.5,"resetTime":"2026-07-02T18:00:00Z"}},
		"hidden":{"displayName":"Gemini Hidden","model":"MODEL_PLACEHOLDER_M9","quotaInfo":{"remainingFraction":0}},
		"internal":{"displayName":"Gemini Internal","isInternal":true,"quotaInfo":{"remainingFraction":0}}
	}}`)
	windows := parseAntigravityModelPools(body)
	if len(windows) != 2 || windows[0].Kind != "gemini_5h" || windows[0].Used != 75 || windows[0].ResetsAt != "2026-07-02T17:00:00Z" {
		t.Fatalf("gemini pool: %+v", windows)
	}
	if windows[1].Kind != "claude_5h" || windows[1].Used != 50 || windows[1].DurationSeconds != antigravitySessionSeconds {
		t.Fatalf("claude pool: %+v", windows[1])
	}
}

func TestFormatAntigravityPlan(t *testing.T) {
	if got := formatAntigravityPlan("Google AI Pro"); got != "Pro" {
		t.Fatalf("google ai: %q", got)
	}
	if got := formatAntigravityPlan("Gemini Code Assist in Google One AI Pro"); got != "Pro" {
		t.Fatalf("code assist: %q", got)
	}
	if got := parseAntigravityPlan([]byte(`{"paidTier":{"name":"Google AI Ultra"},"currentTier":{"name":"Free"}}`)); got != "Ultra" {
		t.Fatalf("paid tier: %q", got)
	}
}

func TestFetchAntigravityLimitsStopsAfterAuthoritativeSummary(t *testing.T) {
	var sawModels bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1internal:retrieveUserQuotaSummary":
			w.Write([]byte(`{"groups":[]}`))
		case "/v1internal:loadCodeAssist":
			w.Write([]byte(`{"paidTier":{"name":"Google AI Pro"}}`))
		default:
			sawModels = true
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	previous := antigravityQuotaBases
	antigravityQuotaBases = []string{server.URL}
	t.Cleanup(func() { antigravityQuotaBases = previous })

	usage, err := fetchAntigravityLimits(t.Context(), map[string]any{"access_token": "token"})
	if err != nil || len(usage.Windows) != 0 || usage.Plan != "Pro" || sawModels {
		t.Fatalf("usage=%+v err=%v sawModels=%v", usage, err, sawModels)
	}
}

func TestFetchAntigravityLimitsRefreshesRejectedToken(t *testing.T) {
	summaries := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Write([]byte(`{"access_token":"fresh"}`))
		case "/v1internal:retrieveUserQuotaSummary":
			summaries++
			if r.Header.Get("Authorization") != "Bearer fresh" {
				http.Error(w, "no", http.StatusUnauthorized)
				return
			}
			w.Write([]byte(`{"groups":[{"buckets":[{"bucketId":"gemini-5h","remainingFraction":0.5}]}]}`))
		case "/v1internal:loadCodeAssist":
			w.Write([]byte(`{}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	previousBases := antigravityQuotaBases
	previousToken := antigravityTokenEndpoint
	antigravityQuotaBases = []string{server.URL}
	antigravityTokenEndpoint = server.URL + "/token"
	t.Cleanup(func() {
		antigravityQuotaBases = previousBases
		antigravityTokenEndpoint = previousToken
	})

	usage, err := fetchAntigravityLimits(t.Context(), map[string]any{
		"access_token": "stale", "refresh_token": "refresh",
	})
	if err != nil || summaries != 2 || len(usage.Windows) != 1 || usage.Windows[0].Used != 50 {
		t.Fatalf("usage=%+v err=%v summaries=%d", usage, err, summaries)
	}
}
