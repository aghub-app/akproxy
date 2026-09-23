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

const (
	kimiSessionSeconds = int64(5 * time.Hour / time.Second)
	kimiWeekSeconds    = int64(7 * 24 * time.Hour / time.Second)
	kimiMonthSeconds   = int64(30 * 24 * time.Hour / time.Second)
	kimiSessionMinutes = 5 * 60
	kimiWeekMinutes    = 7 * 24 * 60
)

var kimiUsageHosts = map[string]string{
	"api.kimi.com": "https://api.kimi.com",
	"api.kimi.ai":  "https://api.kimi.ai",
}

// Kimi 的 Coding Plan 额度来自中国站或国际站的 usages 接口。不读开放平台余额。
func fetchKimiLimits(ctx context.Context, provider string, metadata map[string]any) (AccountUsage, error) {
	token := metadataString(metadata, "access_token")
	if token == "" {
		if nested, ok := metadata["token"].(map[string]any); ok {
			token = metadataString(nested, "access_token")
		}
	}
	if token == "" {
		return AccountUsage{}, fmt.Errorf("这个账号没有可用的登录令牌")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, kimiUsagesURL(provider, metadata), nil)
	if err != nil {
		return AccountUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if deviceID := metadataString(metadata, "device_id"); deviceID != "" {
		req.Header.Set("X-Msh-Device-Id", deviceID)
	}
	body, err := doLimits(req)
	if err != nil {
		return AccountUsage{}, err
	}
	windows, plan, err := parseKimiLimits(body)
	if err != nil {
		return AccountUsage{Plan: plan}, err
	}
	return AccountUsage{Windows: windows, Plan: plan}, nil
}

func kimiUsagesURL(provider string, metadata map[string]any) string {
	host := "api.kimi.com"
	domain := strings.ToLower(metadataString(metadata, "domain"))
	base := strings.ToLower(metadataString(metadata, "base_url"))
	switch {
	case strings.Contains(domain, "kimi.ai") || strings.Contains(base, "kimi.ai"):
		host = "api.kimi.ai"
	case strings.Contains(domain, "kimi.com") || strings.Contains(base, "kimi.com"):
		host = "api.kimi.com"
	case provider == "kimi-ai" || provider == "kimi.ai":
		host = "api.kimi.ai"
	}
	origin := kimiUsageHosts[host]
	if origin == "" {
		origin = "https://" + host
	}
	return strings.TrimRight(origin, "/") + "/coding/v1/usages"
}

func parseKimiLimits(body []byte) ([]UsageWindow, string, error) {
	var root map[string]json.RawMessage
	if json.Unmarshal(body, &root) != nil {
		return nil, "", fmt.Errorf("额度响应无法解析")
	}
	pools := kimiRatioPools(root["usages"])
	weeklyDetail := kimiDetail(root["usage"])
	rateDetail, rateMinutes := kimiFirstLimit(root["limits"])
	monthlyPresent := pools.monthlyKey
	weeklyCounts, weeklyCountsOK := kimiCounts(weeklyDetail)
	weeklyReliable := weeklyCountsOK && weeklyCounts.reliable

	session := kimiResolvedWindow("5h", pools.session, kimiSessionSeconds, int64(rateMinutes)*60, kimiSessionMinutes, rateDetail, rateMinutes, weeklyDetail, weeklyReliable, monthlyPresent)
	weekly := kimiResolvedWindow("weekly", pools.weekly, kimiWeekSeconds, kimiWeekSeconds, kimiWeekMinutes, weeklyDetail, kimiWeekMinutes, weeklyDetail, weeklyReliable, monthlyPresent)
	var monthly *UsageWindow
	if window, ok := kimiRatioWindow("monthly", pools.monthly, kimiMonthSeconds); ok {
		monthly = &window
	}
	out := make([]UsageWindow, 0, 3)
	for _, window := range []*UsageWindow{session, weekly, monthly} {
		if window != nil {
			out = append(out, *window)
		}
	}
	if len(out) == 0 {
		return []UsageWindow{}, "免费版", nil
	}
	return out, kimiPlanName(root), nil
}

type kimiPools struct {
	session    *kimiRatio
	weekly     *kimiRatio
	monthly    *kimiRatio
	monthlyKey bool
}

type kimiRatio struct {
	used  *float64
	reset string
}

type kimiCountDetail struct {
	limit     json.RawMessage
	used      json.RawMessage
	remaining json.RawMessage
	reset     string
}

type kimiCount struct {
	used     int
	limit    int
	reliable bool
}

func kimiRatioPools(raw json.RawMessage) kimiPools {
	if len(raw) == 0 {
		return kimiPools{}
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return kimiPools{}
	}
	_, monthlyKey := object["limit_month_total"]
	return kimiPools{
		session:    kimiRatioValue(object["limit_5h"]),
		weekly:     kimiRatioValue(object["limit_7d"]),
		monthly:    kimiRatioValue(object["limit_month_total"]),
		monthlyKey: monthlyKey,
	}
}

func kimiRatioValue(raw json.RawMessage) *kimiRatio {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return nil
	}
	ratio := &kimiRatio{reset: kimiResetText(object)}
	if value, ok := kimiFloat(object["used_ratio"]); ok {
		ratio.used = &value
	}
	return ratio
}

