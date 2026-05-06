package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/logging"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func init() {
	// Force the cgo (system) DNS resolver so we honor macOS resolver configs
	// (/etc/resolver/*, VPN DNS, mDNSResponder). Go's pure-Go resolver misses
	// these and fails "no such host" on hosts browsers can reach.
	if os.Getenv("GODEBUG") == "" {
		_ = os.Setenv("GODEBUG", "netdns=cgo")
	}
}

//go:embed all:frontend/dist
var assets embed.FS

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// Initialize persistent logging before anything else so startup errors
	// land in errors.txt. AppDataDir also ensures the base dir exists.
	dataDir, err := database.AppDataDir()
	if err != nil {
		log.Fatalf("resolving app data dir: %v", err)
	}
	if err := logging.Init(filepath.Join(dataDir, "logs")); err != nil {
		log.Fatalf("initializing logger: %v", err)
	}
	log.Printf("oversite version=%s", version)

	app, err := NewApp()
	if err != nil {
		log.Fatalf("creating app: %v", err)
	}

	err = wails.Run(&options.App{
		Title:  "Oversite",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
