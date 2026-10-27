package main

import (
	"embed"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Wails uses WebviewGpuPolicyNever when Linux options are nil, so the
	// persisted preference must be applied explicitly. The policy is consumed
	// when the webview is created, which is why this is read before wails.Run.
	gpuPolicy := linux.WebviewGpuPolicyNever
	if configs.LoadHardwareAcceleration() {
		gpuPolicy = linux.WebviewGpuPolicyAlways
	}

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "mod-downloader",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		Linux: &linux.Options{
			WebviewGpuPolicy: gpuPolicy,
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
