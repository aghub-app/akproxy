package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

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
	prefs := scheduler.Status().Prefs
	menu, about := aboutToSettingsMenu(app, prefs.Language == "en")
	service.aboutMenu = about
	app.Menu.Set(menu)
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:                  "main",
		Title:                 "akproxy",
		Width:                 1120,
		Height:                760,
		DisableResize:         true,
		MaximiseButtonState:   application.ButtonDisabled,
		FullscreenButtonState: application.ButtonDisabled,
		Mac:                   application.MacWindow{CollectionBehavior: application.MacWindowCollectionBehaviorFullScreenNone},
		BackgroundColour:      windowBackground(darkForTheme(prefs.Theme, app.Env.IsDarkMode())),
		URL:                   "/",
	})
	window.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) { app.Quit() })
	app.Event.OnApplicationEvent(events.Common.ThemeChanged, func(_ *application.ApplicationEvent) {
		window.SetBackgroundColour(windowBackground(darkForTheme(scheduler.Status().Prefs.Theme, app.Env.IsDarkMode())))
	})
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		// The Dock caches bundle icons aggressively; set the icon at runtime so a
		// new build shows up without flushing the icon cache.
		icon := appIcon
		if app.Env.Info().OS == "darwin" {
			icon = dockIcon(appIcon)
		}
		app.SetIcon(icon)
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

func darkForTheme(theme string, systemDark bool) bool {
	return theme == "dark" || theme == "system" && systemDark
}

// aboutToSettingsMenu routes About to the in-app tab.
func aboutToSettingsMenu(app *application.App, english bool) (*application.Menu, *application.MenuItem) {
	menu := application.NewMenu()
	appMenu := menu.AddSubmenu("akproxy")
	label := "关于 akproxy"
	if english {
		label = "About akproxy"
	}
	about := appMenu.Add(label)
	about.OnClick(func(*application.Context) {
		app.Event.Emit("app:about")
	})
	appMenu.AddSeparator()
	appMenu.AddRole(application.ServicesMenu)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Hide)
	appMenu.AddRole(application.HideOthers)
	appMenu.AddRole(application.UnHide)
	appMenu.AddSeparator()
	appMenu.AddRole(application.Quit)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.WindowMenu)
	menu.AddRole(application.HelpMenu)
	return menu, about
}
