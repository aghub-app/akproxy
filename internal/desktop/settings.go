package desktop

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/config"
)

const (
	listenLocal  = "local"
	listenAll    = "all"
	listenCustom = "custom"

	strategyRoundRobin         = "round-robin"
	strategyWeightedRoundRobin = "weighted-round-robin"
	strategyFillFirst          = "fill-first"

	kimiName   = "kimi"
	kimiAIName = "kimi-ai"

	defaultPort = 8317
)

// ReadService copies the 服务 page from a loaded config.
func ReadService(cfg *config.Config) ServiceSettings {
	if cfg == nil {
		return ServiceSettings{ListenMode: listenLocal, Port: defaultPort, RoutingStrategy: strategyRoundRobin}
	}
	mode, custom := listenModeFromHost(cfg.Host)
	strategy := cfg.Routing.Strategy
	if strategy == "" {
		strategy = strategyRoundRobin
	}
	port := cfg.Port
	if port == 0 {
		port = defaultPort
	}
	return ServiceSettings{
		ListenMode:            mode,
		CustomHost:            custom,
		Port:                  port,
		ClientAPIKeys:         append([]string(nil), cfg.APIKeys...),
		ProxyURL:              cfg.ProxyURL,
		RoutingStrategy:       strategy,
		Debug:                 cfg.Debug,
	}
}

// ApplyService writes the 服务 page onto cfg and pins the account directory.
func ApplyService(cfg *config.Config, in ServiceSettings, authDir string) error {
	if cfg == nil {
		return fmt.Errorf("没有配置")
	}
	host, err := hostFromSettings(in)
	if err != nil {
		return err
	}
	if in.Port < 1 || in.Port > 65535 {
		return fmt.Errorf("端口要在 1 到 65535 之间")
	}
	if !validStrategy(in.RoutingStrategy) {
		return fmt.Errorf("路由策略无效")
	}
	keys, err := clientKeys(in.ClientAPIKeys)
	if err != nil {
		return err
	}
	cfg.Host = host
	cfg.Port = in.Port
	cfg.APIKeys = keys
	cfg.ProxyURL = strings.TrimSpace(in.ProxyURL)
	cfg.Routing.Strategy = in.RoutingStrategy
	cfg.Debug = in.Debug
	cfg.AuthDir = authDir
	return nil
}

// ReadKeys returns the API-key rows for codex, grok, claude, or gemini.
func ReadKeys(cfg *config.Config, provider string) ([]KeyDraft, error) {
	if cfg == nil {
		return nil, fmt.Errorf("没有配置")
	}
	switch provider {
	case "codex":
		return codexDrafts(cfg.CodexKey), nil
	case "grok":
		return codexDrafts(cfg.XAIKey), nil
	case "claude":
		return claudeDrafts(cfg.ClaudeKey), nil
	case "gemini":
		return geminiDrafts(cfg.GeminiKey), nil
	default:
		return nil, fmt.Errorf("这个页面没有 API key 列表")
	}
}

// ApplyKeys replaces one native provider's API keys and keeps fields the form does not edit.
func ApplyKeys(cfg *config.Config, provider string, drafts []KeyDraft, authDir string) error {
	if cfg == nil {
		return fmt.Errorf("没有配置")
	}
	cleaned, err := cleanKeyDrafts(drafts)
	if err != nil {
		return err
	}
	switch provider {
	case "codex":
		cfg.CodexKey = mergeCodex(cfg.CodexKey, cleaned)
	case "grok":
		cfg.XAIKey = mergeCodex(cfg.XAIKey, cleaned)
	case "claude":
		cfg.ClaudeKey = mergeClaude(cfg.ClaudeKey, cleaned)
	case "gemini":
		cfg.GeminiKey = mergeGemini(cfg.GeminiKey, cleaned)
	default:
		return fmt.Errorf("这个页面没有 API key 列表")
	}
	cfg.AuthDir = authDir
	return nil
}

