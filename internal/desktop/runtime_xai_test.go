package desktop

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	"golang.org/x/mod/semver"
)

// Exercise the embedded executor through the same endpoint used by local clients.
func TestRuntimeXAIOAuthUsesSupportedGrokClientVersion(t *testing.T) {
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
	credential := `{"type":"xai","auth_kind":"oauth","access_token":"fake-xai-token","expired":"2099-01-01T00:00:00Z"}`
	if err := os.WriteFile(filepath.Join(authDir, "xai-test.json"), []byte(credential), 0o600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config.yaml")
	config := strings.Replace(defaultConfig(authDir, "sk-test"), "port: 8317", "port: "+strconv.Itoa(port), 1)
	config = strings.Replace(config, "request-retry: 3", "request-retry: 0", 1)
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime := &Runtime{paths: Paths{Config: configPath, Auth: authDir}, emit: func(string, any) {}}
	if err := runtime.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtime.Stop() })

	var upstreamCalls atomic.Int32
	runtime.coreAuth.SetRoundTripperProvider(xaiTestTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Scheme != "https" || req.URL.Host != "cli-chat-proxy.grok.com" || req.URL.Path != "/v1/responses" || req.Method != http.MethodPost {
			return nil, fmt.Errorf("unexpected upstream request: %s %s", req.Method, req.URL)
		}
		upstreamCalls.Add(1)
		version := req.Header.Get("x-grok-client-version")
		if !semver.IsValid("v"+version) || semver.Compare("v"+version, "v1.0.13") < 0 {
			t.Errorf("embedded xAI executor advertised unsupported Grok CLI version %q", version)
			return &http.Response{
				StatusCode: http.StatusUpgradeRequired,
				Header:     http.Header{"Content-Type": {"application/json"}},
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"Your Grok CLI version is outdated. Please update to version 1.0.13 or later via grok update"}}`)),
			}, nil
		}
		if got := req.Header.Get("User-Agent"); got != "xai-grok-workspace/"+version {
			t.Errorf("User-Agent = %q, want version matching %q", got, version)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer fake-xai-token" {
			t.Errorf("upstream Authorization = %q, want fake OAuth token", got)
		}
		body := "event: response.completed\ndata: " + `{"type":"response.completed","response":{"id":"resp-test","object":"response","created_at":1,"status":"completed","model":"grok-4.7","output":[{"id":"msg-test","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"pong","annotations":[]}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}` + "\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}))

	ready := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline) && !ready; {
		models, err := runtime.HomeModels()
		if err != nil {
			t.Fatal(err)
		}
		for _, model := range models {
			ready = ready || model == "grok-4.7"
		}
		if !ready {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if !ready {
		t.Fatal("fake xAI OAuth account exposed no grok-4.7 model")
	}
	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, url, strings.NewReader(`{"model":"grok-4.7","messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer sk-test")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Transport: &http.Transport{}, Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chat completion status = %d, body = %s", resp.StatusCode, body)
	}
	var completion struct {
		Choices []struct {
			Message struct{ Content string }
		}
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		t.Fatal(err)
	}
	if len(completion.Choices) != 1 || completion.Choices[0].Message.Content != "pong" {
		t.Fatalf("chat completion did not return assistant text pong: %s", body)
	}
	if upstreamCalls.Load() == 0 {
		t.Fatal("chat completion never reached the embedded xAI executor")
	}
}

type xaiTestTransport func(*http.Request) (*http.Response, error)

func (transport xaiTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func (transport xaiTestTransport) RoundTripperFor(*coreauth.Auth) http.RoundTripper {
	return transport
}
