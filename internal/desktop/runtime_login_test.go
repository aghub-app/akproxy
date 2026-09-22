package desktop

import (
	"context"
	"errors"
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
	r.runLogin = func(ctx context.Context, provider string) (*coreauth.Auth, error) {
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

func TestLoginMapsSDKErrorAndEmitsDone(t *testing.T) {
	r, err := NewRuntime(nil)
	if err != nil {
		t.Fatalf("NewRuntime: %v", err)
	}
	r.runLogin = func(ctx context.Context, provider string) (*coreauth.Auth, error) {
		if provider != "claude" {
			t.Fatalf("provider = %q, want claude", provider)
		}
		return &coreauth.Auth{ID: "a@x.com", Provider: "claude"}, nil
	}
	events := make(chan string, 1)
	r.emit = func(event string, _ any) { events <- event }

	if err := r.Login("claude"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	select {
	case event := <-events:
		if event != "login:done" {
			t.Fatalf("event = %q, want login:done", event)
		}
	case <-time.After(time.Second):
		t.Fatal("login:done not emitted")
	}

	r.runLogin = func(ctx context.Context, provider string) (*coreauth.Auth, error) {
		return nil, errors.New("boom")
	}
	err = r.Login("claude")
	if err == nil || err.Error() != "登录失败: boom" {
		t.Fatalf("error = %v, want 登录失败: boom", err)
	}
}
