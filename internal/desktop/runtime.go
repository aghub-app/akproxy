package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// Runtime owns the saved config and the embedded proxy process.
type Runtime struct {
	paths Paths

	controlMu sync.Mutex
	editMu    sync.Mutex
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	running   bool
	coreAuth  *coreauth.Manager
	bound     Listen
	runError  string

	loginMu     sync.Mutex
	loginCancel context.CancelFunc
	runLogin    func(ctx context.Context, provider string, target *reauthTarget) (*coreauth.Auth, error)

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
	r.loginMu.Lock()
	loginActive := r.loginCancel != nil
	r.loginMu.Unlock()
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
		LoginActive:     loginActive,
	}, nil
}

// Start binds the saved config.
func (r *Runtime) Start() error {
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	return r.start()
}

func (r *Runtime) start() error {
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
	manager := coreauth.NewManager(r.accountStore(), authSelector(cfg), nil)
	svc, err := cliproxy.NewBuilder().
		WithConfig(cfg).
		WithConfigPath(r.paths.Config).
		WithCoreAuthManager(manager).
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
		r.coreAuth = nil
		r.runError = err.Error()
		r.cancel = nil
		r.mu.Unlock()
		r.emit("server:status", mustStatus(r))
		return err
	}

	r.mu.Lock()
	r.running = true
	r.coreAuth = manager
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
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	return r.stop()
}

func (r *Runtime) stop() error {
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
	r.coreAuth = nil
	r.cancel = nil
	r.done = nil
	r.runError = ""
	r.mu.Unlock()
	r.emit("server:status", mustStatus(r))
	return nil
}

// Restart stops the service and starts it with the saved config.
func (r *Runtime) Restart() error {
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	if err := r.stop(); err != nil {
		return err
	}
	if err := r.start(); err != nil {
		return err
	}
	return nil
}

// Shutdown stops the service when the window closes.
func (r *Runtime) Shutdown() {
	r.CancelLogin()
	r.controlMu.Lock()
	defer r.controlMu.Unlock()
	r.mu.Lock()
	running := r.running
	r.mu.Unlock()
	if running {
		_ = r.stop()
	}
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
	if r.done != done {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.coreAuth = nil
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
