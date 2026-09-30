// Package app is the application lifecycle coordinator.
//
// It wires together config, providers, personas, sessions, themes, and the
// TUI/CLI entry points. The App struct is the single dependency-injection root.
package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/persona"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/provider/hermes"
	"github.com/NubiMa/hikari-cli/internal/provider/ollama"
	"github.com/NubiMa/hikari-cli/internal/provider/openclaw"
	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/theme"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
)

// App holds all top-level application state and dependencies.
type App struct {
	Config   *config.Config
	Registry *provider.Registry
	Router   *provider.Router
	Personas *persona.Manager
	Sessions *session.Manager
	Themes   *theme.Manager
	Logger   *log.Logger // structured file logger
	LogPath  string      // absolute path to the log file
}

// New initialises the full application stack:
//  1. Load configuration
//  2. Register all provider constructors
//  3. Instantiate providers from config
//  4. Load personas
//  5. Initialise session manager
//  6. Initialise theme manager & apply initial styles
//  7. Set the default active provider and persona
func New() (*App, error) {
	// -- Config --
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// -- Provider Registry --
	registry := provider.NewRegistry()
	registry.RegisterType("ollama", ollama.New)
	registry.RegisterType("openclaw", openclaw.New)
	registry.RegisterType("hermes", hermes.New)

	if err := registry.Build(cfg); err != nil {
		return nil, fmt.Errorf("building providers: %w", err)
	}

	// -- Router --
	router := provider.NewRouter(registry)
	if cfg.Default.Provider != "" {
		if err := router.SetActive(cfg.Default.Provider); err != nil {
			return nil, fmt.Errorf("setting default provider: %w", err)
		}
	}

	// -- Personas --
	personas, err := persona.NewManagerDefault()
	if err != nil {
		return nil, fmt.Errorf("loading personas: %w", err)
	}
	if cfg.Default.Persona != "" {
		// Ignore error: if the configured persona doesn't exist, we fall back
		// to the default persona gracefully.
		_ = personas.SetActive(cfg.Default.Persona)
	}

	// -- Sessions --
	sessions, err := session.NewManagerDefault()
	if err != nil {
		return nil, fmt.Errorf("initialising sessions: %w", err)
	}

	// -- Themes --
	themes, err := theme.NewManagerDefault()
	if err != nil {
		return nil, fmt.Errorf("initialising themes: %w", err)
	}
	if cfg.UI.Theme != "" {
		_ = themes.SetActive(cfg.UI.Theme)
	}
	styles.ApplyTheme(themes.Active())

	// -- Logger --
	logDir := filepath.Join(config.Dir(), "logs")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return nil, fmt.Errorf("creating log dir: %w", err)
	}
	logPath := filepath.Join(logDir, "hikari.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Printf("hikari started")

	return &App{
		Config:   cfg,
		Registry: registry,
		Router:   router,
		Personas: personas,
		Sessions: sessions,
		Themes:   themes,
		Logger:   logger,
		LogPath:  logPath,
	}, nil
}

// ConnectActive connects the currently active provider.
// Should be called after New() and before starting any TUI/CLI mode.
func (a *App) ConnectActive(ctx context.Context) error {
	p, name, err := a.Router.Active()
	if err != nil {
		return err // no provider configured — not a fatal error for setup
	}
	if err := p.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to provider %q: %w", name, err)
	}
	return nil
}

// NewSession creates and saves a new session using the current active state.
func (a *App) NewSession() (*session.Session, error) {
	activeProvider := a.Router.ActiveName()
	activePersona := a.Personas.Active().Name

	providerType := ""
	if a.Config != nil {
		if pcfg, ok := a.Config.Providers[activeProvider]; ok {
			providerType = pcfg.Type
		}
	}

	model := ""
	if a.Config != nil {
		if pcfg, ok := a.Config.Providers[activeProvider]; ok {
			model = pcfg.Model
		}
	}

	return a.Sessions.Create(activeProvider, providerType, activePersona, model)
}
