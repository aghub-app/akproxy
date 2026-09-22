package updates

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	xsemver "golang.org/x/mod/semver"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

const atomMaxBody = 4 << 20

type atomFeed struct {
	Entries []struct {
		ID        string    `xml:"id"`
		UpdatedAt time.Time `xml:"updated"`
	} `xml:"entry"`
}

// tagFromEntryID extracts the release tag from an entry id like
// "tag:github.com,2008:Repository/<id>/v1.2.3". The id carries the
// immutable tag; the entry title is the editable release name.
func tagFromEntryID(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[i+1:]
	}
	return ""
}

// errNoStableTag reports that the feed has no plain vMAJOR.MINOR.PATCH entry.
var errNoStableTag = errors.New("atom: no stable release in feed")

// AtomSource resolves the latest stable release from the releases.atom feed
// on github.com, which does not share the api.github.com rate limit.
type AtomSource struct {
	// Repository is "owner/repo".
	Repository string
	Client     *http.Client

	// host overrides "https://github.com" for tests; it may carry a scheme.
	host string
}

func (s *AtomSource) baseURL() string {
	if s.host != "" {
		return s.host
	}
	return "https://github.com"
}

// Latest returns the newest pure three-segment tag in the feed.
func (s *AtomSource) Latest(ctx context.Context) (tag string, published time.Time, err error) {
	url := s.baseURL() + fmt.Sprintf("/%s/releases.atom", s.Repository)
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", time.Time{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("atom: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", time.Time{}, fmt.Errorf("atom: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, atomMaxBody))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("atom: %w", err)
	}
	var feed atomFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return "", time.Time{}, fmt.Errorf("atom: parse: %w", err)
	}
	best := ""
	var bestAt time.Time
	for _, e := range feed.Entries {
		tagName := strings.TrimPrefix(tagFromEntryID(e.ID), "v")
		// Only plain vMAJOR.MINOR.PATCH, so prereleases can never pass.
		if !isPureSemver(tagName) {
			continue
		}
		if best == "" || xsemver.Compare("v"+tagName, "v"+best) > 0 {
			best, bestAt = tagName, e.UpdatedAt
		}
	}
	if best == "" {
		return "", time.Time{}, errNoStableTag
	}
	return best, bestAt, nil
}

// Check builds an updater.Release for req from the feed plus deterministic
// release/download URLs. The checksum sidecar is fetched from the same
// deterministic URL; a missing or malformed entry fails the check.
func (s *AtomSource) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	tag, published, err := s.Latest(ctx)
	if err != nil {
		return nil, err
	}
	if !isNewerThan(tag, req.CurrentVersion) {
		return nil, nil
	}
	asset := assetNameFor(req.Platform, req.Arch)
	base := s.baseURL() + fmt.Sprintf("/%s/releases/download/v%s", s.Repository, tag)
	digest, err := s.fetchDigest(ctx, base+"/SHA256SUMS", asset)
	if err != nil {
		return nil, err
	}
	return &updater.Release{
		Version:     tag,
		Channel:     "stable",
		Name:        "v" + tag,
		PublishedAt: published,
		Artifact: updater.Artifact{
			Filename: asset,
			Filetype: strings.TrimPrefix(path.Ext(asset), "."),
			Platform: req.Platform,
			Arch:     req.Arch,
		},
		Verification: &updater.Verification{DigestAlgo: "sha256", Digest: digest},
		Metadata:     map[string]any{"atom.asset.url": base + "/" + asset},
	}, nil
}

// Download streams the asset URL stashed by Check.
func (s *AtomSource) Download(ctx context.Context, rel *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	urlStr, ok := rel.Metadata["atom.asset.url"].(string)
	if !ok || urlStr == "" {
		return errors.New("atom: release metadata missing asset URL")
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Minute}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("atom: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("atom: download: HTTP %d", resp.StatusCode)
	}
	total := rel.Artifact.Size
	if total == 0 && resp.ContentLength > 0 {
		total = resp.ContentLength
	}
	written := int64(0)
	buf := make([]byte, 64*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written, total)
			}
		}
		if rerr == io.EOF {
			return nil
		}
		if rerr != nil {
			return rerr
		}
	}
}

func (s *AtomSource) fetchDigest(ctx context.Context, sumsURL, asset string) ([]byte, error) {
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sumsURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("atom: checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("atom: checksum: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("atom: checksum: %w", err)
	}
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			return hexDigest(fields[0])
		}
	}
	return nil, fmt.Errorf("atom: checksum: checksum for %s not found", asset)
}

// assetNameFor is the single source of the CI asset naming convention:
// akproxy-<platform>-<arch><ext>, with darwin assets published as universal.
func assetNameFor(platform, arch string) string {
	assetArch := arch
	if platform == "darwin" && (arch == "arm64" || arch == "amd64") {
		assetArch = "universal"
	}
	ext := ".zip"
	if platform == "linux" {
		ext = ".tar.gz"
	}
	return fmt.Sprintf("akproxy-%s-%s%s", platform, assetArch, ext)
}

// isNewerThan mirrors the updater's semver.IsNewer (its semver package is
// internal, so the three lines are duplicated here): strictly newer, "v"
// prefix tolerant, prerelease-aware via x/mod/semver.
func isNewerThan(tag, current string) bool {
	tag, current = "v"+tag, "v"+current
	if tag == "v" {
		return false
	}
	if current == "v" {
		return true
	}
	return xsemver.Compare(tag, current) > 0
}

// isPureSemver accepts only plain vMAJOR.MINOR.PATCH digits, excluding
// prereleases and build metadata.
func isPureSemver(tag string) bool {
	parts := strings.Split(tag, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, r := range p {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	return true
}

func hexDigest(s string) ([]byte, error) {
	out, err := hex.DecodeString(s)
	if err != nil || len(out) != sha256.Size {
		return nil, errors.New("atom: checksum is not a sha256 hex digest")
	}
	return out, nil
}
