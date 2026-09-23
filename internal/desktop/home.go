package desktop

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
)

// HomeSnapshot contains client-facing setup data, never upstream secrets.
type HomeSnapshot struct {
	HasCredentials bool     `json:"hasCredentials"`
	ClientKeys     []string `json:"clientKeys"`
	Status         Status   `json:"status"`
}

func (r *Runtime) Home() (HomeSnapshot, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return HomeSnapshot{}, err
	}
	status, err := r.Status()
	if err != nil {
		return HomeSnapshot{}, err
	}
	out := HomeSnapshot{Status: status, ClientKeys: append([]string{}, cfg.APIKeys...)}
	for _, provider := range []string{"codex", "grok", "claude", "gemini"} {
		keys, err := ReadKeys(cfg, provider)
		if err != nil {
			return HomeSnapshot{}, err
		}
		for _, key := range keys {
			if strings.TrimSpace(key.APIKey) != "" {
				out.HasCredentials = true
			}
		}
	}
	for _, provider := range cfg.OpenAICompatibility {
		for _, key := range provider.APIKeyEntries {
			if strings.TrimSpace(key.APIKey) != "" {
				out.HasCredentials = true
			}
		}
	}
	store := auth.NewFileTokenStore()
	store.SetBaseDir(r.paths.Auth)
	records, err := store.List(context.Background())
	if err != nil {
		return HomeSnapshot{}, fmt.Errorf("读取账号失败: %w", err)
	}
	for _, record := range records {
		if record == nil {
			continue
		}
		for _, providers := range accountProviders {
			if providers[record.Provider] {
				out.HasCredentials = true
			}
		}
	}
	return out, nil
}

// HomeModels queries the actual client endpoint, using its current binding.
func (r *Runtime) HomeModels() ([]string, error) {
	home, err := r.Home()
	if err != nil {
		return nil, err
	}
	if !home.Status.Running {
		return nil, fmt.Errorf("服务尚未启动")
	}
	if len(home.ClientKeys) == 0 || home.ClientKeys[0] == "" {
		return nil, fmt.Errorf("没有客户端密钥")
	}
	req, err := http.NewRequest(http.MethodGet, home.Status.Address+"/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("服务地址无效")
	}
	req.Header.Set("Authorization", "Bearer "+home.ClientKeys[0])
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	defer transport.CloseIdleConnections()
	client := &http.Client{Timeout: 5 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法读取模型列表，请检查服务连接与 TLS 证书")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("读取模型列表失败（HTTP %d）", response.StatusCode)
	}
	const maxBody = 4 << 20
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil || len(body) > maxBody {
		return nil, fmt.Errorf("模型列表响应无法读取或过大")
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil || payload.Data == nil {
		return nil, fmt.Errorf("模型列表格式无效")
	}
	models := make([]string, 0, len(payload.Data))
	for _, model := range payload.Data {
		if strings.TrimSpace(model.ID) != "" {
			models = append(models, model.ID)
		}
	}
	slices.Sort(models)
	return slices.Compact(models), nil
}
