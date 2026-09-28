package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

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