// ReadKimi returns the OpenAI-compatible entries owned by the Kimi page.
func ReadKimi(cfg *config.Config) []OpenAIDraft {
	if cfg == nil {
		return nil
	}
	return filterOpenAI(cfg.OpenAICompatibility, true)
}

// ApplyKimi replaces only the kimi and kimi-ai entries.
func ApplyKimi(cfg *config.Config, drafts []OpenAIDraft, authDir string) error {
	if cfg == nil {
		return fmt.Errorf("没有配置")
	}
	cleaned, err := cleanOpenAI(drafts)
	if err != nil {
		return err
	}
	cfg.OpenAICompatibility = mergeOwnedOpenAI(cfg.OpenAICompatibility, cleaned, true)
	cfg.AuthDir = authDir
	return nil
}

// CheckRunnable reports why a saved config must not be started.
func CheckRunnable(cfg *config.Config, authDir string) error {
	if err := CheckSavedFile(cfg, authDir); err != nil {
		return err
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("端口要在 1 到 65535 之间")
	}
	if cfg.TLS.Enable && (strings.TrimSpace(cfg.TLS.Cert) == "" || strings.TrimSpace(cfg.TLS.Key) == "") {
		return fmt.Errorf("打开 TLS 时要填写证书和私钥路径")
	}
	return nil
}

// CheckSavedFile rejects a config that must not be written or kept.
func CheckSavedFile(cfg *config.Config, authDir string) error {
	if cfg == nil {
		return fmt.Errorf("没有配置")
	}
	if err := atLeastOneClientKey(cfg.APIKeys); err != nil {
		return err
	}
	if filepathClean(cfg.AuthDir) != filepathClean(authDir) {
		return fmt.Errorf("账号目录必须留在这个应用里")
	}
	return nil
}

func atLeastOneClientKey(keys []string) error {
	for _, key := range keys {
		if strings.TrimSpace(key) != "" {
			return nil
		}
	}
	return fmt.Errorf("至少保留一把客户端密钥")
}

// SavedListen reads the listen tuple that a start would bind.
func SavedListen(cfg *config.Config) Listen {
	if cfg == nil {
		return Listen{Port: defaultPort}
	}
	return Listen{
		Host:       cfg.Host,
		Port:       cfg.Port,
		TLSEnabled: cfg.TLS.Enable,
		TLSCert:    cfg.TLS.Cert,
		TLSKey:     cfg.TLS.Key,
	}
}

// ControlAction is the single navbar verb.
func ControlAction(running bool, bound, saved Listen) string {
	if !running {
		return "start"
	}
	if bound != saved {
		return "restart"
	}
	return "stop"
}

// ClientURL is the address a local client should call.
func ClientURL(listen Listen) (address string, allInterfaces bool) {
	scheme := "http"
	if listen.TLSEnabled {
		scheme = "https"
	}
	host := strings.TrimSpace(listen.Host)
	allInterfaces = host == "" || host == "0.0.0.0" || host == "::"
	if allInterfaces {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, host, listen.Port), allInterfaces
}

func listenModeFromHost(host string) (mode, custom string) {
	switch strings.TrimSpace(host) {
	case "", "0.0.0.0", "::":
		return listenAll, ""
	case "127.0.0.1", "localhost":
		return listenLocal, ""
	default:
		return listenCustom, strings.TrimSpace(host)
	}
}

func hostFromSettings(in ServiceSettings) (string, error) {
	switch in.ListenMode {
	case listenLocal:
		return "127.0.0.1", nil
	case listenAll:
		return "", nil
	case listenCustom:
		host := strings.TrimSpace(in.CustomHost)
		if host == "" {
			return "", fmt.Errorf("要填写自定义监听地址")
		}
		if strings.Contains(host, "://") || strings.ContainsAny(host, " /\\") {
			return "", fmt.Errorf("监听地址无效")
		}
		return host, nil
	default:
		return "", fmt.Errorf("监听范围无效")
	}
}

func validStrategy(value string) bool {
	switch value {
	case strategyRoundRobin, strategyWeightedRoundRobin, strategyFillFirst:
		return true
	default:
		return false
	}
}

