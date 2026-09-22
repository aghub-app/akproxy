package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const limitsTimeout = 15 * time.Second

// UsageWindow is one vendor rate-limit window. Used is the percent already consumed, 0–100.
type UsageWindow struct {
	Kind     string  `json:"kind"`
	Used     float64 `json:"used"`
	ResetsAt string  `json:"resetsAt"`
}

// AccountUsage is the session-limit reading for one saved login.
type AccountUsage struct {
	ID      string        `json:"id"`
	Windows []UsageWindow `json:"windows"`
	Error   string        `json:"error"`
}

func supportsSessionLimit(provider string) bool {
	switch provider {
	case "codex", "claude":
		return true
	default:
		return false
	}
}

func sessionLimits(ctx context.Context, provider string, metadata map[string]any) ([]UsageWindow, error) {
	token := metadataString(metadata, "access_token")
	if token == "" {
		if nested, ok := metadata["token"].(map[string]any); ok {
			token = metadataString(nested, "access_token")
		}
	}
	if token == "" {
		return nil, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	switch provider {
	case "claude":
		return fetchClaudeLimits(ctx, token)
	case "codex":
		return fetchCodexLimits(ctx, token, metadataString(metadata, "account_id"))
	default:
		return nil, fmt.Errorf("这个平台没有额度接口")
	}
}

func fetchClaudeLimits(ctx context.Context, token string) ([]UsageWindow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	body, err := doLimits(req)
	if err != nil {
		return nil, err
	}
	windows, err := parseClaudeLimits(body)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("额度响应里没有可用的窗口")
	}
	return windows, nil
}

func fetchCodexLimits(ctx context.Context, token, accountID string) ([]UsageWindow, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	body, err := doLimits(req)
	if err != nil {
		return nil, err
	}
	windows, err := parseCodexLimits(body)
	if err != nil {
		return nil, err
	}
	if len(windows) == 0 {
		return nil, fmt.Errorf("额度响应里没有可用的窗口")
	}
	return windows, nil
}

func doLimits(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: limitsTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("额度暂时读不到")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("额度暂时读不到")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("额度接口返回 %d", resp.StatusCode)
	}
	return body, nil
}

func parseClaudeLimits(body []byte) ([]UsageWindow, error) {
	var payload struct {
		FiveHour       *claudeWindow `json:"five_hour"`
		SevenDay       *claudeWindow `json:"seven_day"`
		SevenDayOpus   *claudeWindow `json:"seven_day_opus"`
		SevenDaySonnet *claudeWindow `json:"seven_day_sonnet"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("额度响应无法解析")
	}
	pairs := []struct {
		kind   string
		window *claudeWindow
	}{
		{"5h", payload.FiveHour},
		{"weekly", payload.SevenDay},
		{"weekly_opus", payload.SevenDayOpus},
		{"weekly_sonnet", payload.SevenDaySonnet},
	}
	out := make([]UsageWindow, 0, len(pairs))
	for _, pair := range pairs {
		if pair.window == nil {
			continue
		}
		out = append(out, UsageWindow{
			Kind:     pair.kind,
			Used:     clampPercent(pair.window.Utilization),
			ResetsAt: strings.TrimSpace(pair.window.ResetsAt),
		})
	}
	return out, nil
}

type claudeWindow struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    string  `json:"resets_at"`
}

func parseCodexLimits(body []byte) ([]UsageWindow, error) {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("额度响应无法解析")
	}
	root := payload
	if nested, ok := objectField(payload, "rate_limit"); ok {
		root = nested
	} else if nested, ok := objectField(payload, "rate_limits"); ok {
		root = nested
	}
	pairs := []struct {
		keys     []string
		fallback string
	}{
		{[]string{"primary_window", "primary"}, "5h"},
		{[]string{"secondary_window", "secondary"}, "weekly"},
	}
	out := make([]UsageWindow, 0, len(pairs))
	for _, pair := range pairs {
		var raw any
		var found bool
		for _, key := range pair.keys {
			if value, ok := root[key]; ok {
				raw = value
				found = true
				break
			}
		}
		if !found {
			continue
		}
		window, ok := codexWindow(raw, pair.fallback, time.Now())
		if ok {
			out = append(out, window)
		}
	}
	return out, nil
}

func codexWindow(raw any, fallback string, now time.Time) (UsageWindow, bool) {
	obj, ok := raw.(map[string]any)
	if !ok {
		return UsageWindow{}, false
	}
	used, ok := numberField(obj, "used_percent")
	if !ok {
		left, hasLeft := numberField(obj, "percent_left")
		if !hasLeft {
			return UsageWindow{}, false
		}
		used = 100 - left
	}
	kind := fallback
	if seconds, hasSeconds := numberField(obj, "limit_window_seconds"); hasSeconds {
		if seconds <= 6*3600 {
			kind = "5h"
		} else {
			kind = "weekly"
		}
	}
	return UsageWindow{
		Kind:     kind,
		Used:     clampPercent(used),
		ResetsAt: codexReset(obj, now),
	}, true
}

func codexReset(obj map[string]any, now time.Time) string {
	if text := metadataString(obj, "resets_at"); text != "" {
		return text
	}
	if seconds, ok := numberField(obj, "reset_at"); ok {
		return time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
	}
	if seconds, ok := numberField(obj, "resets_in_seconds"); ok {
		return now.UTC().Add(time.Duration(seconds) * time.Second).Format(time.RFC3339)
	}
	if seconds, ok := numberField(obj, "reset_after_seconds"); ok {
		return now.UTC().Add(time.Duration(seconds) * time.Second).Format(time.RFC3339)
	}
	if millis, ok := numberField(obj, "reset_time_ms"); ok {
		return time.UnixMilli(int64(millis)).UTC().Format(time.RFC3339)
	}
	return ""
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	text, ok := metadata[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func objectField(metadata map[string]any, key string) (map[string]any, bool) {
	value, ok := metadata[key].(map[string]any)
	return value, ok
}

func numberField(metadata map[string]any, key string) (float64, bool) {
	switch value := metadata[key].(type) {
	case float64:
		return value, true
	case json.Number:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
