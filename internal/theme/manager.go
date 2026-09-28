package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nubiv/hikari/internal/config"
)

// Manager handles discovering, switching, and retrieving color themes.
type Manager struct {
	themes   map[string]Theme
	order    []string
	active   string
	asciiDir string
}

// NewManager creates a ThemeManager with built-in presets and user themes.
func NewManager(userThemesDir, userASCIIDir string) (*Manager, error) {
	m := &Manager{
		themes:   make(map[string]Theme),
		asciiDir: userASCIIDir,
	}

	// Register built-in presets
	presets := []Theme{
		DefaultTheme(),
		MinimalTheme(),
		TokyoNightTheme(),
	}
	for _, p := range presets {
		m.themes[p.Name] = p
		m.order = append(m.order, p.Name)
	}
	m.active = "default"

	// Load user themes if directory exists
	if userThemesDir != "" {
		if err := m.loadUserThemes(userThemesDir); err != nil {
			// Don't fail completely on user theme read error, just proceed with presets
			fmt.Fprintf(os.Stderr, "warning: loading user themes: %v\n", err)
		}
	}

	return m, nil
}

// NewManagerDefault initialises the manager using default paths.
func NewManagerDefault() (*Manager, error) {
	return NewManager(config.ThemesDir(), config.ASCIIDir())
}

func (m *Manager) loadUserThemes(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		t, err := LoadFile(path)
		if err != nil {
			continue
		}

		if _, exists := m.themes[t.Name]; !exists {
			m.order = append(m.order, t.Name)
		}
		m.themes[t.Name] = t
	}
	return nil
}

// Active returns the currently selected theme.
func (m *Manager) Active() Theme {
	if t, ok := m.themes[m.active]; ok {
		return t
	}
	return DefaultTheme()
}

// SetActive changes the current theme by name.
func (m *Manager) SetActive(name string) error {
	if _, ok := m.themes[name]; !ok {
		return fmt.Errorf("theme %q not found", name)
	}
	m.active = name
	return nil
}

// All returns all available themes in registration order.
func (m *Manager) All() []Theme {
	result := make([]Theme, 0, len(m.order))
	for _, name := range m.order {
		if t, ok := m.themes[name]; ok {
			result = append(result, t)
		}
	}
	return result
}

// LoadASCII loads an ASCII banner by name. It searches user ASCIIDir first,
// falling back to the provided default string or embedded default.
func (m *Manager) LoadASCII(name string, defaultASCII string) string {
	if m.asciiDir != "" && name != "" {
		candidate := filepath.Join(m.asciiDir, name+".txt")
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
	}
	return defaultASCII
}