func clientKeys(keys []string) ([]string, error) {
	out := make([]string, 0, len(keys))
	seen := map[string]struct{}{}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("客户端密钥不能重复")
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("至少保留一把客户端密钥")
	}
	return out, nil
}

func cleanKeyDrafts(drafts []KeyDraft) ([]KeyDraft, error) {
	out := make([]KeyDraft, 0, len(drafts))
	seen := map[string]struct{}{}
	for _, draft := range drafts {
		draft.APIKey = strings.TrimSpace(draft.APIKey)
		draft.BaseURL = strings.TrimSpace(draft.BaseURL)
		draft.Prefix = strings.TrimSpace(draft.Prefix)
		if draft.APIKey == "" && draft.BaseURL == "" && draft.Prefix == "" {
			continue
		}
		if draft.APIKey == "" {
			return nil, fmt.Errorf("上游密钥不能为空")
		}
		if _, ok := seen[draft.APIKey]; ok {
			return nil, fmt.Errorf("上游密钥不能重复")
		}
		seen[draft.APIKey] = struct{}{}
		out = append(out, draft)
	}
	return out, nil
}

func codexDrafts(keys []config.CodexKey) []KeyDraft {
	out := make([]KeyDraft, 0, len(keys))
	for _, key := range keys {
		out = append(out, KeyDraft{APIKey: key.APIKey, BaseURL: key.BaseURL, Prefix: key.Prefix})
	}
	return out
}

func claudeDrafts(keys []config.ClaudeKey) []KeyDraft {
	out := make([]KeyDraft, 0, len(keys))
	for _, key := range keys {
		out = append(out, KeyDraft{APIKey: key.APIKey, BaseURL: key.BaseURL, Prefix: key.Prefix})
	}
	return out
}

func geminiDrafts(keys []config.GeminiKey) []KeyDraft {
	out := make([]KeyDraft, 0, len(keys))
	for _, key := range keys {
		out = append(out, KeyDraft{APIKey: key.APIKey, BaseURL: key.BaseURL, Prefix: key.Prefix})
	}
	return out
}

func mergeCodex(existing []config.CodexKey, drafts []KeyDraft) []config.CodexKey {
	by := map[string]config.CodexKey{}
	for _, item := range existing {
		by[item.APIKey] = item
	}
	out := make([]config.CodexKey, 0, len(drafts))
	for _, draft := range drafts {
		item := by[draft.APIKey]
		item.APIKey = draft.APIKey
		item.BaseURL = draft.BaseURL
		item.Prefix = draft.Prefix
		out = append(out, item)
	}
	return out
}

func mergeClaude(existing []config.ClaudeKey, drafts []KeyDraft) []config.ClaudeKey {
	by := map[string]config.ClaudeKey{}
	for _, item := range existing {
		by[item.APIKey] = item
	}
	out := make([]config.ClaudeKey, 0, len(drafts))
	for _, draft := range drafts {
		item := by[draft.APIKey]
		item.APIKey = draft.APIKey
		item.BaseURL = draft.BaseURL
		item.Prefix = draft.Prefix
		out = append(out, item)
	}
	return out
}

func mergeGemini(existing []config.GeminiKey, drafts []KeyDraft) []config.GeminiKey {
	by := map[string]config.GeminiKey{}
	for _, item := range existing {
		by[item.APIKey] = item
	}
	out := make([]config.GeminiKey, 0, len(drafts))
	for _, draft := range drafts {
		item := by[draft.APIKey]
		item.APIKey = draft.APIKey
		item.BaseURL = draft.BaseURL
		item.Prefix = draft.Prefix
		out = append(out, item)
	}
	return out
}

func kimiOwned(name string) bool {
	return name == kimiName || name == kimiAIName
}

func filterOpenAI(items []config.OpenAICompatibility, owned bool) []OpenAIDraft {
	out := make([]OpenAIDraft, 0)
	for _, item := range items {
		if kimiOwned(item.Name) != owned {
			continue
		}
		out = append(out, openAIToDraft(item))
	}
	return out
}

