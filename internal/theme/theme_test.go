package theme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/theme"
)

func TestThemePresets(t *testing.T) {
	def := theme.DefaultTheme()
	if def.Name != "default" {
		t.Errorf("expected default theme name, got %q", def.Name)
	}
	if def.Colors.Primary == "" {
		t.Error("expected non-empty primary color")
	}

	min := theme.MinimalTheme()
	if min.Name != "minimal" {
		t.Errorf("expected minimal theme name, got %q", min.Name)
	}

	tokyo := theme.TokyoNightTheme()
	if tokyo.Name != "tokyo-night" {
		t.Errorf("expected tokyo-night theme name, got %q", tokyo.Name)
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.toml")

	content := `name = "custom"
description = "My Custom Theme"

[colors]
primary = "#FF0000"
accent = "#00FF00"
background = "#000000"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	thm, err := theme.LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thm.Name != "custom" {
		t.Errorf("expected name 'custom', got %q", thm.Name)
	}
	if thm.Colors.Primary != "#FF0000" {
		t.Errorf("expected primary #FF0000, got %q", thm.Colors.Primary)
	}
}

func TestThemeManager(t *testing.T) {
	dir := t.TempDir()
	themesDir := filepath.Join(dir, "themes")
	asciiDir := filepath.Join(dir, "ascii")
	_ = os.MkdirAll(themesDir, 0700)
	_ = os.MkdirAll(asciiDir, 0700)

	// Write custom theme
	customContent := `name = "cyber"
description = "Cyberpunk"

[colors]
primary = "#00FFFF"
`
	_ = os.WriteFile(filepath.Join(themesDir, "cyber.toml"), []byte(customContent), 0600)

	mgr, err := theme.NewManager(themesDir, asciiDir)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if mgr.Active().Name != "default" {
		t.Errorf("expected active default, got %q", mgr.Active().Name)
	}

	if err := mgr.SetActive("cyber"); err != nil {
		t.Fatalf("failed to set active: %v", err)
	}
	if mgr.Active().Name != "cyber" {
		t.Errorf("expected cyber, got %q", mgr.Active().Name)
	}

	if err := mgr.SetActive("nonexistent"); err == nil {
		t.Error("expected error for nonexistent theme")
	}

	// Test ASCII loading
	_ = os.WriteFile(filepath.Join(asciiDir, "custom.txt"), []byte("ART"), 0600)
	loaded := mgr.LoadASCII("custom", "FALLBACK")
	if loaded != "ART" {
		t.Errorf("expected ART, got %q", loaded)
	}

	fallback := mgr.LoadASCII("missing", "FALLBACK")
	if fallback != "FALLBACK" {
		t.Errorf("expected FALLBACK, got %q", fallback)
	}
}
