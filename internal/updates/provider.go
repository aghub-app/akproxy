package updates

import (
	"context"
	"crypto/sha256"
	"errors"
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
	if err := requireDigest(release); err != nil {
		return nil, err
	}
	return release, nil
}

func requireDigest(release *updater.Release) error {
	if release.Verification == nil || release.Verification.DigestAlgo != "sha256" || len(release.Verification.Digest) != sha256.Size {
		return ErrMissingChecksum
	}
	return nil
}

func MatchAsset(req updater.CheckRequest, assets []github.ReleaseAsset) int {
	name := assetNameFor(req.Platform, req.Arch)
	for i, asset := range assets {
		if asset.Name == name {
			return i
		}
	}
	return -1
}
