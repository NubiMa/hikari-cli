package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/config"
)

func TestLoadMissingFile(t *testing.T) {
	// Point to a non-existent path — should return defaults, not error.
	cfg, err := config.LoadFrom("/tmp/hikari_test_nonexistent_config.toml")
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

func TestLoadValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
[default]
provider = "ollama-local"
persona  = "nino"

[providers.ollama-local]
type     = "ollama"
endpoint = "http://127.0.0.1:11434"
model    = "llama3.2"

[ui]
theme = "default"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Default.Provider != "ollama-local" {
		t.Errorf("expected provider 'ollama-local', got %q", cfg.Default.Provider)
	}
	p, ok := cfg.Providers["ollama-local"]
	if !ok {
		t.Fatal("expected 'ollama-local' in providers map")
	}
	if p.Type != "ollama" {
		t.Errorf("expected type 'ollama', got %q", p.Type)
	}
	if p.Endpoint != "http://127.0.0.1:11434" {
		t.Errorf("unexpected endpoint: %q", p.Endpoint)
	}
}

func TestValidationUnknownProviderType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
[providers.bad]
type     = "nonexistent"
endpoint = "http://127.0.0.1:9999"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected validation error for unknown provider type")
	}
}

func TestValidationDefaultProviderNotDefined(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
[default]
provider = "does-not-exist"

[providers.ollama-local]
type     = "ollama"
endpoint = "http://127.0.0.1:11434"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadFrom(path)
	if err == nil {
		t.Fatal("expected validation error for undefined default provider")
	}
}

func TestEnvVarTokenOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	content := `
[default]
provider = "openclaw-vps"

[providers.openclaw-vps]
type     = "openclaw"
endpoint = "https://agent.example.com"
token    = "old-token"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HIKARI_OPENCLAW_VPS_TOKEN", "new-secure-token")

	cfg, err := config.LoadFrom(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	p := cfg.Providers["openclaw-vps"]
	if p.Token != "new-secure-token" {
		t.Errorf("expected env var to override token, got %q", p.Token)
	}
}
