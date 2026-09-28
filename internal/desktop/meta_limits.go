package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

type metaQuotaWindow struct {
	UsedPercent        *float64 `json:"used_percent"`
	ResetsAt           any      `json:"resets_at"`
	WindowDurationMins *float64 `json:"window_duration_mins"`
}

type metaQuotaResponse struct {
	Plan   string `json:"subs_tier_name"`
	PlanID string `json:"subs_tier_id"`
	Usage  struct {
		Window *metaQuotaWindow `json:"window"`
		Weekly *metaQuotaWindow `json:"weekly"`
	} `json:"subs_usage"`
}

func fetchMetaLimits(ctx context.Context, metadata map[string]any) (AccountUsage, error) {
	return fetchMetaLimitsAt(ctx, metadata, metaKeyURL, &http.Client{Timeout: limitsTimeout})
}

func fetchMetaLimitsAt(ctx context.Context, metadata map[string]any, endpoint string, client *http.Client) (AccountUsage, error) {
	token := metadataString(metadata, "dca_token")
	if token == "" {
		return AccountUsage{}, fmt.Errorf("这个账号没有可用的 Meta 授权令牌")
	}
	body, _ := json.Marshal(map[string]string{"dca_token": token})
	var response metaQuotaResponse
	if err := metaPost(ctx, client, endpoint, string(body), "application/json", token, &response, false); err != nil {
		return AccountUsage{}, fmt.Errorf("Meta 额度暂时读不到: %w", err)
	}
	usage := AccountUsage{Plan: response.Plan, Windows: []UsageWindow{}}
	if usage.Plan == "" {
		usage.Plan = response.PlanID
	}
	if window, ok := parseMetaQuotaWindow("rolling", response.Usage.Window); ok {
		usage.Windows = append(usage.Windows, window)
	}
	if window, ok := parseMetaQuotaWindow("weekly", response.Usage.Weekly); ok {
		usage.Windows = append(usage.Windows, window)
	}
	return usage, nil
}

func parseMetaQuotaWindow(kind string, input *metaQuotaWindow) (UsageWindow, bool) {
	if input == nil || input.UsedPercent == nil || math.IsNaN(*input.UsedPercent) || math.IsInf(*input.UsedPercent, 0) || *input.UsedPercent < 0 {
		return UsageWindow{}, false
	}
	duration := int64(7 * 24 * time.Hour / time.Second)
	if kind == "rolling" {
		duration = 0
		if input.WindowDurationMins != nil && *input.WindowDurationMins > 0 {
			duration = int64(*input.WindowDurationMins * 60)
		}
	}
	window := UsageWindow{Kind: kind, Used: clampPercent(*input.UsedPercent), DurationSeconds: duration}
	switch reset := input.ResetsAt.(type) {
	case string:
		if date, err := time.Parse(time.RFC3339, reset); err == nil {
			window.ResetsAt = date.UTC().Format(time.RFC3339)
		}
	case float64:
		if reset > 0 {
			window.ResetsAt = time.Unix(int64(reset), 0).UTC().Format(time.RFC3339)
		}
	}
	return window, true
}
