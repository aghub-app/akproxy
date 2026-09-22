package main

import (
	"errors"
	"time"

	"akproxy/internal/updates"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// Release builds inject the tag. Development builds never replace themselves.
var version = "dev"

// Build-time override for the local end-to-end fixture; production uses GitHub.
var updateAPIBase string

func configureUpdates(app *application.App) error {
	p, err := updates.NewProvider(updateAPIBase)
	if err != nil {
		return err
	}
	interval := time.Duration(0)
	if version != "dev" {
		interval = 24 * time.Hour
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{p},
		CheckInterval:  interval,
		// The built-in window opens as soon as a check starts, including the
		// startup check and the up-to-date result. The app window shows a dialog
		// only when there is something to act on.
		Window: updater.WindowNone,
	}); err != nil {
		return err
	}
	app.OnShutdown(app.Updater.StopPeriodicCheck)
	return nil
}

func (a *App) Version() string { return version }

func (a *App) CheckUpdates() error {
	if version == "dev" {
		return errors.New("开发版本不执行更新，请使用正式安装包")
	}
	u := application.Get().Updater
	switch u.State() {
	case updater.StateChecking, updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return errors.New("更新正在进行中")
	}
	return u.CheckAndInstall(a.ctx)
}

// ApplyUpdate restarts into a downloaded release. It does not restart the proxy.
func (a *App) ApplyUpdate() error {
	if version == "dev" {
		return errors.New("开发版本不执行更新，请使用正式安装包")
	}
	return application.Get().Updater.Restart(a.ctx)
}
