package desktop

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// Some SDK authenticators ignore context cancellation and only return after
// their own callback timeout. Cancelling a login must free the slot at once,
// so a follow-up login can start instead of hitting the busy-slot error.
func TestCancelLoginFreesSlotWhileSDKStillBlocking(t *testing.T) {
	r, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	blocked := make(chan struct{})
	r.runLogin = func(ctx context.Context, provider string, _ *reauthTarget) (*coreauth.Auth, error) {
		close(blocked)
		<-ctx.Done()
		time.Sleep(50 * time.Millisecond)
		return nil, context.Canceled
	}

	loginDone := make(chan error, 1)
	go func() {
		loginDone <- r.Login("claude")
	}()

	select {
	case <-blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("runLogin was never called")
	}

	r.CancelLogin()
	cancelled := make(chan error, 1)
	go func() { cancelled <- r.Login("codex") }()

	select {
	case err := <-cancelled:
		if err != nil && err.Error() != "已经有一个登录在进行" {
			t.Fatalf("second login returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second login still blocked after cancel: slot not freed")
	}

	select {
	case err := <-loginDone:
		if err == nil || err.Error() != "登录已取消" {
			t.Fatalf("first login error = %v, want 登录已取消", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first login did not return after cancel")
	}
}

// Cancelling a codex login must send a synthetic error callback to the
// authenticator's local callback server so it stops and frees the port.
func TestCancelLoginUnblocksCallbackServer(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.URL.Query().Get("error") == "" {
			t.Errorf("callback query missing error param: %s", r.URL.RawQuery)
		}
	}))
	defer server.Close()

	r, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	_, portStr, _ := net.SplitHostPort(server.Listener.Addr().String())
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}
	previous := callbackLoginPorts["codex"]
	callbackLoginPorts["codex"] = struct {
		port int
		path string
	}{port: port, path: "/auth/callback"}
	defer func() { callbackLoginPorts["codex"] = previous }()

	blocked := make(chan struct{})
	r.runLogin = func(ctx context.Context, provider string, _ *reauthTarget) (*coreauth.Auth, error) {
		close(blocked)
		<-ctx.Done()
		time.Sleep(300 * time.Millisecond)
		return nil, context.Canceled
	}

	go func() { _ = r.Login("codex") }()
	select {
	case <-blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("runLogin was never called")
	}

	r.CancelLogin()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if hits.Load() > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("synthetic error callback was never delivered")
}

func TestLoginMapsSDKErrorAndEmitsDone(t *testing.T) {
	r, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	r.runLogin = func(ctx context.Context, provider string, _ *reauthTarget) (*coreauth.Auth, error) {
		if provider != "claude" {
			t.Fatalf("provider = %q, want claude", provider)
		}
		return &coreauth.Auth{ID: "a@x.com", Provider: "claude"}, nil
	}
	events := make(chan string, 8)
	r.emit = func(event string, _ any) { events <- event }

	if err := r.Login("claude"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	var sawLoginDone bool
	for i := 0; i < 8; i++ {
		select {
		case event := <-events:
			if event == "login:done" {
				sawLoginDone = true
			}
		case <-time.After(time.Second):
			break
		}
	}
	if !sawLoginDone {
		t.Fatal("login:done not emitted")
	}

	r.runLogin = func(ctx context.Context, provider string, _ *reauthTarget) (*coreauth.Auth, error) {
		return nil, errors.New("boom")
	}
	err = r.Login("claude")
	if err == nil || err.Error() != "登录失败: boom" {
		t.Fatalf("error = %v, want 登录失败: boom", err)
	}
}
