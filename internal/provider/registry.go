package provider

import (
	"fmt"

	"github.com/nubiv/hikari/internal/config"
)

// ConstructorFn is a function that creates a Provider from a ProviderConfig.
// Each provider implementation registers a constructor here.
type ConstructorFn func(name string, cfg config.ProviderConfig) (Provider, error)

// Registry holds provider constructors (by type) and live instances (by name).
//
// The separation between type constructors and named instances allows the same
// provider type (e.g. "ollama") to have multiple configured instances
// (e.g. "ollama-local", "ollama-vps").
type Registry struct {
	constructors map[string]ConstructorFn // key: provider type ("ollama", "openclaw", …)
	instances    map[string]Provider      // key: provider name from config
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		constructors: make(map[string]ConstructorFn),
		instances:    make(map[string]Provider),
	}
}

// RegisterType registers a constructor for a provider type.
// This must be called at program startup before any providers are instantiated.
// Panics if the type is already registered (programming error).
func (r *Registry) RegisterType(providerType string, fn ConstructorFn) {
	if _, exists := r.constructors[providerType]; exists {
		panic(fmt.Sprintf("provider type %q already registered", providerType))
	}
	r.constructors[providerType] = fn
}

// Build instantiates all providers defined in the config and stores them
// in the registry. Returns the first error encountered.
func (r *Registry) Build(cfg *config.Config) error {
	for name, pcfg := range cfg.Providers {
		constructor, ok := r.constructors[pcfg.Type]
		if !ok {
			return fmt.Errorf("no constructor registered for provider type %q (used by %q)", pcfg.Type, name)
		}
		p, err := constructor(name, pcfg)
		if err != nil {
			return fmt.Errorf("constructing provider %q: %w", name, err)
		}
		r.instances[name] = p
	}
	return nil
}

// Get returns a provider instance by its configured name.
// Returns nil, error if not found.
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.instances[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found; check your config.toml [providers] section", name)
	}
	return p, nil
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.instances))
	for name := range r.instances {
		names = append(names, name)
	}
	return names
}

// Has reports whether a provider with the given name exists.
func (r *Registry) Has(name string) bool {
	_, ok := r.instances[name]
	return ok
}
