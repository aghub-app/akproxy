package cli

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"akproxy/internal/desktop"
)

func TestVersionFlag(t *testing.T) {
	previous := Version
	Version = "9.9.9"
	t.Cleanup(func() { Version = previous })
	var out bytes.Buffer
	paths := desktop.Paths{Root: t.TempDir()}
	if err := Execute(&out, paths, []string{"--version"}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "9.9.9\n" {
		t.Fatalf("version = %q", out.String())
	}
}

func TestCommandSurface(t *testing.T) {
	paths := desktop.Paths{Root: t.TempDir(), Config: filepath.Join(t.TempDir(), "missing.yaml")}
	var out bytes.Buffer
	if err := Execute(&out, paths, nil); err != nil || !strings.Contains(out.String(), "model") || !strings.Contains(out.String(), "claude") || !strings.Contains(out.String(), "codex") || !strings.Contains(out.String(), "opencode") || !strings.Contains(out.String(), "pi") {
		t.Fatalf("root help: %v %q", err, out.String())
	}
	out.Reset()
	if err := Execute(&out, paths, []string{"model", "extra"}); err == nil {
		t.Fatal("model should reject arguments")
	}
	if err := Execute(&out, paths, []string{"grok"}); err == nil {
		t.Fatal("unknown command should fail")
	}
	if err := Execute(&out, paths, []string{"opencode", "--resume", "abc"}); err == nil || !strings.Contains(err.Error(), "请先打开 akproxy") {
		t.Fatalf("claude flags should pass through: %v", err)
	}
}

func TestReadModelID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cli.json")
	if _, err := ReadModelID(path); err == nil || !strings.Contains(err.Error(), "akproxy model") {
		t.Fatalf("missing file: %v", err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModelID(path); err == nil {
		t.Fatal("broken json should fail")
	}
	if err := os.WriteFile(path, []byte(`{"model":{"id":" claude-sonnet "}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	id, err := ReadModelID(path)
	if err != nil || id != "claude-sonnet" {
		t.Fatalf("id = %q %v", id, err)
	}
}

func TestBuildLaunch(t *testing.T) {
	claude, err := BuildLaunch("claude", "http://127.0.0.1:8317/", "sk-test", "sonnet", []string{"--resume"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(claude.Argv, " ") != "claude --model sonnet --resume" {
		t.Fatalf("claude argv = %#v", claude.Argv)
	}
	if claude.Env["ANTHROPIC_BASE_URL"] != "http://127.0.0.1:8317" || claude.Env["ANTHROPIC_AUTH_TOKEN"] != "sk-test" || claude.Env["ANTHROPIC_MODEL"] != "sonnet" || claude.Env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] != "sonnet" {
		t.Fatalf("claude env = %#v", claude.Env)
	}
	if len(claude.Clear) != 1 || claude.Clear[0] != "ANTHROPIC_API_KEY" {
		t.Fatalf("claude clear = %#v", claude.Clear)
	}
	codex, err := BuildLaunch("codex", "http://127.0.0.1:8317", "sk-test", "gpt space", []string{"exec", "hi"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(codex.Argv, " ")
	if !strings.HasPrefix(joined, "codex -c ") || !strings.HasSuffix(joined, "exec hi") {
		t.Fatalf("codex argv = %s", joined)
	}
	if !strings.Contains(joined, "base_url=http://127.0.0.1:8317/v1") || !strings.Contains(joined, `model="gpt space"`) {
		t.Fatalf("codex config = %s", joined)
	}
	if strings.Contains(joined, "sk-test") || codex.Env[CodexKeyEnv] != "sk-test" {
		t.Fatalf("key leaked into argv or missing from env: %s %#v", joined, codex.Env)
	}
	opencode, err := BuildLaunch("opencode", "http://127.0.0.1:8317", "sk-test", "sonnet", []string{"--port", "1"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(opencode.Argv, " ") != "opencode --port 1" || !strings.Contains(opencode.Env["OPENCODE_CONFIG_CONTENT"], `"baseURL":"http://127.0.0.1:8317/v1"`) || !strings.Contains(opencode.Env["OPENCODE_CONFIG_CONTENT"], `"model":"akproxy/sonnet"`) {
		t.Fatalf("opencode = %#v", opencode)
	}
	pi, err := BuildLaunch("pi", "http://127.0.0.1:8317", "sk-test", "sonnet", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(pi.Dir) })
	if strings.Join(pi.Argv, " ") != "pi" || pi.Env["PI_CODING_AGENT_DIR"] == "" || pi.Env[CodexKeyEnv] != "sk-test" {
		t.Fatalf("pi = %#v", pi)
	}
	models, err := os.ReadFile(filepath.Join(pi.Dir, "models.json"))
	if err != nil || !strings.Contains(string(models), "http://127.0.0.1:8317/v1") || !strings.Contains(string(models), "$"+CodexKeyEnv) {
		t.Fatalf("pi models = %s %v", models, err)
	}
	if _, err := BuildLaunch("claude", "http://127.0.0.1:8317", "", "m", nil); err == nil {
		t.Fatal("missing key should fail")
	}
}

func TestSelectModelSavesTheGumChoice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-test" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"zeta"},{"id":"alpha"},{"id":"alpha"}]}`)
	}))
	t.Cleanup(server.Close)
	port := server.Listener.Addr().(*net.TCPAddr).Port
	dir := t.TempDir()
	paths := desktop.Paths{Root: dir, Config: filepath.Join(dir, "config.yaml"), Auth: filepath.Join(dir, "auths")}
	if err := desktop.Ensure(paths); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(paths.Config)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(body), `port: 8317`, "port: "+strconv.Itoa(port), 1)
	keyLine := regexp.MustCompile(`(?m)^  - "sk-[^"]*"`)
	updated = keyLine.ReplaceAllString(updated, `  - "sk-test"`)
	if err := os.WriteFile(paths.Config, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = selectModel(&out, paths, func(models []string, current string) (string, error) {
		if current != "" || strings.Join(models, ",") != "alpha,zeta" {
			t.Fatalf("picker got %q %q", models, current)
		}
		return "zeta", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != "zeta\n" {
		t.Fatalf("stdout = %q", out.String())
	}
	id, err := ReadModelID(filepath.Join(dir, "cli.json"))
	if err != nil || id != "zeta" {
		t.Fatalf("saved model = %q %v", id, err)
	}
	out.Reset()
	if err := selectModel(&out, paths, func(_ []string, current string) (string, error) {
		if current != "zeta" {
			t.Fatalf("current = %q", current)
		}
		return "", errors.New("已取消")
	}); err == nil || err.Error() != "已取消" {
		t.Fatalf("cancel = %v", err)
	}
	id, err = ReadModelID(filepath.Join(dir, "cli.json"))
	if err != nil || id != "zeta" {
		t.Fatalf("cancel changed model to %q %v", id, err)
	}
}

func TestExecuteStopsBeforeLaunch(t *testing.T) {
	dir := t.TempDir()
	paths := desktop.Paths{Root: dir, Config: filepath.Join(dir, "config.yaml"), Auth: filepath.Join(dir, "auths")}
	var out bytes.Buffer
	if err := Execute(&out, paths, []string{"claude"}); err == nil || !strings.Contains(err.Error(), "请先打开 akproxy") {
		t.Fatalf("missing app: %v", err)
	}
	if err := desktop.Ensure(paths); err != nil {
		t.Fatal(err)
	}
	if err := Execute(&out, paths, []string{"codex", "exec"}); err == nil || !strings.Contains(err.Error(), "akproxy model") {
		t.Fatalf("missing model: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cli.json"), []byte(`{"model":{"id":"sonnet"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := start
	var launched Launch
	start = func(launch Launch) error {
		launched = launch
		return nil
	}
	t.Cleanup(func() { start = previous })
	if err := Execute(&out, paths, []string{"claude", "--resume"}); err != nil {
		t.Fatal(err)
	}
	if launched.Env["ANTHROPIC_AUTH_TOKEN"] == "" || !strings.Contains(strings.Join(launched.Argv, " "), "--model sonnet --resume") {
		t.Fatalf("launch = %#v", launched)
	}
	out.Reset()
	if err := Execute(&out, paths, []string{"--help"}); err != nil || !strings.Contains(out.String(), "claude") {
		t.Fatalf("help: %v %q", err, out.String())
	}
}
