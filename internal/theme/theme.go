// Package theme provides color theme configuration and management for Hikari.
package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Theme represents a color palette configuration for the Hikari TUI.
type Theme struct {
	Name        string       `toml:"name"`
	Description string       `toml:"description"`
	Colors      ColorPalette `toml:"colors"`
}

// ColorPalette defines hex colors for all TUI elements.
type ColorPalette struct {
	Primary    string `toml:"primary"`
	Accent     string `toml:"accent"`
	Dim        string `toml:"dim"`
	Subtle     string `toml:"subtle"`
	Text       string `toml:"text"`
	TextMuted  string `toml:"text_muted"`
	User       string `toml:"user"`
	Assistant  string `toml:"assistant"`
	System     string `toml:"system"`
	Success    string `toml:"success"`
	Error      string `toml:"error"`
	Warning    string `toml:"warning"`
	Border     string `toml:"border"`
	Background string `toml:"background"`
}

// DefaultTheme returns the default purple/violet dark theme for Hikari.
func DefaultTheme() Theme {
	return Theme{
		Name:        "default",
		Description: "Hikari Violet Night (Default)",
		Colors: ColorPalette{
			Primary:    "#7C3AED", // violet-600
			Accent:     "#A78BFA", // violet-400
			Dim:        "#6B7280", // gray-500
			Subtle:     "#374151", // gray-700
			Text:       "#F9FAFB", // gray-50
			TextMuted:  "#9CA3AF", // gray-400
			User:       "#34D399", // emerald-400
			Assistant:  "#A78BFA", // violet-400
			System:     "#FCD34D", // amber-300
			Success:    "#10B981", // emerald-500
			Error:      "#F87171", // red-400
			Warning:    "#FBBF24", // amber-400
			Border:     "#4C1D95", // violet-900
			Background: "#0F0F1A", // near-black
		},
	}
}

// MinimalTheme returns a clean monochrome / minimal theme.
func MinimalTheme() Theme {
	return Theme{
		Name:        "minimal",
		Description: "Minimal Clean Monochrome",
		Colors: ColorPalette{
			Primary:    "#4B5563", // gray-600
			Accent:     "#E5E7EB", // gray-200
			Dim:        "#4B5563", // gray-600
			Subtle:     "#1F2937", // gray-800
			Text:       "#F3F4F6", // gray-100
			TextMuted:  "#9CA3AF", // gray-400
			User:       "#60A5FA", // blue-400
			Assistant:  "#E5E7EB", // gray-200
			System:     "#FBBF24", // amber-400
			Success:    "#34D399", // emerald-400
			Error:      "#EF4444", // red-500
			Warning:    "#F59E0B", // amber-500
			Border:     "#374151", // gray-700
			Background: "#111827", // gray-900
		},
	}
}

// TokyoNightTheme returns a vibrant Tokyo Night theme.
func TokyoNightTheme() Theme {
	return Theme{
		Name:        "tokyo-night",
		Description: "Tokyo Night dark neon theme",
		Colors: ColorPalette{
			Primary:    "#7AA2F7", // blue
			Accent:     "#BB9AF7", // purple
			Dim:        "#565F89", // dark comment
			Subtle:     "#24283B", // dark blue
			Text:       "#C0CAF5", // light foreground
			TextMuted:  "#787C99", // muted
			User:       "#73DACA", // cyan
			Assistant:  "#BB9AF7", // purple
			System:     "#E0AF68", // yellow
			Success:    "#9ECE6A", // green
			Error:      "#F7768E", // red
			Warning:    "#FF9E64", // orange
			Border:     "#3B4261", // border blue
			Background: "#1A1B26", // background
		},
	}
}

// LoadFile parses a theme from a TOML file.
func LoadFile(path string) (Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Theme{}, fmt.Errorf("reading theme %s: %w", path, err)
	}

	var t Theme
	if err := toml.Unmarshal(data, &t); err != nil {
		return Theme{}, fmt.Errorf("parsing theme %s: %w", path, err)
	}

	if t.Name == "" {
		t.Name = filepath.Base(path)
		ext := filepath.Ext(t.Name)
		t.Name = t.Name[:len(t.Name)-len(ext)]
	}

	return t, nil
}