func kimiDetail(raw json.RawMessage) *kimiCountDetail {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) != nil {
		return nil
	}
	if _, ok := object["limit"]; !ok {
		return nil
	}
	return &kimiCountDetail{
		limit: object["limit"], used: object["used"], remaining: object["remaining"], reset: kimiResetText(object),
	}
}

func kimiFirstLimit(raw json.RawMessage) (*kimiCountDetail, int) {
	if len(raw) == 0 {
		return nil, kimiSessionMinutes
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(raw, &items) != nil || len(items) == 0 {
		return nil, kimiSessionMinutes
	}
	detail := kimiDetail(items[0]["detail"])
	minutes := kimiSessionMinutes
	var window struct {
		Duration int    `json:"duration"`
		TimeUnit string `json:"timeUnit"`
	}
	if json.Unmarshal(items[0]["window"], &window) == nil {
		if parsed, ok := kimiWindowMinutes(window.Duration, window.TimeUnit); ok {
			minutes = parsed
		}
	}
	return detail, minutes
}

func kimiWindowMinutes(duration int, unit string) (int, bool) {
	if duration <= 0 {
		return 0, false
	}
	multiplier := 0
	switch unit {
	case "TIME_UNIT_MINUTE":
		multiplier = 1
	case "TIME_UNIT_HOUR":
		multiplier = 60
	case "TIME_UNIT_DAY":
		multiplier = 24 * 60
	default:
		return 0, false
	}
	if duration > math.MaxInt/multiplier {
		return 0, false
	}
	return duration * multiplier, true
}

func kimiResolvedWindow(kind string, ratio *kimiRatio, ratioDuration, countDuration int64, ratioMinutes int, countDetail *kimiCountDetail, countMinutes int, weeklyDetail *kimiCountDetail, weeklyReliable, monthlyPresent bool) *UsageWindow {
	if ratio != nil {
		if window, ok := kimiRatioWindow(kind, ratio, ratioDuration); ok {
			if !kimiZeroRatioPlaceholder(window, countDetail, countMinutes, ratioMinutes, weeklyDetail, weeklyReliable, monthlyPresent) {
				return &window
			}
		}
	}
	window, ok := kimiCountWindow(kind, countDetail, countDuration)
	if !ok {
		return nil
	}
	return &window
}

func kimiZeroRatioPlaceholder(window UsageWindow, detail *kimiCountDetail, countMinutes, ratioMinutes int, weeklyDetail *kimiCountDetail, weeklyReliable, monthlyPresent bool) bool {
	if window.Used != 0 || monthlyPresent || !weeklyReliable || countMinutes != ratioMinutes {
		return false
	}
	weeklyCounts, ok := kimiCounts(weeklyDetail)
	if !ok || !weeklyCounts.reliable {
		return false
	}
	counts, ok := kimiCounts(detail)
	if !ok || !counts.reliable || counts.used <= 0 || detail == nil {
		return false
	}
	countReset := kimiTime(detail.reset)
	ratioReset := kimiTime(window.ResetsAt)
	if countReset.IsZero() || ratioReset.IsZero() {
		return false
	}
	delta := countReset.Sub(ratioReset)
	if delta < 0 {
		delta = -delta
	}
	return delta <= 2*time.Second
}

func kimiRatioWindow(kind string, ratio *kimiRatio, duration int64) (UsageWindow, bool) {
	if ratio == nil || ratio.used == nil || math.IsNaN(*ratio.used) || math.IsInf(*ratio.used, 0) || *ratio.used < 0 {
		return UsageWindow{}, false
	}
	used := *ratio.used
	if used > 1 {
		used = 1
	}
	return UsageWindow{
		Kind: kind, Used: clampPercent(used * 100), ResetsAt: kimiReset(ratio.reset), DurationSeconds: duration,
	}, true
}

func kimiCountWindow(kind string, detail *kimiCountDetail, duration int64) (UsageWindow, bool) {
	counts, ok := kimiCounts(detail)
	if !ok {
		return UsageWindow{}, false
	}
	window := UsageWindow{
		Kind:     kind,
		Used:     clampPercent(float64(counts.used) / float64(counts.limit) * 100),
		ResetsAt: kimiReset(detail.reset),
	}
	if counts.reliable {
		window.DurationSeconds = duration
	}
	return window, true
}

func kimiCounts(detail *kimiCountDetail) (kimiCount, bool) {
	if detail == nil {
		return kimiCount{}, false
	}
	limit, ok := kimiInt(detail.limit)
	if !ok || limit <= 0 {
		return kimiCount{}, false
	}
	if used, ok := kimiInt(detail.used); ok && used >= 0 {
		return kimiCount{used: used, limit: limit, reliable: true}, true
	}
	if remaining, ok := kimiInt(detail.remaining); ok && remaining >= 0 && remaining <= limit {
		return kimiCount{used: limit - remaining, limit: limit, reliable: true}, true
	}
	return kimiCount{limit: limit}, true
}

func kimiPlanName(root map[string]json.RawMessage) string {
	var user struct {
		Membership struct {
			Level string `json:"level"`
		} `json:"membership"`
	}
	if json.Unmarshal(root["user"], &user) != nil {
		return ""
	}
	level := strings.TrimSpace(user.Membership.Level)
	if level == "" || level == "LEVEL_UNSPECIFIED" {
		return ""
	}
	version := ""
	if raw := root["version"]; len(raw) > 0 && string(raw) != "null" {
		if json.Unmarshal(raw, &version) != nil {
			return level
		}
	}
	if version != "" && version != "GOODS_VERSION_V1" {
		return level
	}
	switch level {
	case "LEVEL_FREE":
		return "Adagio"
	case "LEVEL_TRIAL":
		return "Andante"
	case "LEVEL_BASIC":
		return "Moderato"
	case "LEVEL_INTERMEDIATE":
		return "Allegretto"
	case "LEVEL_ADVANCED":
		return "Allegro"
	default:
		return level
	}
}

func kimiResetText(object map[string]json.RawMessage) string {
	for _, key := range []string{"resetTime", "resetAt", "reset_time", "reset_at"} {
		if text := kimiString(object[key]); text != "" {
			return text
		}
	}
	return ""
}

func kimiString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	if value, ok := kimiFloat(raw); ok && value == math.Trunc(value) {
		return strconv.FormatInt(int64(value), 10)
	}
	return ""
}

func kimiFloat(raw json.RawMessage) (float64, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var value float64
	if json.Unmarshal(raw, &value) == nil && !math.IsNaN(value) && !math.IsInf(value, 0) {
		return value, true
	}
	var text string
	if json.Unmarshal(raw, &text) != nil {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
	if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, false
	}
	return parsed, true
}

func kimiInt(raw json.RawMessage) (int, bool) {
	value, ok := kimiFloat(raw)
	if !ok || value < 0 || value > math.MaxInt || value != math.Trunc(value) {
		return 0, false
	}
	return int(value), true
}

func kimiReset(value string) string {
	parsed := kimiTime(value)
	if parsed.IsZero() {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339)
}

func kimiTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}
