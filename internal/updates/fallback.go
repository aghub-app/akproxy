package updates

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

// FallbackProvider tries the primary GitHub API provider first and, on
// rate-limit or transport failure, retries once against the releases.atom feed.
// Verification failures never fall back. See docs/adr/updater-fallback.md.
type FallbackProvider struct {
	primary *Provider
	atom    *AtomSource
}

func NewFallbackProvider(primary *Provider) *FallbackProvider {
	return &FallbackProvider{primary: primary, atom: &AtomSource{Repository: "aghub-app/akproxy"}}
}

func (p *FallbackProvider) Name() string { return "github" }

func (p *FallbackProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	rel, err := p.primary.Check(ctx, req)
	if err == nil {
		return rel, nil
	}
	if !shouldFallback(err) {
		return nil, err
	}
	rel, atomErr := p.atom.Check(ctx, req)
	if atomErr != nil {
		return nil, fmt.Errorf("%w; atom fallback: %w", err, atomErr)
	}
	if rel == nil {
		return nil, nil
	}
	if err := requireDigest(rel); err != nil {
		return nil, err
	}
	return rel, nil
}

func (p *FallbackProvider) Download(ctx context.Context, rel *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	if _, ok := rel.Metadata["atom.asset.url"]; ok {
		return p.atom.Download(ctx, rel, dst, onProgress)
	}
	return p.primary.Download(ctx, rel, dst, onProgress)
}

// shouldFallback limits the retry to shared-IP rate limiting, server errors,
// and transport failures. A 404 means the repository has no releases at all;
// the atom feed would be empty too.
func shouldFallback(err error) bool {
	msg := err.Error()
	if strings.Contains(msg, "403") || strings.Contains(msg, "429") || strings.Contains(msg, "rate limit") ||
		strings.Contains(msg, "HTTP 5") {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}
