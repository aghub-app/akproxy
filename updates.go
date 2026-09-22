package main

import (
	"errors"
	"runtime"

	"akproxy/internal/desktop"
	"akproxy/internal/updates"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

// Release builds inject the tag. Development builds never replace themselves.
var version = "dev"

// Build-time override for the local end-to-end fixture; production uses GitHub.
var updateAPIBase string

var scheduler *updateScheduler

func prefsRoot() string {
	paths, err := desktop.Resolve()
	if err != nil {
		return ""
	}
	return paths.Root
}

func platformLabel() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS " + runtime.GOARCH
	case "windows":
		return "Windows " + runtime.GOARCH
	case "linux":
		return "Linux " + runtime.GOARCH
	default:
		return runtime.GOOS + " " + runtime.GOARCH
	}
}

func configureUpdates(app *application.App) error {
	p, err := updates.NewProvider(updateAPIBase)
	if err != nil {
		return err
	}
	if err := app.Updater.Init(updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{p},
		// The app runs its own timer so the 关于 preferences can toggle and
		// retune checks at runtime; the updater's built-in loop cannot be
		// restarted after StopPeriodicCheck.
		CheckInterval: 0,
		// The built-in window opens as soon as a check starts, including the
		// startup check and the up-to-date result. The app window shows a dialog
		// only when there is something to act on.
		Window: updater.WindowNone,
	}); err != nil {
		return err
	}
	scheduler = newUpdateScheduler(func(event string, data any) {
		app.Event.Emit(event, data)
	})
	scheduler.Start()
	app.OnShutdown(func() {
		scheduler.Stop()
		app.Updater.StopPeriodicCheck()
	})
	return nil
}

func (a *App) Version() string { return version }

// UpdatePrefStatus returns the 关于 tab's preferences and last check result.
func (a *App) UpdatePrefStatus() (UpdatePrefsStatus, error) {
	return scheduler.Status(), nil
}

// SaveUpdatePrefs applies the 关于 tab's update preferences immediately.
func (a *App) SaveUpdatePrefs(in desktop.AppPrefs) (UpdatePrefsStatus, error) {
	return scheduler.Save(in)
}

// DownloadPendingUpdate downloads the release found by a find-only automatic
// check, after the user picks 立即下载 in the new-release dialog.
func (a *App) DownloadPendingUpdate() error {
	if version == "dev" {
		return errors.New("开发版本不执行更新，请使用正式安装包")
	}
	return scheduler.DownloadPendingUpdate()
}

func (a *App) CheckUpdates() error {
	if version == "dev" {
		return errors.New("开发版本不执行更新，请使用正式安装包")
	}
	u := application.Get().Updater
	switch u.State() {
	case updater.StateChecking, updater.StateDownloading, updater.StateVerifying, updater.StateInstalling:
		return errors.New("更新正在进行中")
	}
	err := u.CheckAndInstall(a.ctx)
	scheduler.RecordManualCheck(err)
	return err
}

// ApplyUpdate restarts into a downloaded release. It does not restart the proxy.
func (a *App) ApplyUpdate() error {
	if version == "dev" {
		return errors.New("开发版本不执行更新，请使用正式安装包")
	}
	return application.Get().Updater.Restart(a.ctx)
}
