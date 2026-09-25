package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

const (
	metaClientID  = "1031625952748946"
	metaDeviceURL = "https://auth.meta.com/oidc/device/authorization/"
	metaTokenURL  = "https://auth.meta.com/oidc/device/token/"
	metaKeyURL    = "https://api.meta.ai/muse-code/key"
)

type metaDeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type MetaLoginCode struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type metaAuthenticator struct {
	emit      func(string, any)
	client    *http.Client
	deviceURL string
	tokenURL  string
	keyURL    string
}

type metaStorage struct{ metadata map[string]any }

type metaCancelStore struct{ coreauth.Store }

func (s metaCancelStore) Save(ctx context.Context, record *coreauth.Auth) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return s.Store.Save(ctx, record)
}

func (s *metaStorage) SetMetadata(metadata map[string]any) { s.metadata = metadata }

func (s *metaStorage) SaveTokenToFile(path string) error {
	raw, err := json.MarshalIndent(s.metadata, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".meta-token-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(append(raw, '\n')); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

var _ auth.Authenticator = metaAuthenticator{}

func (metaAuthenticator) Provider() string            { return "meta" }
func (metaAuthenticator) RefreshLead() *time.Duration { return nil }

func (a metaAuthenticator) Login(ctx context.Context, cfg *config.Config, _ *auth.LoginOptions) (*coreauth.Auth, error) {
	if cfg == nil {
		return nil, errors.New("Meta 配置不可用")
	}
	client := a.client
	if client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.ProxyURL != "" {
			proxy, err := url.Parse(cfg.ProxyURL)
			if err != nil {
				return nil, fmt.Errorf("Meta 出站代理无效: %w", err)
			}
			transport.Proxy = http.ProxyURL(proxy)
		}
		client = &http.Client{Timeout: 30 * time.Second, Transport: transport}
	}
	deviceURL, tokenURL, keyURL := a.deviceURL, a.tokenURL, a.keyURL
	if deviceURL == "" {
		deviceURL = metaDeviceURL
	}
	if tokenURL == "" {
		tokenURL = metaTokenURL
	}
	if keyURL == "" {
		keyURL = metaKeyURL
	}

	var device metaDeviceCode
	if err := metaPost(ctx, client, deviceURL, url.Values{"client_id": {metaClientID}}.Encode(), "application/x-www-form-urlencoded", "", &device, false); err != nil {
		return nil, fmt.Errorf("获取 Meta 验证码失败: %w", err)
	}
	verificationURL := strings.TrimSpace(device.VerificationURIComplete)
	if verificationURL == "" {
		verificationURL = strings.TrimSpace(device.VerificationURI)
	}
	if device.DeviceCode == "" || device.UserCode == "" || verificationURL == "" {
		return nil, errors.New("Meta 没有返回有效的验证码或验证链接")
	}
	parsedURL, err := url.Parse(verificationURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.User != nil || !metaVerificationHost(parsedURL.Hostname()) {
		return nil, errors.New("Meta 返回了无效的验证链接")
	}
	if a.emit != nil && ctx.Err() == nil {
		a.emit("login:meta-code", MetaLoginCode{Code: device.UserCode, URL: verificationURL})
	}
	interval := time.Duration(max(device.Interval, 1)) * time.Second
	limit := 15 * time.Minute
	if device.ExpiresIn > 0 {
		limit = min(limit, time.Duration(device.ExpiresIn)*time.Second)
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	for {
		if err := waitMetaPoll(ctx, interval); err != nil {
			return nil, err
		}
		var token struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int    `json:"expires_in"`
			Error       string `json:"error"`
		}
		form := url.Values{"client_id": {metaClientID}, "device_code": {device.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
		if err := metaPost(ctx, client, tokenURL, form.Encode(), "application/x-www-form-urlencoded", "", &token, true); err != nil {
			return nil, fmt.Errorf("Meta 授权失败: %w", err)
		}
		switch token.Error {
		case "authorization_pending":
			continue
		case "slow_down":
			interval += 5 * time.Second
			continue
		case "access_denied":
			return nil, errors.New("Meta 授权已拒绝")
		case "expired_token":
			return nil, errors.New("Meta 验证码已过期")
		case "":
			if token.AccessToken == "" {
				return nil, errors.New("Meta 没有返回登录令牌")
			}
		default:
			return nil, fmt.Errorf("Meta 授权失败: %s", token.Error)
		}
		record, err := metaMintAccount(ctx, client, keyURL, token.AccessToken, token.TokenType, token.ExpiresIn)
		if err == nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return record, err
	}
}

func metaVerificationHost(host string) bool {
	host = strings.ToLower(host)
	return host == "meta.com" || strings.HasSuffix(host, ".meta.com") || host == "meta.ai" || strings.HasSuffix(host, ".meta.ai")
}

func waitMetaPoll(ctx context.Context, interval time.Duration) error {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// metaPost never includes remote response bodies in errors: they may contain credentials.
func metaPost(ctx context.Context, client *http.Client, endpoint, body, contentType, bearer string, result any, acceptOAuthError bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "muse-code/1.0.2")
	req.Header.Set("Content-Type", contentType)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// OAuth polling uses non-2xx responses for pending and denied states.
		if acceptOAuthError && json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result) == nil {
			return nil
		}
		return fmt.Errorf("接口返回 %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(result); err != nil {
		return errors.New("接口响应无效")
	}
	return nil
}

func metaMintAccount(ctx context.Context, client *http.Client, endpoint, dcaToken, tokenType string, expiresIn int) (*coreauth.Auth, error) {
	var minted struct {
		APIKey     string `json:"api_key"`
		BaseURL    string `json:"base_url"`
		Email      string `json:"user_email"`
		Name       string `json:"user_full_name"`
		SubsActive *bool  `json:"is_subs_active"`
		RequirePay bool   `json:"require_payment"`
	}
	body, _ := json.Marshal(map[string]string{"dca_token": dcaToken})
	if err := metaPost(ctx, client, endpoint, string(body), "application/json", dcaToken, &minted, false); err != nil {
		return nil, fmt.Errorf("Meta 订阅密钥签发失败: %w", err)
	}
	if minted.RequirePay || minted.SubsActive != nil && !*minted.SubsActive {
		return nil, errors.New("Meta 账号没有可用的 Muse 订阅")
	}
	if minted.APIKey == "" {
		return nil, errors.New("Meta 没有返回可用的订阅密钥")
	}
	file := metaCredentialFileName(minted.Email, dcaToken)
	baseURL := strings.TrimSpace(minted.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.meta.ai/v1"
	}
	metadata := map[string]any{
		"type": "meta", "auth_kind": "oauth", "access_token": minted.APIKey,
		"api_key": minted.APIKey, "dca_token": dcaToken, "base_url": baseURL,
		"last_refresh": time.Now().UTC().Format(time.RFC3339),
	}
	if tokenType != "" {
		metadata["token_type"] = tokenType
	}
	if expiresIn > 0 {
		metadata["expires_in"] = expiresIn
		metadata["dca_expires_at"] = time.Now().Add(time.Duration(expiresIn) * time.Second).Unix()
		metadata["dca_expired"] = time.Now().Add(time.Duration(expiresIn) * time.Second).UTC().Format(time.RFC3339)
	}
	if minted.Email != "" {
		metadata["email"] = minted.Email
	}
	if minted.Name != "" {
		metadata["name"] = minted.Name
	}
	label := minted.Email
	if label == "" {
		label = "Meta"
	}
	return &coreauth.Auth{ID: file, FileName: file, Provider: "meta", Label: label, Metadata: metadata, Storage: &metaStorage{}}, nil
}

func metaCredentialFileName(email, token string) string {
	identity := strings.TrimSpace(email)
	if identity == "" {
		hash := sha256.Sum256([]byte(token))
		return "meta-" + hex.EncodeToString(hash[:8]) + ".json"
	}
	name := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, identity)
	if len(name) > 120 {
		name = name[:120]
	}
	hash := sha256.Sum256([]byte(identity))
	return "meta-" + name + "-" + hex.EncodeToString(hash[:8]) + ".json"
}
