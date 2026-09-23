package desktop

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestRuntimeEditPreservesConcurrentPages(t *testing.T) {
	root := t.TempDir()
	authDir := filepath.Join(root, "auths")
	if err := os.Mkdir(authDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(configPath, []byte(defaultConfig(authDir, "sk-test")), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{paths: Paths{Config: configPath, Auth: authDir}, emit: func(string, any) {}}
	for range 20 {
		if err := os.WriteFile(configPath, []byte(defaultConfig(authDir, "sk-test")), 0o600); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := r.edit(func(cfg *config.Config) error { cfg.ProxyURL = "http://proxy.test"; return nil }); err != nil {
				t.Errorf("proxy edit: %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := r.edit(func(cfg *config.Config) error { cfg.Routing.Strategy = "fill-first"; return nil }); err != nil {
				t.Errorf("routing edit: %v", err)
			}
		}()
		wg.Wait()
		cfg, err := loadConfig(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.ProxyURL != "http://proxy.test" || cfg.Routing.Strategy != "fill-first" || len(cfg.APIKeys) != 1 {
			t.Fatalf("lost an edit: proxy=%q strategy=%q keys=%d", cfg.ProxyURL, cfg.Routing.Strategy, len(cfg.APIKeys))
		}
	}
}
