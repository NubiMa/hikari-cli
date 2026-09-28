// Package config handles loading and validating Hikari's configuration.
//
// Configuration is stored at ~/.config/hikari/config.toml (or the
// XDG_CONFIG_HOME equivalent). Environment variables can override sensitive
// fields such as provider tokens.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// ---------------------------------------------------------------------------
// Top-level Config
// ---------------------------------------------------------------------------

// Config is the root configuration structure for Hikari.
type Config struct {
	Default   DefaultConfig             `toml:"default"`
	Providers map[string]ProviderConfig `toml:"providers"`
	UI        UIConfig                  `toml:"ui"`
}

// DefaultConfig holds the active defaults chosen by the user.
type DefaultConfig struct {
	Provider string `toml:"provider"`
	Persona  string `toml:"persona"`
	Model    string `toml:"model"`
}

// UIConfig holds display preferences.
type UIConfig struct {
	Theme string `toml:"theme"`
	ASCII string `toml:"ascii"`
}

// ---------------------------------------------------------------------------
// Provider Config
// ---------------------------------------------------------------------------

// ProviderConfig holds the configuration for a single provider instance.
type ProviderConfig struct {
	// Type identifies which provider implementation to use.
	// Valid values: "openclaw", "hermes", "ollama"
	Type string `toml:"type"`

	// Endpoint is the base URL for the provider API.
	Endpoint string `toml:"endpoint"`

	// Token is an optional API key / bearer token.
	// Can also be set via HIKARI_<NAME>_TOKEN environment variable,
	// where <NAME> is the provider's map key uppercased.
	Token string `toml:"token"`

	// Model is the default model to use with this provider.
	// Only meaningful for model providers like Ollama.
	Model string `toml:"model"`

	// TimeoutSeconds is the HTTP timeout for requests to this provider.
	TimeoutSeconds int `toml:"timeout_seconds"`

	// TLSSkipVerify disables TLS certificate verification.
	// Use only for development / self-signed certs.
	TLSSkipVerify bool `toml:"tls_skip_verify"`
}

// ---------------------------------------------------------------------------
// Load / Default
// ---------------------------------------------------------------------------

// Load reads config from the standard config file path.
// Missing file is not an error — defaults are returned instead.
func Load() (*Config, error) {
	path := ConfigFile()
	return LoadFrom(path)
}

// LoadFrom reads config from a specific path.
func LoadFrom(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	applyEnvOverrides(cfg)

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// defaultConfig returns a Config with sensible defaults.
func defaultConfig() *Config {
	return &Config{
		Default: DefaultConfig{
			Provider: "",
			Persona:  "default",
			Model:    "",
		},
		Providers: make(map[string]ProviderConfig),
		UI: UIConfig{
			Theme: "default",
			ASCII: "default",
		},
	}
}

// ---------------------------------------------------------------------------
// Environment variable overrides
// ---------------------------------------------------------------------------

// applyEnvOverrides replaces token values from environment variables.
// The env var name is HIKARI_<PROVIDER_NAME_UPPERCASE>_TOKEN.
// This prevents secrets from being stored in plaintext config files.
func applyEnvOverrides(cfg *Config) {
	for name, pCfg := range cfg.Providers {
		envKey := "HIKARI_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_TOKEN"
		if val := os.Getenv(envKey); val != "" {
			pCfg.Token = val
			cfg.Providers[name] = pCfg
		}
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validate(cfg *Config) error {
	validTypes := map[string]bool{
		"openclaw": true,
		"hermes":   true,
		"ollama":   true,
	}

	for name, p := range cfg.Providers {
		if !validTypes[p.Type] {
			return fmt.Errorf("provider %q has unknown type %q (valid: openclaw, hermes, ollama)", name, p.Type)
		}
		if p.Endpoint == "" {
			return fmt.Errorf("provider %q missing required field: endpoint", name)
		}
	}

	if cfg.Default.Provider != "" {
		if _, ok := cfg.Providers[cfg.Default.Provider]; !ok {
			return fmt.Errorf("default provider %q is not defined in [providers]", cfg.Default.Provider)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Write (initial setup)
// ---------------------------------------------------------------------------

// WriteDefault writes a starter config.toml if none exists.
func WriteDefault() error {
	path := ConfigFile()
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	if err := EnsureDirs(); err != nil {
		return err
	}

	const defaultTOML = `# Hikari Configuration
# Documentation: https://github.com/nubiv/hikari

[default]
provider = ""
persona  = "default"
model    = ""

[ui]
theme = "default"
ascii = "default"

# Add providers below. Example:
#
# [providers.ollama-local]
# type     = "ollama"
# endpoint = "http://127.0.0.1:11434"
# model    = "llama3.2"
#
# [providers.openclaw-vps]
# type     = "openclaw"
# endpoint = "https://agent.example.com"
# token    = ""   # or set HIKARI_OPENCLAW_VPS_TOKEN env var
`

	return os.WriteFile(path, []byte(defaultTOML), 0600)
}
