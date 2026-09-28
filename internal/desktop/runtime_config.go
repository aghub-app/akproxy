package desktop

import "github.com/router-for-me/CLIProxyAPI/v7/sdk/config"

// ServiceSettings returns the 服务 page.
func (r *Runtime) ServiceSettings() (ServiceSettings, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return ServiceSettings{}, err
	}
	return ReadService(cfg), nil
}

// SaveService writes the 服务 page.
func (r *Runtime) SaveService(in ServiceSettings) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyService(cfg, in, r.paths.Auth)
	})
}

// ProviderKeys returns API keys for codex, grok, claude, or gemini.
func (r *Runtime) ProviderKeys(provider string) ([]KeyDraft, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	return ReadKeys(cfg, provider)
}

// SaveProviderKeys writes one native provider's API keys.
func (r *Runtime) SaveProviderKeys(provider string, drafts []KeyDraft) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyKeys(cfg, provider, drafts, r.paths.Auth)
	})
}

// KimiProviders returns the Kimi page.
func (r *Runtime) KimiProviders() ([]OpenAIDraft, error) {
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		return nil, err
	}
	return ReadKimi(cfg), nil
}

// SaveKimi writes the Kimi page.
func (r *Runtime) SaveKimi(drafts []OpenAIDraft) error {
	return r.edit(func(cfg *config.Config) error {
		return ApplyKimi(cfg, drafts, r.paths.Auth)
	})
}

func (r *Runtime) edit(apply func(*config.Config) error) error {
	r.editMu.Lock()
	cfg, err := loadConfig(r.paths.Config)
	if err != nil {
		r.editMu.Unlock()
		return err
	}
	if err := apply(cfg); err != nil {
		r.editMu.Unlock()
		return err
	}
	if err := atLeastOneClientKey(cfg.APIKeys); err != nil {
		r.editMu.Unlock()
		return err
	}
	if err := saveConfig(r.paths.Config, cfg); err != nil {
		r.editMu.Unlock()
		return err
	}
	r.editMu.Unlock()
	r.emit("server:status", mustStatus(r))
	return nil
}
