package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// Dir returns the root configuration directory for Hikari.
// Respects XDG_CONFIG_HOME on Linux/macOS, uses %APPDATA% on Windows.
func Dir() string {
	if dir := os.Getenv("HIKARI_CONFIG_DIR"); dir != "" {
		return dir
	}

	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(base, "hikari")
	default:
		base := os.Getenv("XDG_CONFIG_HOME")
		if base == "" {
			base = filepath.Join(os.Getenv("HOME"), ".config")
		}
		return filepath.Join(base, "hikari")
	}
}

// ConfigFile returns the path to config.toml.
func ConfigFile() string {
	return filepath.Join(Dir(), "config.toml")
}

// PersonasDir returns the path to the user personas directory.
func PersonasDir() string {
	return filepath.Join(Dir(), "personas")
}

// ThemesDir returns the path to the user themes directory.
func ThemesDir() string {
	return filepath.Join(Dir(), "themes")
}

// ASCIIDir returns the path to the user ASCII art directory.
func ASCIIDir() string {
	return filepath.Join(Dir(), "ascii")
}

// SessionsDir returns the path to the sessions directory.
func SessionsDir() string {
	return filepath.Join(Dir(), "sessions")
}

// EnsureDirs creates all required config directories if they do not exist.
func EnsureDirs() error {
	dirs := []string{
		Dir(),
		PersonasDir(),
		ThemesDir(),
		ASCIIDir(),
		SessionsDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			return err
		}
	}
	return nil
}
