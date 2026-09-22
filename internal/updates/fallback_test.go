package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

const fixtureAsset = "akproxy-darwin-universal.zip"

type fixture struct {
	tag        string
	payload    []byte
	apiStatus  int
	apiBody    string
	atomStatus int
	sumsStatus int
	sumsBody   string
	apiHits    int
	atomHits   int
}

func newFallback(t *testing.T, f *fixture) *FallbackProvider {
	t.Helper()
	var sums string
	if f.payload != nil {
		sums = fmt.Sprintf("%x  %s\n", sha256.Sum256(f.payload), fixtureAsset)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/aghub-app/akproxy/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		f.apiHits++
		if f.apiStatus != 0 {
			http.Error(w, f.apiBody, f.apiStatus)
			return
		}
		http.Error(w, "unexpected api call", http.StatusInternalServerError)
	})
	mux.HandleFunc("/repo/releases.atom", func(w http.ResponseWriter, r *http.Request) {
		f.atomHits++
		if f.atomStatus != 0 {
			http.Error(w, "atom error", f.atomStatus)
			return
		}
		// The id carries the immutable tag; the title is the editable
		// release name, which may differ.
		fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>tag:github.com,2008:Repository/1/%s</id><updated>%s</updated><title>Release name that is not the tag</title></entry></feed>`,
			f.tag, time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC).Format(time.RFC3339))
	})
	mux.HandleFunc("/repo/releases/download/v1.1.0/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) {
		if f.sumsStatus != 0 {
			http.Error(w, "sums error", f.sumsStatus)
			return
		}
		if f.sumsBody != "" {
			fmt.Fprint(w, f.sumsBody)
			return
		}
		fmt.Fprint(w, sums)
	})
	mux.HandleFunc("/repo/releases/download/v1.1.0/"+fixtureAsset, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(f.payload)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	primary, err := NewProvider(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	fb := NewFallbackProvider(primary)
	fb.atom.Repository = "repo"
	fb.atom.host = server.URL
	return fb
}

func zipPayload(t *testing.T) []byte {
	t.Helper()
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
	return archive.Bytes()
}

func TestFallbackOnRateLimitInstalls(t *testing.T) {
	for _, env := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(env, t.TempDir())
	}
	payload := zipPayload(t)
	f := &fixture{tag: "v1.1.0", payload: payload, apiStatus: 403, apiBody: "API rate limit exceeded"}
	fb := newFallback(t, f)

	host := &testHost{}
	u := updater.New(host)
	if err := u.Init(updater.Config{CurrentVersion: "1.0.0", Platform: "darwin", Arch: "arm64", Providers: []updater.Provider{fb}, Window: updater.WindowNone}); err != nil {
		t.Fatal(err)
	}
	if err := u.CheckAndInstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	if u.State() != updater.StateReady {
		t.Fatalf("state %s, want ready", u.State())
	}
	if f.apiHits != 1 || f.atomHits != 1 {
		t.Fatalf("api hits %d, atom hits %d", f.apiHits, f.atomHits)
	}
	if host.quit {
		t.Fatal("download must not restart the application")
	}
}

func TestFallbackMatrix(t *testing.T) {
	payload := zipPayload(t)
	for _, tc := range []struct {
		name       string
		apiStatus  int
		apiBody    string
		atomStatus int
		sumsStatus int
		sumsBody   string
		wantErr    string
		wantAtom   bool
	}{
		{name: "429 falls back to atom outage", apiStatus: 429, apiBody: "too many requests", atomStatus: 500, wantErr: "atom fallback", wantAtom: true},
		{name: "502 falls back to atom outage", apiStatus: 502, apiBody: "bad gateway", atomStatus: 500, wantErr: "atom fallback", wantAtom: true},
		{name: "atom checksum missing fails", apiStatus: 403, apiBody: "rate limit", sumsStatus: 404, wantErr: "checksum: HTTP 404", wantAtom: true},
		{name: "atom checksum line missing fails closed", apiStatus: 403, apiBody: "rate limit", sumsBody: "deadbeef  other-file.zip\n", wantErr: "not found", wantAtom: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fixture{tag: "v1.1.0", payload: payload, apiStatus: tc.apiStatus, apiBody: tc.apiBody, atomStatus: tc.atomStatus, sumsStatus: tc.sumsStatus, sumsBody: tc.sumsBody}
			fb := newFallback(t, f)
			rel, err := fb.Check(context.Background(), updater.CheckRequest{CurrentVersion: "1.0.0", Platform: "darwin", Arch: "arm64"})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want %s", err, tc.wantErr)
			}
			if rel != nil {
				t.Fatal("error case returned a release")
			}
			if tc.wantAtom != (f.atomHits > 0) {
				t.Fatalf("atom hits %d, wantAtom %v", f.atomHits, tc.wantAtom)
			}
		})
	}
}

func TestFallbackSkipsAtomOn404(t *testing.T) {
	// A 404 from the API means the repository has no releases; the atom
	// feed would be empty too, so no second request is made. The updater
	// treats (nil, nil) as up-to-date.
	f := &fixture{tag: "v1.1.0", payload: zipPayload(t), apiStatus: 404, apiBody: "Not Found"}
	fb := newFallback(t, f)
	rel, err := fb.Check(context.Background(), updater.CheckRequest{CurrentVersion: "1.0.0", Platform: "darwin", Arch: "arm64"})
	if err != nil {
		t.Fatalf("404 became an error: %v", err)
	}
	if rel != nil {
		t.Fatal("404 returned a release")
	}
	if f.atomHits != 0 {
		t.Fatalf("atom hits %d, want 0", f.atomHits)
	}
}

func TestShouldFallbackClassification(t *testing.T) {
	primaryErr := func(status int, body string) error {
		return fmt.Errorf("github: api %d: %s", status, body)
	}
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{primaryErr(403, "API rate limit exceeded for 1.2.3.4."), true},
		{primaryErr(429, "too many requests"), true},
		{primaryErr(502, "bad gateway"), true},
		{primaryErr(404, "Not Found"), false},
		// A status number appearing inside the response body must not
		// drive the decision.
		{primaryErr(400, `{"message":"branch v0.4.03 not found"}`), false},
		{errors.New("github: asset missing"), false},
	} {
		if got := shouldFallback(tc.err); got != tc.want {
			t.Errorf("shouldFallback(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
	var opErr *net.OpError
	if !shouldFallback(error(opErr)) {
		t.Error("typed transport error should fall back")
	}
}
