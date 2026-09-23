package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

type reauthTarget struct {
	ID       string
	Provider string
}

type sameAccountStore struct {
	coreauth.Store
	target  reauthTarget
	authDir string
}

func (s sameAccountStore) Save(ctx context.Context, record *coreauth.Auth) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if record == nil || record.ID != s.target.ID || !sameAuthProvider(record.Provider, s.target.Provider) {
		return "", fmt.Errorf("重新授权得到的不是所选账号，请在浏览器中选择原账号")
	}
	if s.authDir != "" {
		if _, err := os.Stat(filepath.Join(s.authDir, s.target.ID)); err != nil {
			return "", fmt.Errorf("原账号已不存在: %w", err)
		}
		record.FileName = s.target.ID
		if record.Attributes != nil {
			delete(record.Attributes, coreauth.AttributePath)
		}
	}
	return s.Store.Save(ctx, record)
}

func sameAuthProvider(got, expected string) bool {
	return got == expected || got == "kimi-ai" && expected == "kimi.ai"
}

func needsReauthorization(record *coreauth.Auth) bool {
	if record == nil || record.Status != coreauth.StatusError {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(record.StatusMessage)) {
	case "unauthorized", "invalid_grant":
		return true
	default:
		return false
	}
}

func reauthProvider(provider string) (string, bool) {
	switch provider {
	case "codex", "claude", "kimi", "kimi-ai", "devin":
		return provider, true
	case "xai":
		return "xai", true
	case "antigravity":
		return "antigravity", true
	case "kimi.ai":
		return "kimi-ai", true
	default:
		return "", false
	}
}

// Match the SDK builder's initial routing selector when supplying our manager.
func authSelector(cfg *config.Config) coreauth.Selector {
	var selector coreauth.Selector = &coreauth.RoundRobinSelector{}
	switch strings.ToLower(strings.TrimSpace(cfg.Routing.Strategy)) {
	case "weighted-round-robin", "weightedroundrobin", "wrr":
		selector = &coreauth.WeightedRoundRobinSelector{}
	case "fill-first", "fillfirst", "ff":
		selector = &coreauth.FillFirstSelector{}
	}
	if !cfg.Routing.SessionAffinity {
		return selector
	}
	ttl := time.Hour
	if value, err := time.ParseDuration(strings.TrimSpace(cfg.Routing.SessionAffinityTTL)); err == nil && value > 0 {
		ttl = max(value, time.Second)
	}
	subagents := true
	if cfg.Routing.SessionAffinitySubagents != nil {
		subagents = *cfg.Routing.SessionAffinitySubagents
	}
	return coreauth.NewSessionAffinitySelectorWithConfig(coreauth.SessionAffinityConfig{
		Fallback:         selector,
		TTL:              ttl,
		SubagentAffinity: &subagents,
	})
}
