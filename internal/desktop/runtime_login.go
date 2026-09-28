package desktop

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// Login opens the system browser for one upstream.
func (r *Runtime) Login(provider string) error {
	loginProvider, ok := loginProviders[provider]
	if !ok {
		return fmt.Errorf("这个上游不能用浏览器登录")
	}
	return r.login(loginProvider, nil)
}

// ReauthorizeAccount updates only the selected saved browser login.
func (r *Runtime) ReauthorizeAccount(id string) error {
	if !filepath.IsLocal(id) || strings.TrimSpace(id) == "" {
		return fmt.Errorf("账号无效")
	}
	records, err := r.accountRecords()
	if err != nil {
		return err
	}
	for _, record := range records {
		if record == nil || record.ID != id {
			continue
		}
		provider, ok := reauthProvider(record.Provider)
		if !ok {
			return fmt.Errorf("这个上游不能用浏览器登录")
		}
		return r.login(provider, &reauthTarget{ID: id, Provider: record.Provider})
	}
	return fmt.Errorf("账号不存在")
}

func (r *Runtime) login(loginProvider string, target *reauthTarget) error {
	r.loginMu.Lock()
	if r.loginCancel != nil {
		r.loginMu.Unlock()
		return fmt.Errorf("已经有一个登录在进行")
	}
	timeout := 6 * time.Minute
	if loginProvider == "meta" {
		timeout = 15 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	r.loginCancel = cancel
	r.loginMu.Unlock()
	r.emit("server:status", mustStatus(r))

	release := func() {
		cancel()
		r.loginMu.Lock()
		r.loginCancel = nil
		r.loginMu.Unlock()
		r.emit("server:status", mustStatus(r))
	}

	type outcome struct {
		record *coreauth.Auth
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		record, err := r.runLogin(ctx, loginProvider, target)
		done <- outcome{record: record, err: err}
	}()

	var result outcome
	select {
	case result = <-done:
	case <-ctx.Done():
		unblockCallbackLogin(loginProvider)
		release()
		return fmt.Errorf("登录已取消")
	}
	release()

	if result.err != nil {
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			return fmt.Errorf("登录已取消")
		}
		return fmt.Errorf("登录失败: %w", result.err)
	}
	if result.record == nil {
		return fmt.Errorf("登录失败: 上游没有返回账号")
	}
	label := accountLabel(result.record)
	r.emit("login:done", Account{ID: result.record.ID, Provider: result.record.Provider, Label: label})
	return nil
}

// loginWithSDK runs the SDK browser login. Some authenticators ignore context
// cancellation, so the caller must not hold the login slot while it blocks.
func (r *Runtime) loginWithSDK(ctx context.Context, provider string, target *reauthTarget) (*coreauth.Auth, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	cfg.AuthDir = r.paths.Auth
	store := auth.GetTokenStore()
	if setter, ok := store.(interface{ SetBaseDir(string) }); ok {
		setter.SetBaseDir(r.paths.Auth)
	}
	if target != nil {
		store = sameAccountStore{Store: store, target: *target, authDir: r.paths.Auth}
	} else if provider == "meta" {
		store = metaCancelStore{Store: store}
	}
	manager := auth.NewManager(store, newAuthenticators(r.emit)...)
	record, _, err := manager.Login(ctx, provider, cfg, &auth.LoginOptions{})
	if err != nil {
		return nil, err
	}
	return record, nil
}

// CancelLogin stops the browser login that is waiting.
func (r *Runtime) CancelLogin() {
	r.loginMu.Lock()
	cancel := r.loginCancel
	r.loginMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// callbackLoginPorts maps providers whose authenticators wait on a local HTTP
// callback and ignore context cancellation. Cancelling leaves that callback
// server bound until its own timeout; a synthetic error callback makes the
// authenticator return at once and release the port for the next login.
var callbackLoginPorts = map[string]struct {
	port int
	path string
}{
	"codex":       {port: 1455, path: "/auth/callback"},
	"claude":      {port: 54545, path: "/callback"},
	"antigravity": {port: 51121, path: "/oauth-callback"},
}

func unblockCallbackLogin(provider string) {
	target, ok := callbackLoginPorts[provider]
	if !ok {
		return
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s?error=login_cancelled", target.port, target.path)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return
	}
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

var loginProviders = map[string]string{
	"codex":   "codex",
	"grok":    "xai",
	"claude":  "claude",
	"gemini":  "antigravity",
	"kimi":    "kimi",
	"kimi-ai": "kimi-ai",
	"devin":   "devin",
	"meta":    "meta",
}

func newAuthenticators(emit func(string, any)) []auth.Authenticator {
	return []auth.Authenticator{
		auth.NewCodexAuthenticator(),
		auth.NewClaudeAuthenticator(),
		auth.NewAntigravityAuthenticator(),
		auth.NewKimiAuthenticator(),
		auth.NewKimiAIAuthenticator(),
		auth.NewXAIAuthenticator(),
		auth.NewDevinAuthenticator(),
		metaAuthenticator{emit: emit},
	}
}
