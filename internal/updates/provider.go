package updates

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

var ErrMissingChecksum = errors.New("更新缺少有效的 SHA-256 校验信息")

// Provider requires the checksum that the official GitHub provider treats as optional.
type Provider struct{ *github.Provider }

func NewProvider(baseURL string) (*Provider, error) {
	p, err := github.New(github.Config{
		Repository: "aghub-app/akproxy", ChecksumAsset: "SHA256SUMS",
		BaseURL:      baseURL,
		AssetMatcher: MatchAsset,
		HTTPClient:   &http.Client{Timeout: 30 * time.Minute},
	})
	if err != nil {
		return nil, err
	}
	return &Provider{p}, nil
}

func (p *Provider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	release, err := p.Provider.Check(ctx, req)
	if err != nil || release == nil {
		return release, err
	}
	if release.Channel != "stable" {
		return nil, nil
	}
	if release.Verification == nil || release.Verification.DigestAlgo != "sha256" || len(release.Verification.Digest) != sha256.Size {
		return nil, ErrMissingChecksum
	}
	return release, nil
}

func MatchAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	arch := req.Arch
	if req.Platform == "darwin" && (arch == "arm64" || arch == "amd64") {
		arch = "universal"
	}
	ext := ".zip"
	if req.Platform == "linux" {
		ext = ".tar.gz"
	}
	name := fmt.Sprintf("akproxy-%s-%s%s", req.Platform, arch, ext)
	for i, asset := range assets {
		if asset.Name == name {
			return i
		}
	}
	return -1
}
