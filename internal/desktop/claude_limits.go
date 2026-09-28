package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
)

func fetchClaudeLimits(ctx context.Context, token string) (AccountUsage, error) {
	windows, extra, err := fetchClaudeUsageWindows(ctx, token)
	if err != nil {
		return AccountUsage{}, err
	}
	plan := fetchClaudePlan(ctx, token)
	return AccountUsage{Windows: windows, Plan: plan, Extra: extra}, nil
}

func fetchClaudeUsageWindows(ctx context.Context, token string) ([]UsageWindow, *UsageExtra, error) {
	if token == "" {
		return nil, nil, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	body, err := doLimits(req)
	if err != nil {
		return nil, nil, err
	}
	windows, err := parseClaudeLimits(body)
	if err != nil {
		return nil, nil, err
	}
	if len(windows) == 0 {
		return nil, nil, fmt.Errorf("额度响应里没有可用的窗口")
	}
	return windows, parseClaudeExtra(body), nil
}

// Claude 套餐名来自 profile 接口的组织类型和限额档位；读取失败只损失套餐名，不影响额度。
func fetchClaudePlan(ctx context.Context, token string) string {
	if token == "" {
		return ""
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/api/oauth/profile", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	body, err := doLimits(req)
	if err != nil {
		return ""
	}
	var payload struct {
		Organization struct {
			OrganizationType string `json:"organization_type"`
			RateLimitTier    string `json:"rate_limit_tier"`
		} `json:"organization"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return formatPlanName(payload.Organization.OrganizationType, payload.Organization.RateLimitTier)
}

func formatPlanName(organizationType, rateLimitTier string) string {
	organizationType = strings.TrimSpace(organizationType)
	rateLimitTier = strings.TrimSpace(rateLimitTier)
	base := strings.TrimSpace(strings.TrimPrefix(organizationType, "claude_"))
	if base == "" {
		return ""
	}
	if rateLimitTier == "" {
		return base
	}
	return base + " " + rateLimitTier
}

func parseClaudeExtra(body []byte) *UsageExtra {
	var payload struct {
		Spend struct {
			Enabled bool `json:"enabled"`
			Used    struct {
				AmountMinor *float64 `json:"amount_minor"`
				Currency    string   `json:"currency"`
				Exponent    int      `json:"exponent"`
			} `json:"used"`
		} `json:"spend"`
		Extra struct {
			Enabled       bool     `json:"is_enabled"`
			UsedCredits   *float64 `json:"used_credits"`
			Currency      string   `json:"currency"`
			DecimalPlaces *int     `json:"decimal_places"`
		} `json:"extra_usage"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	if payload.Spend.Enabled && payload.Spend.Used.AmountMinor != nil && payload.Spend.Used.Exponent >= 0 && payload.Spend.Used.Exponent <= 6 {
		used := *payload.Spend.Used.AmountMinor / math.Pow10(payload.Spend.Used.Exponent)
		if used >= 0 && !math.IsInf(used, 0) && !math.IsNaN(used) {
			currency := payload.Spend.Used.Currency
			if currency == "" {
				currency = "USD"
			}
			return &UsageExtra{Amount: used, Currency: currency, Kind: "used"}
		}
	}
	if payload.Extra.Enabled && payload.Extra.UsedCredits != nil {
		places := 2
		if payload.Extra.DecimalPlaces != nil {
			places = *payload.Extra.DecimalPlaces
		}
		if places >= 0 && places <= 6 {
			used := *payload.Extra.UsedCredits / math.Pow10(places)
			if used >= 0 && !math.IsInf(used, 0) && !math.IsNaN(used) {
				currency := payload.Extra.Currency
				if currency == "" {
					currency = "USD"
				}
				return &UsageExtra{Amount: used, Currency: currency, Kind: "used"}
			}
		}
	}
	return nil
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
		if pair.window == nil || pair.window.Utilization == nil {
			continue
		}
		out = append(out, UsageWindow{
			Kind:     pair.kind,
			Used:     clampPercent(*pair.window.Utilization),
			ResetsAt: strings.TrimSpace(pair.window.ResetsAt),
		})
	}
	return out, nil
}

type claudeWindow struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    string   `json:"resets_at"`
}
