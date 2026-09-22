package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

// Runtime owns the saved config and the embedded proxy process.
type Runtime struct {
	paths Paths

	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
	running  bool
	bound    Listen
	runError string

	loginMu     sync.Mutex
	loginCancel context.CancelFunc
	runLogin    func(ctx context.Context, provider string) (*coreauth.Auth, error)

	emit func(event string, data any)
}

// NewRuntime prepares app-owned directories and the initial config.
func NewRuntime(emit func(event string, data any)) (*Runtime, error) {
	paths, err := Resolve()
	if err != nil {
		return nil, err
	}
	if err := Ensure(paths); err != nil {
		return nil, err
	}
	if emit == nil {
		emit = func(string, any) {}
	}
	r := &Runtime{paths: paths, emit: emit}
	r.runLogin = r.loginWithSDK
	return r, nil
}

// Status reports the navbar button and the address clients should use.
func (r *Runtime) Status() (Status, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return Status{}, err
	}
	saved := SavedListen(cfg)
	savedAddress, _ := ClientURL(saved)

	r.mu.Lock()
	defer r.mu.Unlock()
	action := ControlAction(r.running, r.bound, saved)
	shown := saved
	if r.running {
		shown = r.bound
	}
	address, all := ClientURL(shown)
	return Status{
		Running:         r.running,
		Action:          action,
		Address:         address,
		AllInterfaces:   all,
		RestartRequired: action == "restart",
		SavedAddress:    savedAddress,
		Error:           r.runError,
	}, nil
}

// ServiceSettings returns the 服务 page.
func (r *Runtime) ServiceSettings() (ServiceSettings, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return ServiceSettings{}, err
	}
	return ReadService(cfg), nil
}

// SaveService writes the 服务 page.
func (r *Runtime) SaveService(in ServiceSettings) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyService(cfg, in, r.paths.Auth)
	})
}

// ProviderKeys returns API keys for codex, grok, claude, or gemini.
func (r *Runtime) ProviderKeys(provider string) ([]KeyDraft, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	return ReadKeys(cfg, provider)
}

// SaveProviderKeys writes one native provider's API keys.
func (r *Runtime) SaveProviderKeys(provider string, drafts []KeyDraft) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyKeys(cfg, provider, drafts, r.paths.Auth)
	})
}

// KimiProviders returns the Kimi page.
func (r *Runtime) KimiProviders() ([]OpenAIDraft, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	return ReadKimi(cfg), nil
}

// SaveKimi writes the Kimi page.
func (r *Runtime) SaveKimi(drafts []OpenAIDraft) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyKimi(cfg, drafts, r.paths.Auth)
	})
}

// OpenAIProviders returns the custom OpenAI page.
func (r *Runtime) OpenAIProviders() ([]OpenAIDraft, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	return ReadOpenAI(cfg), nil
}

// SaveOpenAI writes the custom OpenAI page.
func (r *Runtime) SaveOpenAI(drafts []OpenAIDraft) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyOpenAI(cfg, drafts, r.paths.Auth)
	})
}

// Start binds the saved config.
func (r *Runtime) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("服务已经在运行")
	}
	r.mu.Unlock()

	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return err
	}
	cfg.AuthDir = r.paths.Auth
	if err := CheckRunnable(cfg, r.paths.Auth); err != nil {
		return err
	}
	svc, err := cliproxy.NewBuilder().
		WithConfig(cfg).
		WithConfigPath(r.paths.Config).
		Build()
	if err != nil {
		return fmt.Errorf("服务无法创建: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	runErr := make(chan error, 1)
	go func() {
		defer close(done)
		err := svc.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			runErr <- err
			return
		}
		runErr <- nil
	}()

	listen := SavedListen(cfg)
	if err := waitUntilListening(ctx, listen, runErr); err != nil {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
		r.mu.Lock()
		r.running = false
		r.runError = err.Error()
		r.cancel = nil
		r.mu.Unlock()
		r.emit("server:status", mustStatus(r))
		return err
	}

	r.mu.Lock()
	r.running = true
	r.bound = listen
	r.runError = ""
	r.cancel = cancel
	r.done = done
	r.mu.Unlock()
	go r.watchExit(done, runErr)
	r.emit("server:status", mustStatus(r))
	return nil
}

