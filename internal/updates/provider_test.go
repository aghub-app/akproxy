package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

func TestOfficialUpdaterWithGitHubReleases(t *testing.T) {
	for _, tc := range []struct {
		name, tag, checksum, wantError         string
		status                                 int
		prerelease, noAsset, noUpdate, corrupt bool
	}{
		{name: "verified download waits for restart", tag: "v1.1.0"},
		{name: "current", tag: "v1.0.0", noUpdate: true},
		{name: "no downgrade", tag: "v0.9.0", noUpdate: true},
		{name: "prerelease ignored", tag: "v1.1.0-beta.1", prerelease: true, noUpdate: true},
		{name: "draft only or no releases", status: 404, noUpdate: true},
		{name: "missing checksum", tag: "v1.1.0", checksum: "missing", wantError: "SHA-256"},
		{name: "malformed checksum", tag: "v1.1.0", checksum: "00", wantError: "SHA-256"},
		{name: "corrupt download", tag: "v1.1.0", checksum: strings.Repeat("0", 64), wantError: "digest mismatch"},
		{name: "invalid archive", tag: "v1.1.0", corrupt: true, wantError: "zip"},
		{name: "wrong architecture", tag: "v1.1.0", noAsset: true, wantError: "no asset"},
		{name: "rate limited", status: 403, wantError: "403"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Official updater staging is contained in this test's disposable directory.
			for _, env := range []string{"TMPDIR", "TMP", "TEMP"} {
				t.Setenv(env, t.TempDir())
			}
			var archive bytes.Buffer
			zw := zip.NewWriter(&archive)
			entry, err := zw.Create("akproxy.app/Contents/MacOS/akproxy")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte("new executable")); err != nil {
				t.Fatal(err)
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			payload := archive.Bytes()
			if tc.corrupt {
				payload = []byte("not a zip")
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(payload))
			if tc.checksum != "" {
				digest = tc.checksum
			}
			const asset = "akproxy-darwin-universal.zip"
			downloads := 0
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/repos/aghub-app/akproxy/releases/latest":
					if tc.status != 0 {
						w.WriteHeader(tc.status)
						return
					}
					assets := []map[string]any{}
					if !tc.noAsset {
						assets = append(assets, map[string]any{"name": asset, "size": len(payload), "browser_download_url": server.URL + "/asset"})
					}
					if tc.checksum != "missing" {
						assets = append(assets, map[string]any{"name": "SHA256SUMS", "browser_download_url": server.URL + "/sums"})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"tag_name": tc.tag, "prerelease": tc.prerelease, "assets": assets})
				case "/sums":
					fmt.Fprintf(w, "%s  %s\n", digest, asset)
				case "/asset":
					downloads++
					_, _ = w.Write(payload)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			provider, err := NewProvider(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			host := &testHost{}
			u := updater.New(host)
			if err := u.Init(updater.Config{CurrentVersion: "1.0.0", Platform: "darwin", Arch: "arm64", Providers: []updater.Provider{provider}, Window: updater.WindowNone}); err != nil {
				t.Fatal(err)
			}
			err = u.CheckAndInstall(context.Background())
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %s", err, tc.wantError)
				}
				if u.DownloadedPath() != "" || u.State() != updater.StateError {
					t.Fatal("failed update became installable")
				}
			} else if err != nil {
				t.Fatal(err)
			} else if tc.noUpdate {
				if u.State() != updater.StateUpToDate || downloads != 0 {
					t.Fatal("downloaded an excluded release")
				}
			} else {
				if u.State() != updater.StateReady || downloads != 1 {
					t.Fatalf("state %s, downloads %d", u.State(), downloads)
				}
				data, err := os.ReadFile(filepath.Join(u.DownloadedPath(), "Contents", "MacOS", "akproxy"))
				if err != nil || string(data) != "new executable" {
					t.Fatalf("staged content: %q, %v", data, err)
				}
			}
			if host.quit {
				t.Fatal("download must not restart the application")
			}
		})
	}
}

func TestExactPlatformAssets(t *testing.T) {
	assets := []github.ReleaseAsset{
		{Name: "akproxy.dmg"}, {Name: "akproxy-darwin-universal.zip"},
		{Name: "akproxy-windows-amd64.zip"}, {Name: "akproxy-linux-amd64.tar.gz"},
	}
	for _, tc := range []struct {
		platform, arch string
		index          int
	}{
		{"darwin", "arm64", 1}, {"darwin", "amd64", 1}, {"windows", "amd64", 2}, {"linux", "amd64", 3}, {"linux", "arm64", -1},
	} {
		if got := MatchAsset(updater.CheckRequest{Platform: tc.platform, Arch: tc.arch}, assets); got != tc.index {
			t.Fatalf("%s/%s = %d", tc.platform, tc.arch, got)
		}
	}
}

type testHost struct{ quit bool }

func (*testHost) Emit(string, ...any) bool         { return true }
func (*testHost) OnEvent(string, func(any)) func() { return func() {} }
func (*testHost) OpenWindow(updater.WindowOptions) updater.WindowHandle {
	panic("headless test opened a window")
}
func (h *testHost) Quit() { h.quit = true }
