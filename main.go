package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:    "silo",
		Width:    720,
		Height:   520,
		MinWidth: 600,
		MinHeight: 400,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// #1a1d23 — warm dark grey terminal background
		BackgroundColour: &options.RGBA{R: 26, G: 29, B: 35, A: 1},
		// Close button hides the window instead of quitting — silo keeps
		// running so an active seal survives the user pressing ⌘W. Re-open
		// from the dock/taskbar icon to bring the window back.
		// Full status-bar tray icon is a Wails v3 migration item (Phase 10).
		HideWindowOnClose: true,
		OnStartup:         app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
