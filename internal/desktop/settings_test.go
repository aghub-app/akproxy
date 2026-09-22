package desktop

import (
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

func TestApplyServiceKeepsUnrelatedFieldsAndRejectsEmptyKeys(t *testing.T) {
	cfg := mustParse(t, `
payload:
  default:
    - models:
        - name: "gemini-2.5-pro"
      params:
        temperature: 0.2
gemini-api-key:
  - api-key: gem
    base-url: https://example.test
    models:
      - name: gemini-2.5-flash
        alias: flash
routing:
  session-affinity: true
tls:
  enable: true
  cert: /tmp/cert.pem
  key: /tmp/key.pem
remote-management:
  allow-remote: true
  secret-key: keep-me
  disable-control-panel: true
`)

	secret := cfg.RemoteManagement.SecretKey
	allowRemote := cfg.RemoteManagement.AllowRemote
	disablePanel := cfg.RemoteManagement.DisableControlPanel
	err := ApplyService(cfg, ServiceSettings{
		ListenMode:      listenLocal,
		Port:            9000,
		ClientAPIKeys:   []string{"sk-a"},
		RoutingStrategy: strategyFillFirst,
	}, "/tmp/akproxy-auths")
	if err != nil {
		t.Fatalf("save service: %v", err)
	}
	if cfg.Port != 9000 || cfg.Host != "127.0.0.1" || cfg.Routing.Strategy != strategyFillFirst {
		t.Fatalf("service fields were not applied: %+v", cfg)
	}
	if !cfg.Routing.SessionAffinity {
		t.Fatal("session affinity was cleared")
	}
	if len(cfg.Payload.Default) != 1 {
		t.Fatal("payload was cleared")
	}
	if len(cfg.GeminiKey) != 1 || cfg.GeminiKey[0].Models[0].Name != "gemini-2.5-flash" {
		t.Fatal("gemini key was changed by the service page")
	}
	if cfg.AuthDir != "/tmp/akproxy-auths" {
		t.Fatalf("auth dir = %q", cfg.AuthDir)
	}
	if !cfg.TLS.Enable || cfg.TLS.Cert != "/tmp/cert.pem" || cfg.TLS.Key != "/tmp/key.pem" {
		t.Fatalf("tls was changed by the service page: %+v", cfg.TLS)
	}
	if cfg.RemoteManagement.SecretKey != secret || cfg.RemoteManagement.AllowRemote != allowRemote || cfg.RemoteManagement.DisableControlPanel != disablePanel {
		t.Fatalf("remote management was changed: %+v", cfg.RemoteManagement)
	}

	err = ApplyService(cfg, ServiceSettings{
		ListenMode:      listenLocal,
		Port:            9000,
		RoutingStrategy: strategyRoundRobin,
	}, "/tmp/akproxy-auths")
	if err == nil || !strings.Contains(err.Error(), "客户端密钥") {
		t.Fatalf("empty client keys err = %v", err)
	}
}

func TestApplyKeysPreservesUneditedFields(t *testing.T) {
	cfg := mustParse(t, `
claude-api-key:
  - api-key: keep-me
codex-api-key:
  - api-key: codex-key
    base-url: https://old.example
    models:
      - name: gpt-5
        alias: g5
`)

	err := ApplyKeys(cfg, "codex", []KeyDraft{{APIKey: "codex-key", BaseURL: "https://new.example", Prefix: "team"}}, "/auth")
	if err != nil {
		t.Fatalf("apply keys: %v", err)
	}
	if cfg.CodexKey[0].BaseURL != "https://new.example" || cfg.CodexKey[0].Prefix != "team" {
		t.Fatalf("draft not applied: %+v", cfg.CodexKey[0])
	}
	if len(cfg.CodexKey[0].Models) != 1 || cfg.CodexKey[0].Models[0].Alias != "g5" {
		t.Fatal("model mapping was dropped")
	}
	if len(cfg.ClaudeKey) != 1 {
		t.Fatal("claude keys were touched")
	}

	if err := ApplyKeys(cfg, "codex", nil, "/auth"); err != nil {
		t.Fatalf("clear keys: %v", err)
	}
	if len(cfg.CodexKey) != 0 {
		t.Fatal("cleared codex keys remained")
	}
}

func TestKimiAndCustomOpenAIStayApart(t *testing.T) {
	cfg := &config.Config{}
	cfg.OpenAICompatibility = []config.OpenAICompatibility{{
		Name:          "openrouter",
		BaseURL:       "https://openrouter.ai/api/v1",
		APIKeyEntries: []config.OpenAICompatibilityAPIKey{{APIKey: "or-key", ProxyURL: "socks5://127.0.0.1:1080"}},
		Models:        []config.OpenAICompatibilityModel{{Name: "gpt-4.1", Alias: "gpt", ForceMapping: true}},
	}}

	kimi := []OpenAIDraft{{
		Name:    kimiName,
		BaseURL: "https://api.kimi.com/coding",
		APIKeys: []string{"kimi-key"},
		Models:  []ModelDraft{{Name: "kimi-k2", Alias: ""}},
	}}
	if err := ApplyKimi(cfg, kimi, "/auth"); err != nil {
		t.Fatalf("apply kimi: %v", err)
	}
	if len(cfg.OpenAICompatibility) != 2 {
		t.Fatalf("entries = %+v", cfg.OpenAICompatibility)
	}
	if cfg.OpenAICompatibility[0].Name != "openrouter" || cfg.OpenAICompatibility[0].APIKeyEntries[0].ProxyURL == "" {
		t.Fatal("custom provider was rewritten")
	}
	if cfg.OpenAICompatibility[1].Models[0].Alias != "kimi-k2" {
		t.Fatal("empty alias was not defaulted")
	}

	if err := ApplyKimi(cfg, []OpenAIDraft{{
		Name: "openrouter", BaseURL: "https://example.test", APIKeys: []string{"x"}, Models: []ModelDraft{{Name: "m"}},
	}}, "/auth"); err == nil {
		t.Fatal("kimi save accepted a custom name")
	}
	if cfg.OpenAICompatibility[0].Name != "openrouter" || cfg.OpenAICompatibility[0].APIKeyEntries[0].APIKey != "or-key" {
		t.Fatal("rejected kimi save rewrote the custom entry")
	}
	if len(ReadKimi(cfg)) != 1 {
		t.Fatal("kimi entry missing")
	}
}

func TestControlAction(t *testing.T) {
	bound := Listen{Host: "127.0.0.1", Port: 8317}
	if ControlAction(false, bound, bound) != "start" {
		t.Fatal("stopped")
	}
	if ControlAction(true, bound, bound) != "stop" {
		t.Fatal("running unchanged")
	}
	saved := bound
	saved.Port = 9000
	if ControlAction(true, bound, saved) != "restart" {
		t.Fatal("port change")
	}
	address, all := ClientURL(Listen{Host: "", Port: 8317})
	if address != "http://127.0.0.1:8317" || !all {
		t.Fatalf("url = %s all=%v", address, all)
	}
}

func mustParse(t *testing.T, body string) *config.Config {
	t.Helper()
	cfg, err := config.ParseConfigBytes([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}
