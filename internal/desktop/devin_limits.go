package desktop

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

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
