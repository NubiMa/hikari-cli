// Package config handles loading and validating Hikari's configuration.
//
// Configuration is stored at ~/.config/hikari/config.toml (or the
// XDG_CONFIG_HOME equivalent). Environment variables can override sensitive
// fields such as provider tokens.
package config

import (
	"bytes"
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
	// Valid values: "openclaw", "hermes", "ollama", "custom"
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

	// Compatibility describes the chat API protocol used by this provider.
	// Only used when Type == "custom".
	// Currently supported: "openai" (default — OpenAI /chat/completions format).
	// Future values: "anthropic", "cohere", "raw".
	Compatibility string `toml:"compatibility"`

	// HealthPath is the URL path used for connection health-checks.
	// Only used when Type == "custom". Defaults to "/models" when empty.
	HealthPath string `toml:"health_path"`
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
		"custom":   true,
	}

	for name, p := range cfg.Providers {
		if !validTypes[p.Type] {
			return fmt.Errorf("provider %q has unknown type %q (valid: openclaw, hermes, ollama, custom)", name, p.Type)
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

// Exists reports whether a config file already exists on disk.
func Exists() bool {
	_, err := os.Stat(ConfigFile())
	return err == nil
}

// WriteDefault writes a richly commented starter config.toml if none exists.
func WriteDefault() error {
	path := ConfigFile()
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}

	if err := EnsureDirs(); err != nil {
		return err
	}

	const defaultTOML = `# Hikari Configuration
# Documentation: https://github.com/NubiMa/hikari-cli
# Run 'hikari config' for an interactive setup wizard.

[default]
# Name of the active provider (must match a key in [providers.*] below).
provider = ""

# Active persona: "hikari", "developer", "sysadmin", or any custom persona name.
persona  = "hikari"

# Default model override (optional — provider config takes precedence if blank).
model    = ""

[ui]
# Theme: "default", "minimal", or "tokyo-night"
theme = "default"
# ASCII banner: "default", "minimal", or "tokyo-night"
ascii = "default"

# ---------------------------------------------------------------------------
# Providers — uncomment and fill in the block(s) you want to use.
# You can define multiple providers and switch between them with /provider.
# ---------------------------------------------------------------------------

# ── Option A: Ollama (Local Workstation) ────────────────────────────────────
# Runs open-source LLMs locally. Free, offline, private.
# 1. Install Ollama: https://ollama.com
# 2. Pull a model: ollama pull llama3.2
# 3. Uncomment and set provider = "ollama-local" in [default] above.
#
# [providers.ollama-local]
# type            = "ollama"
# endpoint        = "http://127.0.0.1:11434"
# model           = "llama3.2"
# timeout_seconds = 120

# ── Option B: Ollama (Remote VPS / LAN) ─────────────────────────────────────
# Point Hikari at an Ollama instance on a remote server or GPU machine.
# On the server: OLLAMA_HOST=0.0.0.0:11434 ollama serve
#
# [providers.ollama-vps]
# type            = "ollama"
# endpoint        = "https://ollama.yourserver.com"
# model           = "qwen2.5-coder:7b"
# timeout_seconds = 180
# tls_skip_verify = false   # set true only for self-signed TLS certs

# ── Option C: OpenClaw (Autonomous Agent) ───────────────────────────────────
# Full autonomous agent with tool execution, shell commands, and memory.
# token can also be set via HIKARI_OPENCLAW_VPS_TOKEN env variable.
#
# [providers.openclaw-vps]
# type            = "openclaw"
# endpoint        = "https://agent.yourserver.com"
# token           = ""   # or export HIKARI_OPENCLAW_VPS_TOKEN=...
# timeout_seconds = 180

# ── Option D: Hermes (Agent Pipeline) ───────────────────────────────────────
# Multi-turn agent backend with function calling and task planning.
# token can also be set via HIKARI_HERMES_LOCAL_TOKEN env variable.
#
# [providers.hermes-local]
# type            = "hermes"
# endpoint        = "http://127.0.0.1:8080"
# token           = ""   # or export HIKARI_HERMES_LOCAL_TOKEN=...
# timeout_seconds = 120
`

	return os.WriteFile(path, []byte(defaultTOML), 0600)
}

// WriteFromStruct serialises a Config struct to config.toml using TOML encoding.
// It writes a header comment, then the TOML representation. It does NOT
// overwrite an existing file — use WriteForcedFromStruct for that.
func WriteFromStruct(cfg *Config) error {
	if err := EnsureDirs(); err != nil {
		return err
	}

	var buf bytes.Buffer
	buf.WriteString("# Hikari Configuration — generated by setup wizard\n")
	buf.WriteString("# Run 'hikari config' to modify settings interactively.\n\n")

	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return os.WriteFile(ConfigFile(), buf.Bytes(), 0600)
}
