// Package logging provides the application's persistent log setup: an
// always-on errors.txt at WARN+ level, and a dev-only network.txt that
// captures full HTTP request/response dumps.
//
// Files live under {AppDataDir}/logs/ and are size-rotated via lumberjack.
package logging

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	errorsFileName  = "errors.txt"
	networkFileName = "network.txt"

	defaultMaxSizeMB  = 5
	defaultMaxBackups = 3
)

var (
	mu          sync.Mutex
	errorsFile  *lumberjack.Logger
	initialized bool
)

// Init opens {dir}/errors.txt, wires slog.Default() to a WARN-level text
// handler that tees to the file and stderr, and redirects the stdlib log
// package so bare log.Printf calls are captured as slog WARNs.
//
// Safe to call only once per process. A second call is a no-op.
func Init(dir string) error {
	return initWith(dir, defaultMaxSizeMB, defaultMaxBackups)
}

func initWith(dir string, maxSizeMB, maxBackups int) error {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir logs: %w", err)
	}

	lj := &lumberjack.Logger{
		Filename:   filepath.Join(dir, errorsFileName),
		MaxSize:    maxSizeMB,
		MaxBackups: maxBackups,
		Compress:   false,
	}

	multi := io.MultiWriter(lj, os.Stderr)
	handler := slog.NewTextHandler(multi, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Bridge stdlib log.Printf -> slog.Warn so existing log sites that haven't
	// been converted still land in errors.txt.
	log.SetOutput(&stdLogBridge{logger: logger})
	log.SetFlags(0)
	log.SetPrefix("")

	errorsFile = lj
	initialized = true
	return nil
}

// Close flushes and closes the errors.txt file. Safe to call multiple times.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if errorsFile == nil {
		return nil
	}
	err := errorsFile.Close()
	errorsFile = nil
	initialized = false
	return err
}

// stdLogBridge is an io.Writer that forwards each write as a slog WARN record.
type stdLogBridge struct {
	logger *slog.Logger
}

func (b *stdLogBridge) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg == "" {
		return len(p), nil
	}
	b.logger.LogAttrs(context.Background(), slog.LevelWarn, msg)
	return len(p), nil
}

var _ io.Writer = (*stdLogBridge)(nil)
