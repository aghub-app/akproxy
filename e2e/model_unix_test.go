//go:build e2e && !windows

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

// terminal drives the CLI through a pseudo-terminal, as a user's shell does.
type terminal struct {
	cmd  *exec.Cmd
	tty  *os.File
	mu   sync.Mutex
	seen bytes.Buffer
	done chan error
}

func startTerminal(t *testing.T, sb *sandbox, args ...string) *terminal {
	t.Helper()
	cmd := exec.Command(cliBin, args...)
	cmd.Env = sb.environ(map[string]string{"TERM": "xterm-256color"})
	tty, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatal(err)
	}
	term := &terminal{cmd: cmd, tty: tty, done: make(chan error, 1)}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := tty.Read(buf)
			term.mu.Lock()
			term.seen.Write(buf[:n])
			term.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	go func() { term.done <- cmd.Wait() }()
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		tty.Close()
	})
	return term
}

func (term *terminal) output() string {
	term.mu.Lock()
	defer term.mu.Unlock()
	return term.seen.String()
}

func (term *terminal) waitFor(t *testing.T, text string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(term.output(), text) {
		if time.Now().After(deadline) {
			t.Fatalf("terminal never showed %q:\n%s", text, term.output())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (term *terminal) send(t *testing.T, keys string) {
	t.Helper()
	if _, err := term.tty.WriteString(keys); err != nil {
		t.Fatal(err)
	}
}

func (term *terminal) exit(t *testing.T) int {
	t.Helper()
	select {
	case err := <-term.done:
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		if err != nil {
			t.Fatal(err)
		}
		return 0
	case <-time.After(15 * time.Second):
		t.Fatalf("akproxy did not exit:\n%s", term.output())
		return -1
	}
}

func TestCLIModelPicksWithGum(t *testing.T) {
	sb := newSandbox(t)
	_, key := sb.openApp(t, 8317)
	sb.setPort(t, modelServer(t, key, "m-a", "m-b", "m-c"))
	sb.chooseModel(t, "m-a")

	term := startTerminal(t, sb, "model")
	term.waitFor(t, "m-c")
	term.send(t, "\x1b[B") // down from the current m-a
	time.Sleep(200 * time.Millisecond)
	term.send(t, "\r")
	if code := term.exit(t); code != 0 {
		t.Fatalf("akproxy model exit %d:\n%s", code, term.output())
	}
	body, err := os.ReadFile(sb.cliJSON())
	if err != nil || !strings.Contains(string(body), `"m-b"`) {
		t.Fatalf("cli.json = %s %v", body, err)
	}
	if !strings.Contains(term.output(), "m-b") {
		t.Fatalf("chosen id should be printed:\n%s", term.output())
	}
}

func TestCLIModelCancelKeepsSelection(t *testing.T) {
	sb := newSandbox(t)
	_, key := sb.openApp(t, 8317)
	sb.setPort(t, modelServer(t, key, "m-a", "m-b"))
	sb.chooseModel(t, "m-a")

	term := startTerminal(t, sb, "model")
	term.waitFor(t, "m-b")
	term.send(t, "\x1b[B")
	time.Sleep(200 * time.Millisecond)
	term.send(t, "\x03") // Ctrl-C
	if code := term.exit(t); code == 0 {
		t.Fatalf("cancel should fail:\n%s", term.output())
	}
	body, _ := os.ReadFile(sb.cliJSON())
	if !strings.Contains(string(body), `"m-a"`) {
		t.Fatalf("cancel changed cli.json: %s", body)
	}
}
