package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	antigravitySessionSeconds = int64(5 * time.Hour / time.Second)
	antigravityWeekSeconds    = int64(7 * 24 * time.Hour / time.Second)
	// Installed-app OAuth client shipped with Antigravity. The secret is not confidential.
	antigravityOAuthClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	antigravityOAuthClientSecret = "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"
)

var antigravityTokenEndpoint = "https://oauth2.googleapis.com/token"

var antigravityQuotaBases = []string{
	"https://daily-cloudcode-pa.googleapis.com",
	"https://cloudcode-pa.googleapis.com",
}

var antigravityModelBlacklist = map[string]bool{
	"MODEL_CHAT_20706": true, "MODEL_CHAT_23310": true,
	"MODEL_GOOGLE_GEMINI_2_5_FLASH": true, "MODEL_GOOGLE_GEMINI_2_5_FLASH_THINKING": true,
	"MODEL_GOOGLE_GEMINI_2_5_FLASH_LITE": true, "MODEL_GOOGLE_GEMINI_2_5_PRO": true,
	"MODEL_PLACEHOLDER_M19": true, "MODEL_PLACEHOLDER_M9": true, "MODEL_PLACEHOLDER_M12": true,
}

// Gemini 额度读取 Antigravity 的 Cloud Code 配额。可解析的摘要不再退回旧接口。
func fetchAntigravityLimits(ctx context.Context, metadata map[string]any) (AccountUsage, error) {
	token := antigravityAccessToken(metadata)
	refresh := metadataString(metadata, "refresh_token")
	if token == "" {
		if refresh == "" {
			return AccountUsage{}, fmt.Errorf("这个账号没有可用的登录令牌")
		}
		var err error
		token, err = refreshAntigravityAccess(ctx, refresh)
		if err != nil {
			return AccountUsage{}, err
		}
	}
	usage, status, err := readAntigravityQuota(ctx, token, metadataString(metadata, "project_id"))
	if (status == http.StatusUnauthorized || status == http.StatusForbidden) && refresh != "" {
		refreshed, refreshErr := refreshAntigravityAccess(ctx, refresh)
		if refreshErr != nil {
			return AccountUsage{}, refreshErr
		}
		usage, _, err = readAntigravityQuota(ctx, refreshed, metadataString(metadata, "project_id"))
	}
	return usage, err
}

func antigravityAccessToken(metadata map[string]any) string {
	token := metadataString(metadata, "access_token")
	if token == "" {
		if nested, ok := metadata["token"].(map[string]any); ok {
			token = metadataString(nested, "access_token")
		}
	}
	return token
}

func readAntigravityQuota(ctx context.Context, token, projectID string) (AccountUsage, int, error) {
	summary := antigravityPost(ctx, "/v1internal:retrieveUserQuotaSummary", token, "antigravity", map[string]any{})
	if summary.status == http.StatusUnauthorized || summary.status == http.StatusForbidden {
		return AccountUsage{}, summary.status, fmt.Errorf("额度接口返回 %d", summary.status)
	}
	if summary.status >= 200 && summary.status < 300 {
		windows, ok := parseAntigravityQuotaSummary(summary.body)
		if ok {
			return AccountUsage{Windows: windows, Plan: fetchAntigravityPlan(ctx, token)}, summary.status, nil
		}
	}

	models := antigravityPost(ctx, "/v1internal:fetchAvailableModels", token, "antigravity", map[string]any{})
	if models.status == http.StatusUnauthorized || models.status == http.StatusForbidden {
		return AccountUsage{}, models.status, fmt.Errorf("额度接口返回 %d", models.status)
	}
	if models.status >= 200 && models.status < 300 {
		if windows := parseAntigravityModelPools(models.body); len(windows) > 0 {
			return AccountUsage{Windows: windows, Plan: fetchAntigravityPlan(ctx, token)}, models.status, nil
		}
	}

	assist := antigravityPost(ctx, "/v1internal:loadCodeAssist", token, "agy", map[string]any{})
	if assist.status == http.StatusUnauthorized || assist.status == http.StatusForbidden {
		return AccountUsage{}, assist.status, fmt.Errorf("额度接口返回 %d", assist.status)
	}
	plan := ""
	if assist.status >= 200 && assist.status < 300 {
		plan = parseAntigravityPlan(assist.body)
		if discovered := parseAntigravityProject(assist.body); discovered != "" {
			projectID = discovered
		}
	}
	body := map[string]any{}
	if projectID != "" {
		body["project"] = projectID
	}
	quota := antigravityPost(ctx, "/v1internal:retrieveUserQuota", token, "agy", body)
	if quota.status == http.StatusUnauthorized || quota.status == http.StatusForbidden {
		return AccountUsage{}, quota.status, fmt.Errorf("额度接口返回 %d", quota.status)
	}
	if (quota.status < 200 || quota.status >= 300) && projectID != "" {
		quota = antigravityPost(ctx, "/v1internal:retrieveUserQuota", token, "agy", map[string]any{})
		if quota.status == http.StatusUnauthorized || quota.status == http.StatusForbidden {
			return AccountUsage{}, quota.status, fmt.Errorf("额度接口返回 %d", quota.status)
		}
	}
	if quota.status >= 200 && quota.status < 300 {
		if windows := parseAntigravityQuotaBuckets(quota.body); len(windows) > 0 {
			return AccountUsage{Windows: windows, Plan: plan}, quota.status, nil
		}
	}
	if summary.err != nil && models.err != nil && quota.err != nil {
		return AccountUsage{}, 0, fmt.Errorf("额度暂时读不到")
	}
	return AccountUsage{Plan: plan}, 0, fmt.Errorf("额度响应里没有可用的窗口")
}

