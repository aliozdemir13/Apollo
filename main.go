package main

import (
	"embed"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger) // to use slog.Info(...) anywhere

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Apollo",
		Width:     380,
		Height:    420,
		MinWidth:  380,
		MinHeight: 420,
		// A widget: no chrome, floats above other windows, transparent corners.
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Mac: &mac.Options{
			// Transparent webview, but NOT translucent: WindowIsTranslucent adds a
			// frosted material that fills the whole rectangular window and shows as
			// a gray frame behind the rounded device. We want the window fully clear.
			WebviewIsTransparent: true,
			WindowIsTranslucent:  false,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
			ProgramName:         "Apollo",
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}

func SetupLogger() (*os.File, error) {
	// Get the standard OS cache directory (e.g., ~/.cache on Linux)
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	appLogDir := filepath.Join(cacheDir, "apollo")
	os.MkdirAll(appLogDir, 0755)

	logFile, err := os.OpenFile(filepath.Join(appLogDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Set up a structured text or JSON logger
	logger := slog.New(slog.NewTextHandler(logFile, nil))
	slog.SetDefault(logger)

	return logFile, nil
}
