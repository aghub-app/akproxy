package desktop

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestRuntimeLifecycleOldExitDoesNotClearNewRun(t *testing.T) {
	oldDone := make(chan struct{})
	newDone := make(chan struct{})
	r := &Runtime{done: newDone, running: true, cancel: func() {}, emit: func(string, any) {}}
	close(oldDone)
	r.watchExit(oldDone, make(chan error, 1))
	if !r.running || r.done != newDone || r.cancel == nil {
		t.Fatal("old exit cleared the current run")
	}
}

func TestRuntimeLifecycleConcurrentStartOwnsOneListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	root := t.TempDir()
	authDir := filepath.Join(root, "auths")
	if err := os.Mkdir(authDir, 0o700); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.yaml")
	body := strings.Replace(defaultConfig(authDir, "sk-test"), "port: 8317", "port: "+strconv.Itoa(port), 1)
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	r := &Runtime{paths: Paths{Config: configPath, Auth: authDir}, emit: func(string, any) {}}
	gate := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-gate
			results <- r.Start()
		}()
	}
	close(gate)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("start successes = %d, want 1", successes)
	}
	t.Cleanup(func() { _ = r.Stop() })
	status, err := r.Status()
	if err != nil || !status.Running {
		t.Fatalf("listener status: %+v, %v", status, err)
	}
	connection, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), time.Second)
	if err != nil {
		t.Fatalf("running service is not listening: %v", err)
	}
	_ = connection.Close()
}
