// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !server

// Package main is the Wails desktop entry point for the mod downloader.
package main

import (
	"errors"
	"log"

	"github.com/link-fgfgui/mod-downloader/core/configs"
	"github.com/link-fgfgui/mod-downloader/core/instance"
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

	// Wails v3 defaults WebviewGpuPolicy to Always (the enum's zero value), so
	// only opt out explicitly. A missing or unreadable config keeps GPU
	// rendering enabled because DefaultHardwareAcceleration is true.
	gpuPolicy := application.WebviewGpuPolicyAlways
	if !configs.LoadHardwareAcceleration() {
		gpuPolicy = application.WebviewGpuPolicyNever
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
		if errors.Is(err, instance.ErrAlreadyRunning) {
			// A second launch exits quietly; the first instance keeps running.
			log.Print(err)
			return
		}
		log.Fatal(err)
	}
}
