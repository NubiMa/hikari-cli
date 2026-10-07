package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/persona"
)

func TestConfigOpenClaw(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	err := runConfigureOpenClaw("my-openclaw", "https://agent.test.com", "secret-token", 90, true, true)
	if err != nil {
		t.Fatalf("runConfigureOpenClaw failed: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading saved config failed: %v", err)
	}

	pcfg, ok := cfg.Providers["my-openclaw"]
	if !ok {
		t.Fatalf("expected provider 'my-openclaw' to exist in config")
	}

	if pcfg.Type != "openclaw" {
		t.Errorf("expected type openclaw, got %s", pcfg.Type)
	}
	if pcfg.Endpoint != "https://agent.test.com" {
		t.Errorf("expected endpoint https://agent.test.com, got %s", pcfg.Endpoint)
	}
	if pcfg.Token != "secret-token" {
		t.Errorf("expected token secret-token, got %s", pcfg.Token)
	}
	if pcfg.TimeoutSeconds != 90 {
		t.Errorf("expected timeout 90, got %d", pcfg.TimeoutSeconds)
	}
	if cfg.Default.Provider != "my-openclaw" {
		t.Errorf("expected default provider my-openclaw, got %s", cfg.Default.Provider)
	}
}

func TestConfigHermes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	err := runConfigureHermes("my-hermes", "http://127.0.0.1:9090", "token123", 60, true, true)
	if err != nil {
		t.Fatalf("runConfigureHermes failed: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading saved config failed: %v", err)
	}

	pcfg, ok := cfg.Providers["my-hermes"]
	if !ok {
		t.Fatalf("expected provider 'my-hermes' to exist in config")
	}

	if pcfg.Type != "hermes" {
		t.Errorf("expected type hermes, got %s", pcfg.Type)
	}
	if pcfg.Endpoint != "http://127.0.0.1:9090" {
		t.Errorf("expected endpoint http://127.0.0.1:9090, got %s", pcfg.Endpoint)
	}
	if pcfg.TimeoutSeconds != 60 {
		t.Errorf("expected timeout 60, got %d", pcfg.TimeoutSeconds)
	}
	if cfg.Default.Provider != "my-hermes" {
		t.Errorf("expected default provider my-hermes, got %s", cfg.Default.Provider)
	}
}

func TestConfigOllama(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	err := runConfigureOllama("my-ollama", "http://127.0.0.1:11434", "llama3.2", "", 120, true, true)
	if err != nil {
		t.Fatalf("runConfigureOllama failed: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading saved config failed: %v", err)
	}

	pcfg, ok := cfg.Providers["my-ollama"]
	if !ok {
		t.Fatalf("expected provider 'my-ollama' to exist in config")
	}

	if pcfg.Type != "ollama" {
		t.Errorf("expected type ollama, got %s", pcfg.Type)
	}
	if pcfg.Model != "llama3.2" {
		t.Errorf("expected model llama3.2, got %s", pcfg.Model)
	}
	if cfg.Default.Provider != "my-ollama" {
		t.Errorf("expected default provider my-ollama, got %s", cfg.Default.Provider)
	}
}

func TestConfigPersonaDirect(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	err := runSetPersona("developer")
	if err != nil {
		t.Fatalf("runSetPersona failed: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading saved config failed: %v", err)
	}

	if cfg.Default.Persona != "developer" {
		t.Errorf("expected default persona developer, got %s", cfg.Default.Persona)
	}
}

func TestConfigThemeDirect(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	err := runSetTheme("tokyo-night")
	if err != nil {
		t.Fatalf("runSetTheme failed: %v", err)
	}

	cfg, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading saved config failed: %v", err)
	}

	if cfg.UI.Theme != "tokyo-night" {
		t.Errorf("expected default theme tokyo-night, got %s", cfg.UI.Theme)
	}
}

func TestOpenPersonaEditor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)
	t.Setenv("EDITOR", "true")

	err := openPersonaEditor("custom-bot")
	if err != nil {
		t.Fatalf("openPersonaEditor failed: %v", err)
	}

	personaFile := filepath.Join(tmpDir, "personas", "custom-bot.yaml")
	if _, err := os.Stat(personaFile); os.IsNotExist(err) {
		t.Fatalf("expected persona file %s to be created", personaFile)
	}
}

func TestOpenThemeEditor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)
	t.Setenv("EDITOR", "true")

	err := openThemeEditor("my-theme")
	if err != nil {
		t.Fatalf("openThemeEditor failed: %v", err)
	}

	themeFile := filepath.Join(tmpDir, "themes", "my-theme.toml")
	if _, err := os.Stat(themeFile); os.IsNotExist(err) {
		t.Fatalf("expected theme file %s to be created", themeFile)
	}
}

func TestListPersonasAndThemes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	if err := listPersonas(); err != nil {
		t.Fatalf("listPersonas failed: %v", err)
	}

	if err := listThemes(); err != nil {
		t.Fatalf("listThemes failed: %v", err)
	}
}

func TestSavePersonaAndConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HIKARI_CONFIG_DIR", tmpDir)

	cfg := &config.Config{}
	p := &persona.Persona{
		Name:         "Super Assistant",
		Description:  "Fast and witty",
		Greeting:     "Yo!",
		SystemPrompt: "You are super fast.",
		Behavior: persona.Behavior{
			Tone:     "witty",
			Language: "English",
		},
	}

	err := savePersonaAndConfig(cfg, "super-assistant", p, true)
	if err != nil {
		t.Fatalf("savePersonaAndConfig failed: %v", err)
	}

	// Verify file was written
	pFile := filepath.Join(tmpDir, "personas", "super-assistant.yaml")
	loaded, err := persona.LoadFromFile(pFile)
	if err != nil {
		t.Fatalf("loading saved persona file failed: %v", err)
	}
	if loaded.Name != "Super Assistant" {
		t.Errorf("expected name 'Super Assistant', got %q", loaded.Name)
	}
	if loaded.Greeting != "Yo!" {
		t.Errorf("expected greeting 'Yo!', got %q", loaded.Greeting)
	}
	if loaded.Behavior.Tone != "witty" {
		t.Errorf("expected tone 'witty', got %q", loaded.Behavior.Tone)
	}

	// Verify config.toml was updated with active default
	cfgLoaded, err := config.LoadFrom(filepath.Join(tmpDir, "config.toml"))
	if err != nil {
		t.Fatalf("loading config.toml failed: %v", err)
	}
	if cfgLoaded.Default.Persona != "super-assistant" {
		t.Errorf("expected default persona 'super-assistant', got %q", cfgLoaded.Default.Persona)
	}

	// Verify buildPersonaItems includes super-assistant
	items := buildPersonaItems()
	found := false
	for _, it := range items {
		if it.id == "super-assistant" {
			found = true
			if it.name != "Super Assistant" {
				t.Errorf("expected item name 'Super Assistant', got %q", it.name)
			}
			break
		}
	}
	if !found {
		t.Errorf("expected 'super-assistant' to be listed in buildPersonaItems")
	}

	// Test deleting persona
	err = deletePersona(cfgLoaded, "super-assistant")
	if err != nil {
		t.Fatalf("deletePersona failed: %v", err)
	}
	if _, err := os.Stat(pFile); !os.IsNotExist(err) {
		t.Errorf("expected persona file to be deleted")
	}
}
