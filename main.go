package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func windowBackground(dark bool) application.RGBA {
	if dark {
		return application.NewRGB(15, 15, 15)
	}
	return application.NewRGB(255, 255, 255)
}

func main() {
	service := NewApp()
	app := application.New(application.Options{
		Name:     "akproxy",
		Services: []application.Service{application.NewService(service)},
		Assets:   application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:      application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	if err := configureUpdates(app); err != nil {
		log.Fatal(err)
	}
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                  "main",
		Title:                 "akproxy",
		Width:                 1120,
		Height:                760,
		DisableResize:         true,
		MaximiseButtonState:   application.ButtonDisabled,
		FullscreenButtonState: application.ButtonDisabled,
		Mac:                   application.MacWindow{CollectionBehavior: application.MacWindowCollectionBehaviorFullScreenNone},
		BackgroundColour:      windowBackground(app.Env.IsDarkMode()),
		URL:                   "/",
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) { app.Quit() })
	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(_ *application.ApplicationEvent) {
		window.SetBackgroundColour(windowBackground(app.Env.IsDarkMode()))
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		if version != "dev" {
			go func() {
				// The scheduler no-ops this when auto-check is off.
				scheduler.tick()
			}()
		}
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
