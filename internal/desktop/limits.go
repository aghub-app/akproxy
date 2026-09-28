package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const limitsTimeout = 15 * time.Second

// UsageWindow is one vendor rate-limit window. Used is the percent already consumed, 0–100.
type UsageWindow struct {
	Kind            string  `json:"kind"`
	Used            float64 `json:"used"`
	ResetsAt        string  `json:"resetsAt"`
	DurationSeconds int64   `json:"durationSeconds"`
}

// AccountUsage is the session-limit reading for one saved login.
type AccountUsage struct {
	ID           string        `json:"id"`
	Plan         string        `json:"plan"`
	Windows      []UsageWindow `json:"windows"`
	Extra        *UsageExtra   `json:"extra"`
	ResetCredits *int          `json:"resetCredits"`
	Error        string        `json:"error"`
}

type UsageExtra struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Kind     string  `json:"kind"`
}

func supportsSessionLimit(provider string) bool {
	switch provider {
	case "codex", "claude", "xai", "devin", "antigravity", "kimi", "kimi-ai", "kimi.ai", "meta":
		return true
	default:
		return false
	}
}

func sessionLimits(ctx context.Context, provider string, metadata map[string]any) (AccountUsage, error) {
	token := metadataString(metadata, "access_token")
	if token == "" {
		if nested, ok := metadata["token"].(map[string]any); ok {
			token = metadataString(nested, "access_token")
		}
	}
	switch provider {
	case "claude":
		return fetchClaudeLimits(ctx, token)
	case "codex":
		return fetchCodexLimits(ctx, token, metadataString(metadata, "account_id"))
	case "xai":
		return fetchGrokLimits(ctx, token)
	case "devin":
		return fetchDevinLimits(ctx, metadataString(metadata, "session_token"))
	case "antigravity":
		return fetchAntigravityLimits(ctx, metadata)
	case "kimi", "kimi-ai", "kimi.ai":
		return fetchKimiLimits(ctx, provider, metadata)
	case "meta":
		return fetchMetaLimits(ctx, metadata)
	default:
		return AccountUsage{}, fmt.Errorf("这个平台没有额度接口")
	}
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

func jsonNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
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
