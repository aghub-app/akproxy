package desktop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestAccountsReportsSDKStatusOnlyWhileRunning(t *testing.T) {
	dir := t.TempDir()
	id := "codex-saved.json"
	if err := os.WriteFile(filepath.Join(dir, id), []byte(`{"type":"codex","email":"a@example.test"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	manager := coreauth.NewManager(nil, nil, nil)
	if _, err := manager.Register(context.Background(), &coreauth.Auth{
		ID: id, Provider: "codex", Status: coreauth.StatusError, StatusMessage: "invalid_grant",
	}); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{paths: Paths{Auth: dir}, running: true, coreAuth: manager}
	accounts, err := r.Accounts("codex")
	if err != nil || len(accounts) != 1 || !accounts[0].NeedsReauthorization {
		t.Fatalf("running accounts = %+v, %v", accounts, err)
	}
	r.running = false
	accounts, err = r.Accounts("codex")
	if err != nil || len(accounts) != 1 || accounts[0].NeedsReauthorization {
		t.Fatalf("stopped accounts = %+v, %v", accounts, err)
	}
}

type recordingAuthStore struct{ saved *coreauth.Auth }

func (s *recordingAuthStore) List(context.Context) ([]*coreauth.Auth, error) { return nil, nil }
func (s *recordingAuthStore) Delete(context.Context, string) error           { return nil }
func (s *recordingAuthStore) Save(_ context.Context, record *coreauth.Auth) (string, error) {
	s.saved = record
	return record.ID, nil
}

func TestAccountReauthorizationUsesOnlySDKAuthErrors(t *testing.T) {
	for _, tc := range []struct {
		message string
		want    bool
	}{
		{"unauthorized", true},
		{"invalid_grant", true},
		{"token expired", false},
		{"quota exhausted", false},
		{"transient upstream error", false},
	} {
		record := &coreauth.Auth{Status: coreauth.StatusError, StatusMessage: tc.message}
		if got := needsReauthorization(record); got != tc.want {
			t.Errorf("message %q: got %v, want %v", tc.message, got, tc.want)
		}
	}
	if needsReauthorization(&coreauth.Auth{Status: coreauth.StatusActive, StatusMessage: "unauthorized"}) {
		t.Fatal("active account must not need reauthorization")
	}
}

func TestReauthorizationStoreRejectsDifferentAccountBeforeSave(t *testing.T) {
	underlying := &recordingAuthStore{}
	store := sameAccountStore{Store: underlying, target: reauthTarget{ID: "selected.json", Provider: "codex"}}
	for _, record := range []*coreauth.Auth{
		{ID: "other.json", Provider: "codex"},
		{ID: "selected.json", Provider: "claude"},
	} {
		if _, err := store.Save(context.Background(), record); err == nil {
			t.Fatalf("Save(%s, %s) accepted another account", record.ID, record.Provider)
		}
		if underlying.saved != nil {
			t.Fatal("rejected account reached disk store")
		}
	}
	matching := &coreauth.Auth{ID: "selected.json", Provider: "codex"}
	if _, err := store.Save(context.Background(), matching); err != nil || underlying.saved != matching {
		t.Fatalf("matching account was not saved: %v", err)
	}
	if !sameAuthProvider("kimi-ai", "kimi.ai") || sameAuthProvider("kimi", "kimi.ai") {
		t.Fatal("Kimi provider aliases were matched incorrectly")
	}
}

func TestReauthorizationStoreKeepsSelectedFile(t *testing.T) {
	dir := t.TempDir()
	id := "selected.json"
	path := filepath.Join(dir, id)
	if err := os.WriteFile(path, []byte(`{"type":"codex"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	underlying := &recordingAuthStore{}
	store := sameAccountStore{Store: underlying, target: reauthTarget{ID: id, Provider: "codex"}, authDir: dir}
	record := &coreauth.Auth{ID: id, Provider: "codex", FileName: "wrong.json", Attributes: map[string]string{coreauth.AttributePath: "wrong.json"}}
	if _, err := store.Save(context.Background(), record); err != nil {
		t.Fatal(err)
	}
	if record.FileName != id || record.Attributes[coreauth.AttributePath] != "" {
		t.Fatal("selected file path was not enforced")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	underlying.saved = nil
	if _, err := store.Save(context.Background(), record); err == nil || underlying.saved != nil {
		t.Fatal("deleted target was recreated")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.Save(ctx, record); err == nil || underlying.saved != nil {
		t.Fatal("canceled login reached disk store")
	}
}

func TestInjectedAuthManagerKeepsConfiguredRouting(t *testing.T) {
	for _, tc := range []struct {
		strategy string
		want     any
	}{
		{"round-robin", &coreauth.RoundRobinSelector{}},
		{"weighted-round-robin", &coreauth.WeightedRoundRobinSelector{}},
		{"fill-first", &coreauth.FillFirstSelector{}},
	} {
		cfg := &config.Config{}
		cfg.Routing.Strategy = tc.strategy
		got := authSelector(cfg)
		switch tc.want.(type) {
		case *coreauth.RoundRobinSelector:
			if _, ok := got.(*coreauth.RoundRobinSelector); !ok {
				t.Errorf("%s: %T", tc.strategy, got)
			}
		case *coreauth.WeightedRoundRobinSelector:
			if _, ok := got.(*coreauth.WeightedRoundRobinSelector); !ok {
				t.Errorf("%s: %T", tc.strategy, got)
			}
		case *coreauth.FillFirstSelector:
			if _, ok := got.(*coreauth.FillFirstSelector); !ok {
				t.Errorf("%s: %T", tc.strategy, got)
			}
		}
	}
	cfg := &config.Config{}
	cfg.Routing.SessionAffinity = true
	if _, ok := authSelector(cfg).(*coreauth.SessionAffinitySelector); !ok {
		t.Fatal("session affinity was dropped")
	}
}