func openAIToDraft(item config.OpenAICompatibility) OpenAIDraft {
	keys := make([]string, 0, len(item.APIKeyEntries))
	for _, entry := range item.APIKeyEntries {
		keys = append(keys, entry.APIKey)
	}
	models := make([]ModelDraft, 0, len(item.Models))
	for _, model := range item.Models {
		models = append(models, ModelDraft{Name: model.Name, Alias: model.Alias})
	}
	return OpenAIDraft{Name: item.Name, BaseURL: item.BaseURL, APIKeys: keys, Models: models}
}

func cleanOpenAI(drafts []OpenAIDraft) ([]OpenAIDraft, error) {
	out := make([]OpenAIDraft, 0, len(drafts))
	seen := map[string]struct{}{}
	for _, draft := range drafts {
		draft.Name = strings.TrimSpace(draft.Name)
		draft.BaseURL = strings.TrimSpace(draft.BaseURL)
		keys := nonEmpty(draft.APIKeys)
		models := make([]ModelDraft, 0, len(draft.Models))
		for _, model := range draft.Models {
			model.Name = strings.TrimSpace(model.Name)
			model.Alias = strings.TrimSpace(model.Alias)
			if model.Name == "" && model.Alias == "" {
				continue
			}
			if model.Name == "" {
				return nil, fmt.Errorf("模型名不能为空")
			}
			if model.Alias == "" {
				model.Alias = model.Name
			}
			models = append(models, model)
		}
		if draft.Name == "" && draft.BaseURL == "" && len(keys) == 0 && len(models) == 0 {
			continue
		}
		if draft.Name == "" || draft.BaseURL == "" || len(keys) == 0 || len(models) == 0 {
			return nil, fmt.Errorf("OpenAI 兼容上游要有名称、地址、密钥和至少一个模型")
		}
		if !kimiOwned(draft.Name) {
			return nil, fmt.Errorf("Kimi 页只能保存 kimi 和 kimi-ai")
		}
		if _, ok := seen[draft.Name]; ok {
			return nil, fmt.Errorf("上游名称不能重复")
		}
		seen[draft.Name] = struct{}{}
		draft.APIKeys = keys
		draft.Models = models
		out = append(out, draft)
	}
	return out, nil
}

func mergeOwnedOpenAI(existing []config.OpenAICompatibility, drafts []OpenAIDraft, owned bool) []config.OpenAICompatibility {
	by := map[string]config.OpenAICompatibility{}
	kept := make([]config.OpenAICompatibility, 0, len(existing))
	for _, item := range existing {
		by[item.Name] = item
		if kimiOwned(item.Name) != owned {
			kept = append(kept, item)
		}
	}
	for _, draft := range drafts {
		item := by[draft.Name]
		item.Name = draft.Name
		item.BaseURL = draft.BaseURL
		item.APIKeyEntries = mergeOpenAIKeys(item.APIKeyEntries, draft.APIKeys)
		item.Models = mergeModels(item.Models, draft.Models)
		kept = append(kept, item)
	}
	return kept
}

func mergeOpenAIKeys(existing []config.OpenAICompatibilityAPIKey, keys []string) []config.OpenAICompatibilityAPIKey {
	by := map[string]config.OpenAICompatibilityAPIKey{}
	for _, item := range existing {
		by[item.APIKey] = item
	}
	out := make([]config.OpenAICompatibilityAPIKey, 0, len(keys))
	for _, key := range keys {
		item := by[key]
		item.APIKey = key
		out = append(out, item)
	}
	return out
}

func mergeModels(existing []config.OpenAICompatibilityModel, drafts []ModelDraft) []config.OpenAICompatibilityModel {
	by := map[string]config.OpenAICompatibilityModel{}
	for _, item := range existing {
		by[item.Name+"\x00"+item.Alias] = item
	}
	out := make([]config.OpenAICompatibilityModel, 0, len(drafts))
	for _, draft := range drafts {
		item := by[draft.Name+"\x00"+draft.Alias]
		item.Name = draft.Name
		item.Alias = draft.Alias
		out = append(out, item)
	}
	return out
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func filepathClean(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abs
}
