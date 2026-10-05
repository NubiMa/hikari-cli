package main

import (
	"path/filepath"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/config"
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
