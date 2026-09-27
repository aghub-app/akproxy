//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"akproxy/internal/desktop"
)

// launchApp starts the built desktop app from $AKPROXY_E2E_APP in the sandbox.
func launchApp(t *testing.T, sb *sandbox) func() {
	t.Helper()
	app := os.Getenv("AKPROXY_E2E_APP")
	if app == "" {
		t.Skip("set AKPROXY_E2E_APP to the built desktop executable")
	}
	cmd := exec.Command(app)
	cmd.Env = sb.environ(map[string]string{"PATH": os.Getenv("PATH")})
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(exited) }()
	stop := func() {
		select {
		case <-exited:
		default:
			_ = cmd.Process.Kill()
			<-exited
		}
	}
	t.Cleanup(stop)
	return stop
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func commandPath(sb *sandbox) string {
	return filepath.Join(sb.home, ".local", "bin", exe("akproxy"))
}

// The first launch puts `akproxy` on PATH for new terminals. The command then
// launches an agent against the app's own config. Turning the switch off and
// relaunching removes both the command and the PATH entry.
func TestAppInstallsCommandForNewTerminals(t *testing.T) {
	guardUserPath(t)
	sb := newSandbox(t)
	want := os.Getenv("AKPROXY_E2E_VERSION")
	if want == "" {
		want = "dev"
	}

	stop := launchApp(t, sb)
	eventually(t, "akproxy command installed", func() bool {
		got, err := freshShell(t, sb, nil, "akproxy --version")
		return err == nil && got == want
	})

	// Claude usually lives in ~/.local/bin too; any agent on PATH works.
	sb.installAgent(t, filepath.Join(sb.home, ".local", "bin"), "claude")
	sb.chooseModel(t, model)
	address, key, err := desktop.ClientCredential(sb.paths.Config)
	if err != nil {
		t.Fatalf("app config: %v", err)
	}
	out := filepath.Join(t.TempDir(), "rec.json")
	if got, err := freshShell(t, sb, map[string]string{"AKPROXY_FAKE_OUT": out}, "akproxy claude --e2e"); err != nil {
		t.Fatalf("akproxy claude in a new terminal: %v %s", err, got)
	}
	rec := readRecord(t, out)
	if rec.Env["ANTHROPIC_BASE_URL"] != address || rec.Env["ANTHROPIC_AUTH_TOKEN"] != key || rec.Args[len(rec.Args)-1] != "--e2e" {
		t.Fatalf("claude launch = %+v", rec)
	}
	stop()

	prefs := desktop.ReadAppPrefs(sb.paths.Root)
	prefs.CLIEnabled = false
	if err := desktop.WriteAppPrefs(sb.paths.Root, prefs); err != nil {
		t.Fatal(err)
	}
	stop = launchApp(t, sb)
	eventually(t, "akproxy command removed", func() bool {
		_, err := os.Stat(commandPath(sb))
		return os.IsNotExist(err) && !userPathHasCommandDir(t, sb)
	})
	stop()
	if _, err := os.Stat(filepath.Join(sb.home, ".local", "bin", exe("claude"))); err != nil {
		t.Fatalf("removal must leave other commands alone: %v", err)
	}
	if got, err := freshShell(t, sb, nil, "akproxy --version"); err == nil {
		t.Fatalf("akproxy still runs in a new terminal: %s", got)
	}
}

func trimLines(out []byte) string {
	return strings.TrimSpace(strings.ReplaceAll(string(out), "\r\n", "\n"))
}
