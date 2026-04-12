package database

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "modernc.org/sqlite" // Register pure-Go SQLite driver as "sqlite".
)

// Open creates a SQLite connection with WAL mode and foreign keys enabled.
// The caller is responsible for closing the returned *sql.DB.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	// Single connection — SQLite is single-writer.
	db.SetMaxOpenConns(1)

	// Enable WAL mode for concurrent reads.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		db.Close()
		return nil, fmt.Errorf("PRAGMA journal_mode=WAL: %w", err)
	}
	if mode != "wal" {
		db.Close()
		return nil, fmt.Errorf("expected journal_mode=wal, got %q", mode)
	}

	// Enable foreign key enforcement.
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("PRAGMA foreign_keys=ON: %w", err)
	}

	return db, nil
}

// RunMigrations applies all pending up migrations from the embedded filesystem.
func RunMigrations(db *sql.DB, sourceFS embed.FS) error {
	return runMigrate(db, sourceFS, func(m *migrate.Migrate) error {
		err := m.Up()
		if err == migrate.ErrNoChange {
			return nil
		}
		return err
	})
}

// DownMigrations rolls back all migrations (used in tests).
func DownMigrations(db *sql.DB, sourceFS embed.FS) error {
	return runMigrate(db, sourceFS, func(m *migrate.Migrate) error {
		err := m.Down()
		if err == migrate.ErrNoChange {
			return nil
		}
		return err
	})
}

func runMigrate(db *sql.DB, sourceFS embed.FS, fn func(*migrate.Migrate) error) error {
	src, err := iofs.New(sourceFS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("sqlite3 migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite3", driver)
	if err != nil {
		return fmt.Errorf("migrate.NewWithInstance: %w", err)
	}

	if err := fn(m); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// OpenWithMigrations opens a SQLite database and runs all pending migrations.
func OpenWithMigrations(dbPath string, fs embed.FS) (*sql.DB, error) {
	db, err := Open(dbPath)
	if err != nil {
		return nil, err
	}
	if err := RunMigrations(db, fs); err != nil {
		db.Close()
		return nil, fmt.Errorf("RunMigrations: %w", err)
	}
	return db, nil
}

// AppDataDir returns the OS-specific application data directory for Oversite.
// The directory is created if it does not exist.
func AppDataDir() (string, error) {
	var base string
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("UserHomeDir: %w", err)
		}
		base = filepath.Join(home, "Library", "Application Support")
	case "windows":
		base = os.Getenv("APPDATA")
		if base == "" {
			return "", fmt.Errorf("APPDATA not set")
		}
	default: // linux and others
		base = os.Getenv("XDG_DATA_HOME")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("UserHomeDir: %w", err)
			}
			base = filepath.Join(home, ".local", "share")
		}
	}

	dir := filepath.Join(base, "oversite")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("MkdirAll %q: %w", dir, err)
	}
	return dir, nil
}

// DefaultDBPath returns the default path for the Oversite database file.
func DefaultDBPath() (string, error) {
	dir, err := AppDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "oversite.db"), nil
}
