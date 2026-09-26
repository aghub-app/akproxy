package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

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
