package main

import (
	"context"
	"fmt"

	"akproxy/internal/desktop"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App is the Wails binding surface. It delegates to the desktop runtime.
type App struct {
	ctx       context.Context
	desktop   *desktop.Runtime
	aboutMenu *application.MenuItem
}

// NewApp creates the application before the window starts.
func NewApp() *App {
	return &App{}
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.ctx = ctx
	service, err := desktop.NewRuntime(func(event string, data any) {
		application.Get().Event.Emit(event, data)
	})
	if err != nil {
		return err
	}
	a.desktop = service
	return nil
}

func (a *App) ServiceShutdown() error {
	if a.desktop != nil {
		a.desktop.Shutdown()
	}
	return nil
}

func (a *App) ready() error {
	if a.desktop == nil {
		return errNotReady
	}
	return nil
}

// SetPresentation keeps the native menu and window background in sync with the renderer.
func (a *App) SetPresentation(language string, dark bool) error {
	if language != "zh-CN" && language != "en" {
		return fmt.Errorf("界面语言无效")
	}
	if a.aboutMenu != nil {
		label := "关于 akproxy"
		if language == "en" {
			label = "About akproxy"
		}
		a.aboutMenu.SetLabel(label)
	}
	if window, ok := application.Get().Window.GetByName("main"); ok {
		window.SetBackgroundColour(windowBackground(dark))
	}
	return nil
}

// Status returns the navbar button and listen address.
func (a *App) Status() (desktop.Status, error) {
	if err := a.ready(); err != nil {
		return desktop.Status{}, err
	}
	return a.desktop.Status()
}

// ServiceSettings returns the 服务 page.
func (a *App) ServiceSettings() (desktop.ServiceSettings, error) {
	if err := a.ready(); err != nil {
		return desktop.ServiceSettings{}, err
	}
	return a.desktop.ServiceSettings()
}

// SaveService writes the 服务 page.
func (a *App) SaveService(in desktop.ServiceSettings) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.SaveService(in)
}

// ProviderKeys returns API keys for codex, grok, claude, or gemini.
func (a *App) ProviderKeys(provider string) ([]desktop.KeyDraft, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.desktop.ProviderKeys(provider)
}

// SaveProviderKeys writes one native provider's API keys.
func (a *App) SaveProviderKeys(provider string, drafts []desktop.KeyDraft) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.SaveProviderKeys(provider, drafts)
}

// KimiProviders returns the Kimi page.
func (a *App) KimiProviders() ([]desktop.OpenAIDraft, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.desktop.KimiProviders()
}

// SaveKimi writes the Kimi page.
func (a *App) SaveKimi(drafts []desktop.OpenAIDraft) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.SaveKimi(drafts)
}

// Start binds the saved config.
func (a *App) Start() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.Start()
}

// Stop cancels the embedded service.
func (a *App) Stop() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.Stop()
}

// Restart stops the service and starts the saved config.
func (a *App) Restart() error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.Restart()
}

// Login opens the system browser for one upstream.
func (a *App) Login(provider string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.Login(provider)
}

// ReauthorizeAccount renews the selected saved browser login.
func (a *App) ReauthorizeAccount(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.ReauthorizeAccount(id)
}

// CancelLogin stops the login that is waiting.
func (a *App) CancelLogin() {
	if a.desktop != nil {
		a.desktop.CancelLogin()
	}
}

// Accounts lists browser logins for a sidebar page.
func (a *App) Accounts(page string) ([]desktop.Account, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.desktop.Accounts(page)
}

// AccountUsage reads session-limit windows for one sidebar page.
func (a *App) AccountUsage(page string) ([]desktop.AccountUsage, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.desktop.AccountUsage(page)
}

// DeleteAccount removes one browser-login file.
func (a *App) DeleteAccount(id string) error {
	if err := a.ready(); err != nil {
		return err
	}
	return a.desktop.DeleteAccount(id)
}

// Home returns the setup state and client-facing connection details.
func (a *App) Home() (desktop.HomeSnapshot, error) {
	if err := a.ready(); err != nil {
		return desktop.HomeSnapshot{}, err
	}
	return a.desktop.Home()
}

// HomeModels reads models from the running local proxy.
func (a *App) HomeModels() ([]string, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	return a.desktop.HomeModels()
}