func fetchAntigravityPlan(ctx context.Context, token string) string {
	assist := antigravityPost(ctx, "/v1internal:loadCodeAssist", token, "agy", map[string]any{})
	if assist.status < 200 || assist.status >= 300 {
		return ""
	}
	return parseAntigravityPlan(assist.body)
}

type antigravityCall struct {
	body   []byte
	status int
	err    error
}

func antigravityPost(ctx context.Context, path, token, userAgent string, body any) antigravityCall {
	payload, err := json.Marshal(body)
	if err != nil {
		return antigravityCall{err: err}
	}
	var last antigravityCall
	for _, base := range antigravityQuotaBases {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, bytes.NewReader(payload))
		if err != nil {
			last = antigravityCall{err: err}
			continue
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("User-Agent", userAgent)
		call := antigravityDo(req)
		if call.status == http.StatusUnauthorized || call.status == http.StatusForbidden || (call.status >= 200 && call.status < 300) {
			return call
		}
		last = call
	}
	return last
}

func antigravityDo(req *http.Request) antigravityCall {
	client := &http.Client{Timeout: limitsTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return antigravityCall{err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return antigravityCall{status: resp.StatusCode, err: err}
	}
	return antigravityCall{body: body, status: resp.StatusCode}
}

func refreshAntigravityAccess(ctx context.Context, refreshToken string) (string, error) {
	form := url.Values{}
	form.Set("client_id", antigravityOAuthClientID)
	form.Set("client_secret", antigravityOAuthClientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, antigravityTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	call := antigravityDo(req)
	if call.err != nil || call.status == http.StatusRequestTimeout || call.status == http.StatusTooManyRequests || call.status >= 500 || call.status == 0 {
		return "", fmt.Errorf("额度暂时读不到")
	}
	if call.status < 200 || call.status >= 300 {
		return "", fmt.Errorf("这个账号没有可用的登录令牌")
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if json.Unmarshal(call.body, &payload) != nil || strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("额度暂时读不到")
	}
	return strings.TrimSpace(payload.AccessToken), nil
}

func parseAntigravityQuotaSummary(body []byte) ([]UsageWindow, bool) {
	var root map[string]json.RawMessage
	if json.Unmarshal(body, &root) != nil {
		return nil, false
	}
	raw, ok := root["groups"]
	if !ok {
		response, hasResponse := root["response"]
		if !hasResponse {
			return nil, false
		}
		var inner map[string]json.RawMessage
		if json.Unmarshal(response, &inner) != nil {
			return nil, false
		}
		raw, ok = inner["groups"]
		if !ok {
			return nil, false
		}
	}
	var groups []struct {
		Buckets []json.RawMessage `json:"buckets"`
	}
	if json.Unmarshal(raw, &groups) != nil {
		return nil, false
	}
	pooled := map[string]UsageWindow{}
	order := []string{"gemini_5h", "gemini_weekly", "claude_5h", "claude_weekly"}
	ids := map[string]string{
		"gemini-5h": "gemini_5h", "gemini-weekly": "gemini_weekly",
		"3p-5h": "claude_5h", "3p-weekly": "claude_weekly",
	}
	for _, group := range groups {
		for _, rawBucket := range group.Buckets {
			var bucket struct {
				BucketID  string          `json:"bucketId"`
				Remaining json.RawMessage `json:"remainingFraction"`
				ResetTime string          `json:"resetTime"`
			}
			if json.Unmarshal(rawBucket, &bucket) != nil || ids[bucket.BucketID] == "" {
				continue
			}
			kind := ids[bucket.BucketID]
			if _, seen := pooled[kind]; seen || len(bucket.Remaining) == 0 {
				continue
			}
			fraction, ok := antigravityFraction(bucket.Remaining)
			if !ok {
				continue
			}
			pooled[kind] = antigravityWindow(kind, fraction, bucket.ResetTime)
		}
	}
	out := make([]UsageWindow, 0, len(order))
	for _, kind := range order {
		if window, found := pooled[kind]; found {
			out = append(out, window)
		}
	}
	return out, true
}

func parseAntigravityModelPools(body []byte) []UsageWindow {
	var payload struct {
		Models map[string]json.RawMessage `json:"models"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	configs := make([]antigravityModel, 0, len(payload.Models))
	for key, raw := range payload.Models {
		var model struct {
			Model       string `json:"model"`
			DisplayName string `json:"displayName"`
			Label       string `json:"label"`
			IsInternal  bool   `json:"isInternal"`
			QuotaInfo   *struct {
				Remaining *float64 `json:"remainingFraction"`
				ResetTime string   `json:"resetTime"`
			} `json:"quotaInfo"`
		}
		if json.Unmarshal(raw, &model) != nil || model.IsInternal {
			continue
		}
		label := strings.TrimSpace(model.DisplayName)
		if label == "" {
			label = strings.TrimSpace(model.Label)
		}
		modelID := strings.TrimSpace(model.Model)
		if modelID == "" {
			modelID = key
		}
		remaining := 0.0
		reset := ""
		if model.QuotaInfo != nil {
			if model.QuotaInfo.Remaining != nil {
				remaining = *model.QuotaInfo.Remaining
			}
			reset = model.QuotaInfo.ResetTime
		}
		configs = append(configs, antigravityModel{label: label, modelID: modelID, remaining: remaining, reset: reset})
	}
	return poolAntigravityModels(configs)
}

func parseAntigravityQuotaBuckets(body []byte) []UsageWindow {
	var payload struct {
		Buckets []struct {
			ModelID   string   `json:"modelId"`
			Remaining *float64 `json:"remainingFraction"`
			ResetTime string   `json:"resetTime"`
		} `json:"buckets"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return nil
	}
	configs := make([]antigravityModel, 0, len(payload.Buckets))
	for _, bucket := range payload.Buckets {
		remaining := 0.0
		if bucket.Remaining != nil {
			remaining = *bucket.Remaining
		}
		configs = append(configs, antigravityModel{
			label: bucket.ModelID, modelID: bucket.ModelID, remaining: remaining, reset: bucket.ResetTime,
		})
	}
	return poolAntigravityModels(configs)
}

type antigravityModel struct {
	label     string
	modelID   string
	remaining float64
	reset     string
}

func poolAntigravityModels(configs []antigravityModel) []UsageWindow {
	type pool struct {
		remaining float64
		reset     string
		seen      bool
	}
	pools := map[string]*pool{}
	for _, config := range configs {
		label := antigravityNormalizeLabel(config.label)
		if label == "" || antigravityModelBlacklist[config.modelID] {
			continue
		}
		kind := "claude_5h"
		if strings.Contains(strings.ToLower(label), "gemini") {
			kind = "gemini_5h"
		}
		current := pools[kind]
		if current == nil || config.remaining < current.remaining {
			pools[kind] = &pool{remaining: config.remaining, reset: config.reset, seen: true}
		}
	}
	out := make([]UsageWindow, 0, 2)
	for _, kind := range []string{"gemini_5h", "claude_5h"} {
		if item := pools[kind]; item != nil {
			out = append(out, antigravityWindow(kind, item.remaining, item.reset))
		}
	}
	return out
}

func antigravityWindow(kind string, remaining float64, reset string) UsageWindow {
	if math.IsNaN(remaining) || math.IsInf(remaining, 0) {
		remaining = 0
	}
	if remaining < 0 {
		remaining = 0
	}
	if remaining > 1 {
		remaining = 1
	}
	duration := antigravityWeekSeconds
	if strings.HasSuffix(kind, "_5h") {
		duration = antigravitySessionSeconds
	}
	return UsageWindow{
		Kind:            kind,
		Used:            clampPercent(math.Round((1 - remaining) * 100)),
		ResetsAt:        antigravityReset(reset),
		DurationSeconds: duration,
	}
}

func antigravityFraction(raw json.RawMessage) (float64, bool) {
	var value float64
	if json.Unmarshal(raw, &value) != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func antigravityReset(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, value)
	}
	if err != nil {
		return ""
	}
	return parsed.UTC().Format(time.RFC3339)
}

func antigravityNormalizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if open := strings.LastIndex(label, "("); open > 0 && strings.HasSuffix(label, ")") {
		label = strings.TrimSpace(label[:open])
	}
	return label
}

func parseAntigravityPlan(body []byte) string {
	var payload struct {
		PaidTier    *struct{ Name string } `json:"paidTier"`
		CurrentTier *struct{ Name string } `json:"currentTier"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	raw := ""
	if payload.PaidTier != nil {
		raw = payload.PaidTier.Name
	}
	if strings.TrimSpace(raw) == "" && payload.CurrentTier != nil {
		raw = payload.CurrentTier.Name
	}
	return formatAntigravityPlan(raw)
}

func parseAntigravityProject(body []byte) string {
	var payload struct {
		Project string `json:"cloudaicompanionProject"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return strings.TrimSpace(payload.Project)
}

func formatAntigravityPlan(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if rest, ok := strings.CutPrefix(trimmed, "Google AI "); ok {
		return antigravityTitle(rest)
	}
	lower := strings.ToLower(trimmed)
	for _, keyword := range []string{"Ultra", "Pro", "Free"} {
		if strings.Contains(lower, strings.ToLower(keyword)) {
			return keyword
		}
	}
	return antigravityTitle(trimmed)
}

func antigravityTitle(value string) string {
	parts := strings.Fields(value)
	for i, part := range parts {
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}