// Stop cancels the embedded service.
func (r *Runtime) Stop() error {
	r.mu.Lock()
	cancel := r.cancel
	done := r.done
	running := r.running
	r.mu.Unlock()
	if !running || cancel == nil {
		return fmt.Errorf("服务没有在运行")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		return fmt.Errorf("停止服务超时")
	}
	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.done = nil
	r.runError = ""
	r.mu.Unlock()
	r.emit("server:status", mustStatus(r))
	return nil
}

// Restart stops the service and starts it with the saved config.
func (r *Runtime) Restart() error {
	if err := r.Stop(); err != nil {
		return err
	}
	if err := r.Start(); err != nil {
		return err
	}
	return nil
}

// Shutdown stops the service when the window closes.
func (r *Runtime) Shutdown() {
	r.CancelLogin()
	r.mu.Lock()
	running := r.running
	r.mu.Unlock()
	if running {
		_ = r.Stop()
	}
}

// Login opens the system browser for one upstream.
func (r *Runtime) Login(provider string) error {
	loginProvider, ok := loginProviders[provider]
	if !ok {
		return fmt.Errorf("这个上游不能用浏览器登录")
	}
	r.loginMu.Lock()
	if r.loginCancel != nil {
		r.loginMu.Unlock()
		return fmt.Errorf("已经有一个登录在进行")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	r.loginCancel = cancel
	r.loginMu.Unlock()

	release := func() {
		cancel()
		r.loginMu.Lock()
		r.loginCancel = nil
		r.loginMu.Unlock()
	}

	type outcome struct {
		record *coreauth.Auth
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		record, err := r.runLogin(ctx, loginProvider)
		done <- outcome{record: record, err: err}
	}()

	var result outcome
	select {
	case result = <-done:
	case <-ctx.Done():
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
	label := accountLabel(result.record)
	r.emit("login:done", Account{ID: result.record.ID, Provider: result.record.Provider, Label: label})
	return nil
}

// loginWithSDK runs the SDK browser login. Some authenticators ignore context
// cancellation, so the caller must not hold the login slot while it blocks.
func (r *Runtime) loginWithSDK(ctx context.Context, provider string) (*coreauth.Auth, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	cfg.AuthDir = r.paths.Auth
	store := auth.GetTokenStore()
	if setter, ok := store.(interface{ SetBaseDir(string) }); ok {
		setter.SetBaseDir(r.paths.Auth)
	}
	manager := auth.NewManager(store, newAuthenticators()...)
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

// Accounts lists browser logins for a sidebar page.
func (r *Runtime) Accounts(page string) ([]Account, error) {
	providers, ok := accountProviders[page]
	if !ok {
		return []Account{}, nil
	}
	store := auth.NewFileTokenStore()
	store.SetBaseDir(r.paths.Auth)
	records, err := store.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("读取账号失败: %w", err)
	}
	out := make([]Account, 0)
	for _, record := range records {
		if record == nil || !providers[record.Provider] {
			continue
		}
		out = append(out, Account{
			ID:       record.ID,
			Provider: record.Provider,
			Label:    accountLabel(record),
		})
	}
	return out, nil
}

// AccountUsage reads session-limit windows for the logins on one sidebar page.
// Providers without a vendor usage endpoint are omitted. A failed read stays on
// that account and does not fail the rest.
func (r *Runtime) AccountUsage(page string) ([]AccountUsage, error) {
	providers, ok := accountProviders[page]
	if !ok {
		return []AccountUsage{}, nil
	}
	store := auth.NewFileTokenStore()
	store.SetBaseDir(r.paths.Auth)
	records, err := store.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("读取账号失败: %w", err)
	}
	jobs := make([]*coreauth.Auth, 0)
	for _, record := range records {
		if record == nil || !providers[record.Provider] || !supportsSessionLimit(record.Provider) {
			continue
		}
		jobs = append(jobs, record)
	}
	out := make([]AccountUsage, len(jobs))
	var wg sync.WaitGroup
	for i, record := range jobs {
		wg.Add(1)
		go func(i int, record *coreauth.Auth) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), limitsTimeout)
			defer cancel()
			windows, err := sessionLimits(ctx, record.Provider, record.Metadata)
			if windows == nil {
				windows = []UsageWindow{}
			}
			item := AccountUsage{ID: record.ID, Windows: windows}
			if err != nil {
				item.Error = err.Error()
			}
			out[i] = item
		}(i, record)
	}
	wg.Wait()
	return out, nil
}

