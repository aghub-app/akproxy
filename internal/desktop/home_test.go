package desktop

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHomeSetupAndBoundModels(t *testing.T) {
	root := t.TempDir()
	r := &Runtime{paths: Paths{Root: root, Config: filepath.Join(root, "config.yaml"), Auth: filepath.Join(root, "auths")}}
	if err := Ensure(r.paths); err != nil {
		t.Fatal(err)
	}
	write := func(extra string) {
		t.Helper()
		if err := os.WriteFile(r.paths.Config, []byte(defaultConfig(r.paths.Auth, "client-test")+extra), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, extra := range []string{
		"codex-api-key:\n  - api-key: upstream\n    base-url: https://example.test\n",
		"claude-api-key:\n  - api-key: upstream\n    base-url: https://example.test\n",
		"gemini-api-key:\n  - api-key: upstream\n    base-url: https://example.test\n",
		"xai-api-key:\n  - api-key: upstream\n    base-url: https://example.test\n",
		"openai-compatibility:\n  - name: kimi\n    base-url: https://example.test\n    api-key-entries:\n      - api-key: upstream\n",
		"openai-compatibility:\n  - name: custom\n    base-url: https://example.test\n    api-key-entries:\n      - api-key: upstream\n",
	} {
		write(extra)
		home, err := r.Home()
		if err != nil || !home.HasCredentials || len(home.ClientKeys) != 1 || home.ClientKeys[0] != "client-test" {
			t.Fatalf("credential branch %q: %+v %v", extra, home, err)
		}
	}
	if err := os.WriteFile(r.paths.Config, []byte(strings.Replace(defaultConfig(r.paths.Auth, "client-test"), "  - \"client-test\"\n", "  - \"client-test\"\n  - \"client-next\"\n", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	home, err := r.Home()
	if err != nil || strings.Join(home.ClientKeys, ",") != "client-test,client-next" {
		t.Fatalf("client key choices: %+v %v", home, err)
	}
	write("")
	home, err = r.Home()
	if err != nil || home.HasCredentials {
		t.Fatalf("client key must not count: %+v %v", home, err)
	}
	account := filepath.Join(r.paths.Auth, "codex.json")
	if err := os.WriteFile(account, []byte(`{"type":"codex","email":"example@example.test","access_token":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}
	home, err = r.Home()
	if err != nil || !home.HasCredentials {
		t.Fatalf("login branch: %+v %v", home, err)
	}
	if err := r.DeleteAccount("codex.json"); err != nil {
		t.Fatal(err)
	}
	home, err = r.Home()
	if err != nil || home.HasCredentials {
		t.Fatalf("delete last credential: %+v %v", home, err)
	}
	if _, err := r.HomeModels(); err == nil {
		t.Fatal("stopped service queried")
	}
	response := `{"data":[{"id":"b"},{"id":"a"},{"id":"a"},{"id":" "}]}`
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/models" || req.Header.Get("Authorization") != "Bearer client-test" {
			t.Error("wrong model endpoint or auth")
		}
		w.Header().Set("Location", "https://example.test")
		w.WriteHeader(status)
		fmt.Fprint(w, response)
	}))
	defer server.Close()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(server.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	n, _ := strconv.Atoi(port)
	r.running, r.bound = true, Listen{Host: host, Port: n}
	home, err = r.Home()
	if err != nil || home.Status.Address != server.URL || !home.Status.RestartRequired {
		t.Fatalf("must use old binding: %+v %v", home, err)
	}
	models, err := r.HomeModels()
	if err != nil || strings.Join(models, ",") != "a,b" {
		t.Fatalf("models: %v %v", models, err)
	}
	for _, code := range []int{http.StatusUnauthorized, http.StatusFound} {
		status = code
		if _, err := r.HomeModels(); err == nil {
			t.Fatalf("accepted HTTP %d", code)
		}
	}
	status = http.StatusOK
	for _, body := range []string{`not json`, `{}`, `{"data":[{"id":123}]}`} {
		response = body
		if _, err := r.HomeModels(); err == nil {
			t.Fatalf("accepted invalid payload %s", body)
		}
	}
	response = `{"data":[]}`
	if models, err := r.HomeModels(); err != nil || len(models) != 0 {
		t.Fatalf("empty list: %v %v", models, err)
	}
}
