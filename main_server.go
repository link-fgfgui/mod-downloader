//go:build server

package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const defaultServerPort = 9245

func main() {
	port := defaultServerPort
	if raw := os.Getenv("MOD_DOWNLOADER_SERVER_PORT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			log.Fatalf("invalid MOD_DOWNLOADER_SERVER_PORT %q", raw)
		}
		port = parsed
	}

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
		Server: application.ServerOptions{
			Host: "127.0.0.1",
			Port: port,
		},
	})
	appService.app = app
	// Server mode has no window; application.Quit alone does not stop the
	// server loop there. Give the graceful path a moment, then force exit.
	appService.quitFunc = func() {
		go func() {
			time.Sleep(2 * time.Second)
			os.Exit(0)
		}()
		if appService.server != nil {
			appService.server.Stop()
		}
		if appService.core != nil {
			appService.core.Shutdown()
		}
		app.Quit()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
