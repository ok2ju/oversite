package main

import (
	"embed"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/ok2ju/oversite/internal/database"
	"github.com/ok2ju/oversite/internal/logging"
	"github.com/ok2ju/oversite/internal/sysinfo"
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

// heapLimits is set by configureMemoryLimits at startup and consumed by NewApp
// to size the parser kill-switch. Lives at package scope so the binding code
// doesn't need a separate plumbing path.
var heapLimits sysinfo.HeapLimits

// configureMemoryLimits sizes the Go GC's soft heap target to a fraction of
// host RAM and stashes the matching kill-switch for the parser. Without
// GOMEMLIMIT the runtime lets the heap grow to ~2× the live set before
// collecting, which on a 16 GB Windows box (WebView2 ~1-2 GB plus OS) was
// pushing the OS into pagefile-backed thrash during demo imports before the
// in-parser watchdog could fire. Skipped if the user already set GOMEMLIMIT
// out-of-band.
func configureMemoryLimits() {
	totalRAM, err := sysinfo.TotalRAM()
	if err != nil {
		slog.Warn("sysinfo: total RAM detection failed; using conservative floor", "err", err)
	}
	heapLimits = sysinfo.RecommendedHeapLimits(totalRAM)

	if os.Getenv("GOMEMLIMIT") == "" {
		debug.SetMemoryLimit(int64(heapLimits.GOMEMLIMIT))
	}
	slog.Info("memory limits configured",
		"total_ram_mb", totalRAM>>20,
		"gomemlimit_mb", heapLimits.GOMEMLIMIT>>20,
		"parser_killswitch_mb", heapLimits.KillSwitch>>20,
	)
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

	configureMemoryLimits()

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
