//go:build e2e

// Package e2e runs the built akproxy binaries against an isolated home
// directory. Run with `go test -tags e2e ./e2e/`.
package e2e

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"akproxy/internal/desktop"
)

var (
	cliBin   string
	agentBin string
)

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "akproxy-e2e-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cliBin = filepath.Join(dir, exe("akproxy"))
	agentBin = filepath.Join(dir, exe("fakeagent"))
	for _, build := range [][]string{
		{"-o", cliBin, "akproxy/cmd/akproxy"},
		{"-o", agentBin, "./testdata/fakeagent"},
	} {
		cmd := exec.Command("go", append([]string{"build"}, build...)...)
		cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
		if out, err := cmd.CombinedOutput(); err != nil {
			fmt.Fprintf(os.Stderr, "go build %v: %v\n%s", build, err, out)
			os.Exit(1)
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// sandbox is one user's home, config directory, and environment.
type sandbox struct {
	home   string
	paths  desktop.Paths
	agents string
	env    map[string]string
}

var sandboxedKeys = []string{"HOME", "USERPROFILE", "APPDATA", "XDG_CONFIG_HOME", "PATH", "ZDOTDIR"}

func newSandbox(t *testing.T) *sandbox {
	t.Helper()
	home := t.TempDir()
	sb := &sandbox{home: home, agents: t.TempDir(), env: map[string]string{}}
	for _, item := range os.Environ() {
		key, value, _ := strings.Cut(item, "=")
		sb.env[key] = value
	}
	for key := range sb.env {
		if strings.HasPrefix(key, "ANTHROPIC_") || strings.HasPrefix(key, "AKPROXY_") || strings.HasPrefix(key, "OPENCODE_") || strings.HasPrefix(key, "PI_") {
			delete(sb.env, key)
		}
	}
	for _, key := range sandboxedKeys {
		delete(sb.env, key)
	}
	sb.env["HOME"] = home
	sb.env["USERPROFILE"] = home
	sb.env["APPDATA"] = filepath.Join(home, "AppData", "Roaming")
	sb.env["XDG_CONFIG_HOME"] = filepath.Join(home, ".config")
	sb.env["PATH"] = sb.agents
	var base string
	switch runtime.GOOS {
	case "darwin":
		base = filepath.Join(home, "Library", "Application Support")
	case "windows":
		base = sb.env["APPDATA"]
	default:
		base = sb.env["XDG_CONFIG_HOME"]
	}
	root := filepath.Join(base, "akproxy")
	sb.paths = desktop.Paths{Root: root, Config: filepath.Join(root, "config.yaml"), Auth: filepath.Join(root, "auths")}
	return sb
}

func (sb *sandbox) environ(extra map[string]string) []string {
	merged := map[string]string{}
	for key, value := range sb.env {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	out := make([]string, 0, len(merged))
	for key, value := range merged {
		out = append(out, key+"="+value)
	}
	return out
}

// openApp creates the config the desktop app writes on first launch and
// points it at port. It returns the proxy address and client key.
func (sb *sandbox) openApp(t *testing.T, port int) (string, string) {
	t.Helper()
	if err := desktop.Ensure(sb.paths); err != nil {
		t.Fatal(err)
	}
	return sb.setPort(t, port)
}

var portLine = regexp.MustCompile(`(?m)^port: \d+$`)

func (sb *sandbox) setPort(t *testing.T, port int) (string, string) {
	t.Helper()
	body, err := os.ReadFile(sb.paths.Config)
	if err != nil {
		t.Fatal(err)
	}
	next := portLine.ReplaceAllString(string(body), "port: "+strconv.Itoa(port))
	if err := os.WriteFile(sb.paths.Config, []byte(next), 0o600); err != nil {
		t.Fatal(err)
	}
	address, key, err := desktop.ClientCredential(sb.paths.Config)
	if err != nil {
		t.Fatal(err)
	}
	return address, key
}

func (sb *sandbox) cliJSON() string {
	return filepath.Join(sb.paths.Root, "cli.json")
}

func (sb *sandbox) chooseModel(t *testing.T, id string) {
	t.Helper()
	body := `{"model":{"id":` + strconv.Quote(id) + `}}` + "\n"
	if err := os.WriteFile(sb.cliJSON(), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// installAgent puts the fake agent on PATH under name.
func (sb *sandbox) installAgent(t *testing.T, dir, name string) {
	t.Helper()
	body, err := os.ReadFile(agentBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, exe(name)), body, 0o755); err != nil {
		t.Fatal(err)
	}
}

type result struct {
	out  string
	code int
}

func run(t *testing.T, sb *sandbox, extra map[string]string, bin string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = sb.environ(extra)
	cmd.Dir = sb.home
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	code := 0
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("run %s %v: %v", bin, args, err)
	}
	return result{out: strings.TrimSpace(buf.String()), code: code}
}

// agentRecord is what testdata/fakeagent wrote about its launch.
type agentRecord struct {
	Name  string            `json:"name"`
	Args  []string          `json:"args"`
	Env   map[string]string `json:"env"`
	Files map[string]string `json:"files"`
}

func readRecord(t *testing.T, path string) agentRecord {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("agent was not launched: %v", err)
	}
	var rec agentRecord
	if err := json.Unmarshal(body, &rec); err != nil {
		t.Fatal(err)
	}
	return rec
}
