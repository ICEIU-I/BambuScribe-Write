package main

import (
	"bambuscribe/desktop/internal/application"
	"bambuscribe/desktop/internal/appmenu"
	"embed"
	"log"
	"os"

	wailsapp "github.com/wailsapp/wails/v2/pkg/application"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	title := "A1 手写助手"
	if os.Getenv("BAMBUSCRIBE_CONFIG_DIR") != "" {
		title += " · 验证"
	}
	options := &options.App{
		Title:            title,
		Width:            1360,
		Height:           900,
		MinWidth:         1100,
		MinHeight:        720,
		BackgroundColour: &options.RGBA{R: 245, G: 246, B: 243, A: 255},
		AssetServer:      &assetserver.Options{Assets: assets},
		Mac:              &mac.Options{DisableEscapeExitsFullscreen: true},
	}
	// The shell is created first so the app can rebuild the native menu later,
	// when the frontend reports the language the editor is running in.
	shell := wailsapp.NewWithOptions(options)
	ourApp, startup, shutdown := application.New(shell)
	options.OnStartup = startup
	options.OnShutdown = shutdown
	options.OnBeforeClose = ourApp.BeforeClose
	options.Bind = []interface{}{ourApp}
	shell.SetApplicationMenu(appmenu.Build(appmenu.State{}, ourApp.MenuAction))
	appmenu.Relabel()
	if err := shell.Run(); err != nil {
		log.Fatal(err)
	}
}
