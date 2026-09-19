package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const appName = "launch-pg"

// Paths holds the on-disk locations the tool reads and writes.
type Paths struct {
	ConfigDir string
	StateDir  string
}

// DefaultPaths resolves XDG-style locations, falling back to ~/.config and
// ~/.local/state on every platform so behavior is the same on macOS and Linux.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	return Paths{
		ConfigDir: xdgDir("XDG_CONFIG_HOME", filepath.Join(home, ".config")),
		StateDir:  xdgDir("XDG_STATE_HOME", filepath.Join(home, ".local", "state")),
	}, nil
}

func xdgDir(envVar, fallback string) string {
	if base := os.Getenv(envVar); base != "" {
		return filepath.Join(base, appName)
	}
	return filepath.Join(fallback, appName)
}

func (p Paths) ConfigFile() string   { return filepath.Join(p.ConfigDir, "config.yaml") }
func (p Paths) PassFile() string     { return filepath.Join(p.ConfigDir, "pgpass") }
func (p Paths) TemplatesDir() string { return filepath.Join(p.ConfigDir, "templates") }
func (p Paths) AuditFile() string    { return filepath.Join(p.StateDir, "audit.jsonl") }
func (p Paths) BackupsDir() string   { return filepath.Join(p.StateDir, "backups") }
