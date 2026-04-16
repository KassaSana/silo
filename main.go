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
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
