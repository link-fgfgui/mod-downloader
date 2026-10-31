//go:build !server

package main

import (
	"log"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func main() {
	appService := NewApp()
	app := application.New(application.Options{
		Name:        "mod-downloader",
		Description: "Minecraft mod downloader and instance manager",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	appService.app = app

	gpuPolicy := application.WebviewGpuPolicyNever
	if configs.LoadHardwareAcceleration() {
		gpuPolicy = application.WebviewGpuPolicyAlways
	}

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "mod-downloader",
		Width:            1024,
		Height:           768,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Linux: application.LinuxWindow{
			WebviewGpuPolicy: gpuPolicy,
		},
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
