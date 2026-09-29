//go:build e2e

package e2e

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const model = "gpt-5.5"

func TestCLIVersion(t *testing.T) {
	sb := newSandbox(t)
	got := run(t, sb, nil, cliBin, "--version")
	if got.code != 0 || got.out != "dev" {
		t.Fatalf("akproxy --version = %d %q", got.code, got.out)
	}
}

func TestCLIRefusesWithoutApp(t *testing.T) {
	sb := newSandbox(t)
	sb.installAgent(t, sb.agents, "claude")
	for _, args := range [][]string{{"claude"}, {"model"}} {
		got := run(t, sb, nil, cliBin, args...)
		if got.code == 0 || !strings.Contains(got.out, "请先打开 akproxy") {
			t.Fatalf("akproxy %v without app = %d %q", args, got.code, got.out)
		}
	}
	if _, err := os.Stat(sb.paths.Config); !os.IsNotExist(err) {
		t.Fatalf("the CLI must not create the app config, err=%v", err)
	}
}

func TestCLIRequiresModel(t *testing.T) {
	sb := newSandbox(t)
	sb.openApp(t, 8317)
	for _, name := range []string{"claude", "codex", "opencode", "pi"} {
		sb.installAgent(t, sb.agents, name)
		out := filepath.Join(t.TempDir(), "rec.json")
		got := run(t, sb, map[string]string{"AKPROXY_FAKE_OUT": out}, cliBin, name)
		if got.code == 0 || !strings.Contains(got.out, "akproxy model") {
			t.Fatalf("akproxy %s without model = %d %q", name, got.code, got.out)
		}
		if _, err := os.Stat(out); err == nil {
			t.Fatalf("%s must not start without a model", name)
		}
	}
}

func TestCLIMissingAgent(t *testing.T) {
	sb := newSandbox(t)
	sb.openApp(t, 8317)
	sb.chooseModel(t, model)
	got := run(t, sb, nil, cliBin, "claude")
	if got.code == 0 || !strings.Contains(got.out, "找不到 claude") {
		t.Fatalf("akproxy claude without claude = %d %q", got.code, got.out)
	}
}

