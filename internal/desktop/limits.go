package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
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
	case "codex", "claude", "xai", "devin", "antigravity", "kimi", "kimi-ai", "kimi.ai":
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
	default:
		return AccountUsage{}, fmt.Errorf("这个平台没有额度接口")
	}
}

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

// Codex 登录令牌里的 chatgpt_plan_type：prolite 是 Pro 5x，pro 是 Pro 20x。
func codexPlanName(planType string) string {
	planType = strings.TrimSpace(planType)
	if planType == "" {
		return ""
	}
	switch strings.ToLower(planType) {
	case "prolite":
		return "Pro 5x"
	case "pro":
		return "Pro 20x"
	case "self_serve_business_prolite":
		return "Business Premium"
	}
	words := strings.Split(strings.ToLower(planType), "_")
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

func fetchCodexLimits(ctx context.Context, token, accountID string) (AccountUsage, error) {
	if token == "" {
		return AccountUsage{}, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", nil)
	if err != nil {
		return AccountUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if accountID != "" {
		req.Header.Set("ChatGPT-Account-Id", accountID)
	}
	body, err := doLimits(req)
	if err != nil {
		return AccountUsage{}, err
	}
	windows, err := parseCodexLimits(body)
	if err != nil {
		return AccountUsage{}, err
	}
	if len(windows) == 0 {
		return AccountUsage{}, fmt.Errorf("额度响应里没有可用的窗口")
	}
	var payload struct {
		PlanType string `json:"plan_type"`
	}
	_ = json.Unmarshal(body, &payload)
	extra, resets := parseCodexExtras(body)
	return AccountUsage{Windows: windows, Plan: codexPlanName(payload.PlanType), Extra: extra, ResetCredits: resets}, nil
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

// Grok 的周窗口百分比来自 credits 计费接口，套餐名来自设置接口。按月计费的旧账号没有周窗口。
func fetchGrokLimits(ctx context.Context, token string) (AccountUsage, error) {
	if token == "" {
		return AccountUsage{}, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	billing, plan, err := fetchGrokBillingAndPlan(ctx, token)
	if err != nil {
		return AccountUsage{}, err
	}
	windows, err := parseGrokLimits(billing)
	if err != nil {
		return AccountUsage{}, err
	}
	return AccountUsage{Windows: windows, Plan: plan, Extra: parseGrokExtra(billing)}, nil
}

func parseCodexExtras(body []byte) (*UsageExtra, *int) {
	var payload struct {
		Credits struct {
			HasCredits bool            `json:"has_credits"`
			Balance    json.RawMessage `json:"balance"`
		} `json:"credits"`
		Resets struct {
			Available *int `json:"available_count"`
		} `json:"rate_limit_reset_credits"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil, nil
	}
	var extra *UsageExtra
	if payload.Credits.HasCredits {
		var text string
		if json.Unmarshal(payload.Credits.Balance, &text) != nil {
			text = string(payload.Credits.Balance)
		}
		if balance, err := strconv.ParseFloat(strings.TrimSpace(text), 64); err == nil && balance >= 0 && !math.IsInf(balance, 0) && !math.IsNaN(balance) {
			extra = &UsageExtra{Amount: balance, Currency: "credits", Kind: "balance"}
		}
	}
	if payload.Resets.Available != nil && *payload.Resets.Available < 0 {
		payload.Resets.Available = nil
	}
	return extra, payload.Resets.Available
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

func fetchGrokBillingAndPlan(ctx context.Context, token string) ([]byte, string, error) {
	billingBody := make(chan []byte, 1)
	var billingErr error
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		body, err := grokGet(ctx, token, "https://cli-chat-proxy.grok.com/v1/billing?format=credits")
		billingBody <- body
		if err != nil {
			billingErr = err
		}
	}()
	plan := make(chan string, 1)
	go func() {
		defer wg.Done()
		body, err := grokGet(ctx, token, "https://cli-chat-proxy.grok.com/v1/settings")
		if err != nil {
			plan <- ""
			return
		}
		plan <- grokPlanName(body)
	}()
	wg.Wait()
	body := <-billingBody
	if billingErr != nil {
		return nil, <-plan, billingErr
	}
	return body, <-plan, nil
}

func grokGet(ctx context.Context, token, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return doLimits(req)
}

func grokPlanName(body []byte) string {
	var payload struct {
		SubscriptionTierDisplay string `json:"subscription_tier_display"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.SubscriptionTierDisplay)
}

func parseGrokLimits(body []byte) ([]UsageWindow, error) {
	var root map[string]any
	if json.Unmarshal(body, &root) != nil {
		return nil, fmt.Errorf("额度响应无法解析")
	}
	if config, ok := objectField(root, "config"); ok {
		period, ok := objectField(config, "currentPeriod")
		if !ok {
			return nil, fmt.Errorf("额度响应无法解析")
		}
		startText := metadataString(period, "start")
		endText := metadataString(period, "end")
		start, startErr := time.Parse(time.RFC3339Nano, startText)
		end, endErr := time.Parse(time.RFC3339Nano, endText)
		if startErr != nil || endErr != nil || !end.After(start) {
			return nil, fmt.Errorf("额度响应无法解析")
		}
		used := 0.0 // proto-JSON omits an explicit zero.
		if value, found := config["creditUsagePercent"]; found {
			var valid bool
			used, valid = jsonNumber(value)
			if !valid || math.IsNaN(used) || math.IsInf(used, 0) {
				return nil, fmt.Errorf("额度响应无法解析")
			}
		}
		if metadataString(period, "type") != "USAGE_PERIOD_TYPE_WEEKLY" {
			return nil, nil
		}
		return []UsageWindow{{Kind: "weekly", Used: clampPercent(used), ResetsAt: end.UTC().Format(time.RFC3339), DurationSeconds: int64(end.Sub(start).Seconds())}}, nil
	}
	var payload struct {
		PeriodType    string   `json:"period_type"`
		UsedPercent   *float64 `json:"used_percent"`
		PeriodEndSec  int64    `json:"period_end_unix"`
		PeriodSeconds int64    `json:"period_duration_seconds"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("额度响应无法解析")
	}
	if payload.PeriodType != "weekly" || payload.UsedPercent == nil {
		return nil, nil
	}
	window := UsageWindow{Kind: "weekly", Used: clampPercent(*payload.UsedPercent), DurationSeconds: payload.PeriodSeconds}
	if payload.PeriodEndSec > 0 {
		window.ResetsAt = time.Unix(payload.PeriodEndSec, 0).UTC().Format(time.RFC3339)
	}
	return []UsageWindow{window}, nil
}

func parseGrokExtra(body []byte) *UsageExtra {
	var root map[string]any
	if json.Unmarshal(body, &root) != nil {
		return nil
	}
	config, ok := objectField(root, "config")
	if !ok {
		return nil
	}
	capObject, found := config["onDemandCap"]
	if !found {
		return &UsageExtra{Kind: "disabled"}
	}
	capMap, ok := capObject.(map[string]any)
	if !ok {
		return nil
	}
	cap := 0.0
	if raw, found := capMap["val"]; found {
		cap, ok = jsonNumber(raw)
		if !ok || cap < 0 || math.IsNaN(cap) || math.IsInf(cap, 0) {
			return nil
		}
	}
	if cap == 0 {
		return &UsageExtra{Kind: "disabled"}
	}
	return &UsageExtra{Amount: cap, Currency: "credits", Kind: "cap"}
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

// Devin 的额度来自 GetUserStatus（Connect-RPC，JSON 编码）。会话令牌长期有效，无需刷新。
func fetchDevinLimits(ctx context.Context, sessionToken string) (AccountUsage, error) {
	if sessionToken == "" {
		return AccountUsage{}, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	requestBody := map[string]any{
		"metadata": map[string]any{
			"apiKey":           sessionToken,
			"ideName":          "devin",
			"ideVersion":       "1.108.2",
			"extensionName":    "devin",
			"extensionVersion": "1.108.2",
			"locale":           "en",
		},
	}
	encoded, err := json.Marshal(requestBody)
	if err != nil {
		return AccountUsage{}, err
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		"https://server.codeium.com/exa.seat_management_pb.SeatManagementService/GetUserStatus",
		bytes.NewReader(encoded),
	)
	if err != nil {
		return AccountUsage{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s-%s", sessionToken, sessionToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connect-Protocol-Version", "1")
	body, err := doLimits(req)
	if err != nil {
		return AccountUsage{}, err
	}
	windows, plan, err := parseDevinLimits(body)
	return AccountUsage{Windows: windows, Plan: plan, Extra: parseDevinExtra(body)}, err
}

func parseDevinExtra(body []byte) *UsageExtra {
	var payload struct {
		UserStatus struct {
			PlanStatus struct {
				OverageBalanceMicros *float64 `json:"overageBalanceMicros"`
			} `json:"planStatus"`
		} `json:"userStatus"`
	}
	if json.Unmarshal(body, &payload) != nil || payload.UserStatus.PlanStatus.OverageBalanceMicros == nil {
		return nil
	}
	balance := *payload.UserStatus.PlanStatus.OverageBalanceMicros / 1_000_000
	if balance < 0 || math.IsNaN(balance) || math.IsInf(balance, 0) {
		return nil
	}
	return &UsageExtra{Amount: balance, Currency: "USD", Kind: "balance"}
}

func parseDevinLimits(body []byte) ([]UsageWindow, string, error) {
	var payload struct {
		UserStatus struct {
			PlanStatus struct {
				PlanInfo struct {
					PlanName       string `json:"planName"`
					HideDailyQuota bool   `json:"hideDailyQuota"`
				} `json:"planInfo"`
				DailyQuotaRemainingPercent  *float64 `json:"dailyQuotaRemainingPercent"`
				WeeklyQuotaRemainingPercent *float64 `json:"weeklyQuotaRemainingPercent"`
				DailyQuotaResetAtUnix       float64  `json:"dailyQuotaResetAtUnix"`
				WeeklyQuotaResetAtUnix      float64  `json:"weeklyQuotaResetAtUnix"`
			} `json:"planStatus"`
		} `json:"userStatus"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", fmt.Errorf("额度响应无法解析")
	}
	planStatus := payload.UserStatus.PlanStatus
	windows := make([]UsageWindow, 0, 2)
	for _, quota := range []struct {
		kind      string
		remaining *float64
		reset     float64
		hidden    bool
	}{
		{"daily", planStatus.DailyQuotaRemainingPercent, planStatus.DailyQuotaResetAtUnix, planStatus.PlanInfo.HideDailyQuota},
		{"weekly", planStatus.WeeklyQuotaRemainingPercent, planStatus.WeeklyQuotaResetAtUnix, false},
	} {
		if quota.hidden || (quota.remaining == nil && quota.reset <= 0) {
			continue
		}
		// A reset identifies an existing window; protobuf JSON may omit its zero remaining value.
		remaining := 0.0
		if quota.remaining != nil {
			remaining = *quota.remaining
		}
		windows = append(windows, UsageWindow{
			Kind:     quota.kind,
			Used:     clampPercent(100 - remaining),
			ResetsAt: unixSecondsToRFC3339(quota.reset),
		})
	}
	plan := strings.TrimSpace(planStatus.PlanInfo.PlanName)
	if len(windows) == 0 {
		return nil, plan, fmt.Errorf("额度响应里没有可用的窗口")
	}
	return windows, plan, nil
}

func unixSecondsToRFC3339(seconds float64) string {
	if seconds <= 0 {
		return ""
	}
	return time.Unix(int64(seconds), 0).UTC().Format(time.RFC3339)
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
	var durationSeconds int64
	if seconds, hasSeconds := numberField(obj, "limit_window_seconds"); hasSeconds {
		if seconds > 0 && seconds <= 366*24*3600 {
			durationSeconds = int64(seconds)
		}
		if seconds <= 6*3600 {
			kind = "5h"
		} else if seconds >= 27*24*3600 && seconds <= 32*24*3600 {
			kind = "monthly"
		} else {
			kind = "weekly"
		}
	}
	return UsageWindow{
		Kind:            kind,
		Used:            clampPercent(used),
		ResetsAt:        codexReset(obj, now),
		DurationSeconds: durationSeconds,
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
