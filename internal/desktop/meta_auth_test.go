package desktop

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestMetaDeviceLoginAndQuota(t *testing.T) {
	var polls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/device":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"device_code": "device-secret", "user_code": "ABCD-1234",
				"verification_uri_complete": "https://auth.meta.com/device?user_code=ABCD-1234",
				"interval":                  1,
			})
		case "/token":
			if polls.Add(1) == 1 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "authorization_pending"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "dca-secret", "token_type": "Bearer", "expires_in": 3600})
		case "/key":
			if r.Header.Get("Authorization") != "Bearer dca-secret" {
				t.Errorf("key request missed DCA authorization")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"api_key": "llm-secret", "user_email": "person@example.com", "subs_tier_name": "Muse Pro",
				"subs_usage": map[string]any{
					"window": map[string]any{"used_percent": 25, "window_duration_mins": 300, "resets_at": "2026-09-25T12:00:00Z"},
					"weekly": map[string]any{"used_percent": 50, "resets_at": "2026-09-30T12:00:00Z"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	var code MetaLoginCode
	provider := metaAuthenticator{
		client: server.Client(), deviceURL: server.URL + "/device", tokenURL: server.URL + "/token", keyURL: server.URL + "/key",
		emit: func(name string, data any) {
			if name == "login:meta-code" {
				code = data.(MetaLoginCode)
			}
		},
	}
	dir := t.TempDir()
	store := auth.NewFileTokenStore()
	store.SetBaseDir(dir)
	manager := auth.NewManager(store, provider)
	record, _, err := manager.Login(context.Background(), "meta", &config.Config{AuthDir: dir}, &auth.LoginOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if code.Code != "ABCD-1234" || code.URL == "" {
		t.Fatalf("missing device dialog event: %+v", code)
	}
	if record.Provider != "meta" || record.Metadata["dca_token"] != "dca-secret" {
		t.Fatalf("wrong Meta record: %+v", record)
	}
	if _, err := os.Stat(filepath.Join(dir, record.ID)); err != nil {
		t.Fatal(err)
	}
	accounts, err := store.List(context.Background())
	if err != nil || len(accounts) != 1 || accounts[0].Provider != "meta" {
		t.Fatalf("saved accounts: %v, %v", accounts, err)
	}
	usage, err := fetchMetaLimitsAt(context.Background(), accounts[0].Metadata, server.URL+"/key", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if usage.Plan != "Muse Pro" || len(usage.Windows) != 2 || usage.Windows[0].Used != 25 || usage.Windows[0].DurationSeconds != 18000 || usage.Windows[1].Used != 50 {
		t.Fatalf("wrong Meta quota: %+v", usage)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	configPath := filepath.Join(dir, "config.yaml")
	configBody := strings.Replace(defaultConfig(dir, "sk-test"), "port: 8317", "port: "+strconv.Itoa(port), 1)
	if err := os.WriteFile(configPath, []byte(configBody), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{paths: Paths{Config: configPath, Auth: dir}, emit: func(string, any) {}}
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Stop() })
	var models []string
	found := false
	for attempt := 0; attempt < 50 && !found; attempt++ {
		models, err = runtime.HomeModels()
		if err != nil {
			t.Fatal(err)
		}
		for _, model := range models {
			if model == "muse-spark-1.3" {
				found = true
			}
		}
		if !found {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if !found {
		t.Fatalf("Meta account exposed no Muse model: %v", models)
	}
}

func TestMetaCancelBeforeSave(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device", "user_code": "CODE", "verification_uri": "https://auth.meta.com/device"})
	}))
	defer server.Close()
	dir := t.TempDir()
	store := auth.NewFileTokenStore()
	store.SetBaseDir(dir)
	manager := auth.NewManager(store, metaAuthenticator{
		client: server.Client(), deviceURL: server.URL, tokenURL: server.URL,
		emit: func(string, any) { cancel() },
	})
	if _, _, err := manager.Login(ctx, "meta", &config.Config{AuthDir: dir}, nil); err == nil {
		t.Fatal("canceled login succeeded")
	}
	accounts, err := store.List(context.Background())
	if err != nil || len(accounts) != 0 {
		t.Fatalf("canceled login saved accounts: %v, %v", accounts, err)
	}
}

func TestMetaFailureNeverSaves(t *testing.T) {
	for _, failure := range []string{"denied", "mint"} {
		t.Run(failure, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/device":
					_ = json.NewEncoder(w).Encode(map[string]any{"device_code": "device", "user_code": "CODE", "verification_uri": "https://auth.meta.com/device", "interval": 1})
				case "/token":
					if failure == "denied" {
						w.WriteHeader(http.StatusBadRequest)
						_ = json.NewEncoder(w).Encode(map[string]string{"error": "access_denied"})
					} else {
						_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "dca-secret"})
					}
				case "/key":
					w.WriteHeader(http.StatusForbidden)
				}
			}))
			defer server.Close()
			dir := t.TempDir()
			store := auth.NewFileTokenStore()
			store.SetBaseDir(dir)
			manager := auth.NewManager(store, metaAuthenticator{client: server.Client(), deviceURL: server.URL + "/device", tokenURL: server.URL + "/token", keyURL: server.URL + "/key"})
			if _, _, err := manager.Login(context.Background(), "meta", &config.Config{AuthDir: dir}, nil); err == nil {
				t.Fatal("failed Meta login succeeded")
			}
			accounts, err := store.List(context.Background())
			if err != nil || len(accounts) != 0 {
				t.Fatalf("failed Meta login saved accounts: %v, %v", accounts, err)
			}
		})
	}
}

func TestMetaQuotaOmitsMissingWindows(t *testing.T) {
	used := 0.0
	window, ok := parseMetaQuotaWindow("weekly", &metaQuotaWindow{UsedPercent: &used})
	if !ok || window.Used != 0 {
		t.Fatalf("explicit zero omitted: %+v", window)
	}
	if _, ok := parseMetaQuotaWindow("rolling", &metaQuotaWindow{}); ok {
		t.Fatal("missing percent displayed as zero")
	}
}