// DeleteAccount removes one browser-login file.
func (r *Runtime) DeleteAccount(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("没有指定账号")
	}
	store := auth.NewFileTokenStore()
	store.SetBaseDir(r.paths.Auth)
	if err := store.Delete(context.Background(), id); err != nil {
		return fmt.Errorf("删除账号失败: %w", err)
	}
	return nil
}

func (r *Runtime) edit(apply func(*config.Config) error) error {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return err
	}
	if err := apply(cfg); err != nil {
		return err
	}
	if err := atLeastOneClientKey(cfg.APIKeys); err != nil {
		return err
	}
	if err := saveConfig(r.paths.Config, cfg); err != nil {
		return err
	}
	r.emit("server:status", mustStatus(r))
	return nil
}

func (r *Runtime) watchExit(done chan struct{}, runErr chan error) {
	<-done
	var message string
	select {
	case err := <-runErr:
		if err != nil {
			message = err.Error()
		}
	default:
	}
	r.mu.Lock()
	r.running = false
	r.cancel = nil
	r.done = nil
	if message != "" {
		r.runError = message
	}
	r.mu.Unlock()
	r.emit("server:status", mustStatus(r))
}

func waitUntilListening(ctx context.Context, listen Listen, runErr chan error) error {
	address, _ := ClientURL(listen)
	target := strings.TrimPrefix(strings.TrimPrefix(address, "https://"), "http://")
	deadline := time.Now().Add(20 * time.Second)
	for {
		dialer := net.Dialer{Timeout: 200 * time.Millisecond}
		conn, err := dialer.DialContext(ctx, "tcp", target)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case runFailure := <-runErr:
			if runFailure == nil {
				return fmt.Errorf("服务启动后立即退出")
			}
			return fmt.Errorf("服务启动失败: %w", runFailure)
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("服务启动超时")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func mustStatus(r *Runtime) Status {
	status, err := r.Status()
	if err != nil {
		return Status{Error: err.Error()}
	}
	return status
}

var loginProviders = map[string]string{
	"codex":   "codex",
	"grok":    "xai",
	"claude":  "claude",
	"gemini":  "antigravity",
	"kimi":    "kimi",
	"kimi-ai": "kimi-ai",
	"devin":   "devin",
}

var accountProviders = map[string]map[string]bool{
	"codex":  {"codex": true},
	"grok":   {"xai": true},
	"claude": {"claude": true},
	"gemini": {"antigravity": true},
	"kimi":   {"kimi": true, "kimi-ai": true, "kimi.ai": true},
	"devin":  {"devin": true},
}

func newAuthenticators() []auth.Authenticator {
	return []auth.Authenticator{
		auth.NewCodexAuthenticator(),
		auth.NewClaudeAuthenticator(),
		auth.NewAntigravityAuthenticator(),
		auth.NewKimiAuthenticator(),
		auth.NewKimiAIAuthenticator(),
		auth.NewXAIAuthenticator(),
		auth.NewDevinAuthenticator(),
	}
}

func accountLabel(record *coreauth.Auth) string {
	if record == nil {
		return ""
	}
	if record.Metadata != nil {
		if email, ok := record.Metadata["email"].(string); ok && strings.TrimSpace(email) != "" {
			return email
		}
	}
	if strings.TrimSpace(record.Label) != "" {
		return record.Label
	}
	return record.ID
}
