package desktop

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/auth"
	coreauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

func (r *Runtime) accountStore() *auth.FileTokenStore {
	store := auth.NewFileTokenStore()
	store.SetBaseDir(r.paths.Auth)
	return store
}

func (r *Runtime) accountRecords() ([]*coreauth.Auth, error) {
	records, err := r.accountStore().List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("读取账号失败: %w", err)
	}
	return records, nil
}

func (r *Runtime) accountsForPage(page string) ([]*coreauth.Auth, error) {
	providers, ok := accountProviders[page]
	if !ok {
		return []*coreauth.Auth{}, nil
	}
	records, err := r.accountRecords()
	if err != nil {
		return nil, err
	}
	out := make([]*coreauth.Auth, 0, len(records))
	for _, record := range records {
		if record != nil && providers[record.Provider] {
			out = append(out, record)
		}
	}
	return out, nil
}

// Accounts lists browser logins for a sidebar page.
func (r *Runtime) Accounts(page string) ([]Account, error) {
	records, err := r.accountsForPage(page)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(records))
	r.mu.Lock()
	manager := r.coreAuth
	if !r.running {
		manager = nil
	}
	r.mu.Unlock()
	for _, record := range records {
		account := Account{
			ID:       record.ID,
			Provider: record.Provider,
			Label:    accountLabel(record),
		}
		if manager != nil {
			if current, ok := manager.GetByID(record.ID); ok {
				account.NeedsReauthorization = needsReauthorization(current)
			}
		}
		out = append(out, account)
	}
	return out, nil
}

// AccountUsage reads session-limit windows for the logins on one sidebar page.
// Providers without a vendor usage endpoint are omitted. A failed read stays on
// that account and does not fail the rest.
func (r *Runtime) AccountUsage(page string) ([]AccountUsage, error) {
	records, err := r.accountsForPage(page)
	if err != nil {
		return nil, err
	}
	jobs := make([]*coreauth.Auth, 0)
	for _, record := range records {
		if !supportsSessionLimit(record.Provider) {
			continue
		}
		jobs = append(jobs, record)
	}
	out := make([]AccountUsage, len(jobs))
	var wg sync.WaitGroup
	for i, record := range jobs {
		wg.Add(1)
		go func(i int, record *coreauth.Auth) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), limitsTimeout)
			defer cancel()
			item, err := sessionLimits(ctx, record.Provider, record.Metadata)
			if item.Windows == nil {
				item.Windows = []UsageWindow{}
			}
			item.ID = record.ID
			if err != nil {
				item.Error = err.Error()
			}
			out[i] = item
		}(i, record)
	}
	wg.Wait()
	return out, nil
}

// DeleteAccount removes one browser-login file.
func (r *Runtime) DeleteAccount(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("没有指定账号")
	}
	if !filepath.IsLocal(id) {
		return fmt.Errorf("账号无效")
	}
	records, err := r.accountRecords()
	if err != nil {
		return err
	}
	found := false
	for _, record := range records {
		if record != nil && record.ID == id {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("账号不存在")
	}
	if err := os.Remove(filepath.Join(r.paths.Auth, id)); err != nil {
		return fmt.Errorf("删除账号失败: %w", err)
	}
	return nil
}

var accountProviders = map[string]map[string]bool{
	"codex":  {"codex": true},
	"grok":   {"xai": true},
	"claude": {"claude": true},
	"gemini": {"antigravity": true},
	"kimi":   {"kimi": true, "kimi-ai": true, "kimi.ai": true},
	"devin":  {"devin": true},
	"meta":   {"meta": true},
}

func accountLabel(record *coreauth.Auth) string {
	if record == nil {
		return ""
	}
	if record.Metadata != nil {
		if email, ok := record.Metadata["email"].(string); ok && strings.TrimSpace(email) != "" {
			return email
		}
	}
	if strings.TrimSpace(record.Label) != "" {
		return record.Label
	}
	return record.ID
}