func TestCLILaunchesAgents(t *testing.T) {
	sb := newSandbox(t)
	address, key := sb.openApp(t, 8317)
	sb.chooseModel(t, model)
	extra := []string{"--print", "hello world", "--flag=1"}

	cases := map[string]func(t *testing.T, rec agentRecord){
		"claude": func(t *testing.T, rec agentRecord) {
			want := append([]string{"--model", model}, extra...)
			if !slices.Equal(rec.Args, want) {
				t.Fatalf("args = %q, want %q", rec.Args, want)
			}
			for name, value := range map[string]string{
				"ANTHROPIC_BASE_URL":   address,
				"ANTHROPIC_AUTH_TOKEN": key,
				"ANTHROPIC_MODEL":      model,
			} {
				if rec.Env[name] != value {
					t.Fatalf("%s = %q, want %q", name, rec.Env[name], value)
				}
			}
			if _, ok := rec.Env["ANTHROPIC_API_KEY"]; ok {
				t.Fatal("ANTHROPIC_API_KEY from the parent shell must be cleared")
			}
		},
		"codex": func(t *testing.T, rec agentRecord) {
			if !slices.Equal(rec.Args[len(rec.Args)-len(extra):], extra) {
				t.Fatalf("extra args must come last: %q", rec.Args)
			}
			joined := strings.Join(rec.Args, " ")
			for _, part := range []string{"model_provider=akproxy", "model_providers.akproxy.base_url=" + address + "/v1", "model=" + model} {
				if !strings.Contains(joined, part) {
					t.Fatalf("args %q missing %q", rec.Args, part)
				}
			}
			if strings.Contains(joined, key) {
				t.Fatal("the client key must not appear in codex arguments")
			}
			if rec.Env["AKPROXY_CLI_KEY"] != key {
				t.Fatalf("AKPROXY_CLI_KEY = %q", rec.Env["AKPROXY_CLI_KEY"])
			}
		},
		"opencode": func(t *testing.T, rec agentRecord) {
			if !slices.Equal(rec.Args, extra) {
				t.Fatalf("args = %q", rec.Args)
			}
			var cfg struct {
				Model    string `json:"model"`
				Provider map[string]struct {
					Options struct {
						BaseURL string `json:"baseURL"`
						APIKey  string `json:"apiKey"`
					} `json:"options"`
				} `json:"provider"`
			}
			if err := json.Unmarshal([]byte(rec.Env["OPENCODE_CONFIG_CONTENT"]), &cfg); err != nil {
				t.Fatalf("OPENCODE_CONFIG_CONTENT: %v", err)
			}
			opts := cfg.Provider["akproxy"].Options
			if cfg.Model != "akproxy/"+model || opts.BaseURL != address+"/v1" || opts.APIKey != key {
				t.Fatalf("opencode config = %+v", cfg)
			}
		},
		"pi": func(t *testing.T, rec agentRecord) {
			if !slices.Equal(rec.Args, extra) {
				t.Fatalf("args = %q", rec.Args)
			}
			if rec.Env["AKPROXY_CLI_KEY"] != key {
				t.Fatalf("AKPROXY_CLI_KEY = %q", rec.Env["AKPROXY_CLI_KEY"])
			}
			models := rec.Files["models.json"]
			if !strings.Contains(models, address+"/v1") || !strings.Contains(models, model) || strings.Contains(models, key) {
				t.Fatalf("pi models.json = %s", models)
			}
			if !strings.Contains(rec.Files["settings.json"], model) {
				t.Fatalf("pi settings.json = %s", rec.Files["settings.json"])
			}
			if _, err := os.Stat(rec.Env["PI_CODING_AGENT_DIR"]); !os.IsNotExist(err) {
				t.Fatalf("pi config dir should be removed after exit, err=%v", err)
			}
		},
	}
	for name, check := range cases {
		t.Run(name, func(t *testing.T) {
			sb.installAgent(t, sb.agents, name)
			out := filepath.Join(t.TempDir(), "rec.json")
			env := map[string]string{"AKPROXY_FAKE_OUT": out, "ANTHROPIC_API_KEY": "sk-user-own"}
			got := run(t, sb, env, cliBin, append([]string{name}, extra...)...)
			if got.code != 0 {
				t.Fatalf("akproxy %s = %d %q", name, got.code, got.out)
			}
			rec := readRecord(t, out)
			if rec.Name != name {
				t.Fatalf("launched %q, want %q", rec.Name, name)
			}
			check(t, rec)
		})
	}

	// The CLIs' own config locations stay untouched.
	for _, dir := range []string{".claude", ".codex", filepath.Join(".config", "opencode"), ".pi"} {
		if _, err := os.Stat(filepath.Join(sb.home, dir)); !os.IsNotExist(err) {
			t.Fatalf("%s should not be created, err=%v", dir, err)
		}
	}
}

func TestCLIPassesAgentExitCode(t *testing.T) {
	sb := newSandbox(t)
	sb.openApp(t, 8317)
	sb.chooseModel(t, model)
	sb.installAgent(t, sb.agents, "claude")
	got := run(t, sb, map[string]string{"AKPROXY_FAKE_EXIT": "7"}, cliBin, "claude")
	if got.code != 7 {
		t.Fatalf("exit = %d %q, want 7", got.code, got.out)
	}
}

func modelServer(t *testing.T, key string, ids ...string) int {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer "+key {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		data := []map[string]string{}
		for _, id := range ids {
			data = append(data, map[string]string{"id": id})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(server.Close)
	_, port, _ := net.SplitHostPort(server.Listener.Addr().String())
	n, _ := strconv.Atoi(port)
	return n
}

func closedPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

// A failed `akproxy model` keeps the previous choice.
func TestCLIModelFailuresKeepSelection(t *testing.T) {
	cases := map[string]struct {
		models []string
		down   bool
		want   string
	}{
		"service down": {down: true, want: "无法读取模型列表"},
		"no models":    {want: "暂无可选模型"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sb := newSandbox(t)
			port := closedPort(t)
			_, key := sb.openApp(t, port)
			if !tc.down {
				// The key is only known after the config exists; repoint the port.
				port = modelServer(t, key, tc.models...)
				sb.setPort(t, port)
			}
			sb.chooseModel(t, "kept")
			got := run(t, sb, nil, cliBin, "model")
			if got.code == 0 || !strings.Contains(got.out, tc.want) {
				t.Fatalf("akproxy model = %d %q", got.code, got.out)
			}
			body, _ := os.ReadFile(sb.cliJSON())
			if !strings.Contains(string(body), `"kept"`) {
				t.Fatalf("cli.json changed: %s", body)
			}
		})
	}
}
