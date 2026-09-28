package provider

import (
	"fmt"
	"sync"
)

// Router manages the currently active provider and supports live switching.
//
// Router is goroutine-safe: provider switches can be triggered from the TUI
// event loop while a Chat() call may be in flight (the caller is responsible
// for cancelling the in-flight context before switching).
type Router struct {
	mu       sync.RWMutex
	registry *Registry
	active   Provider
	name     string // name of the active provider (config key)
}

// NewRouter creates a Router backed by the given registry.
func NewRouter(registry *Registry) *Router {
	return &Router{registry: registry}
}

// SetActive switches the active provider to the one with the given name.
// The previous provider is NOT closed — the caller is responsible for that
// if needed (to allow seamless switching without dropping connections).
func (r *Router) SetActive(name string) error {
	p, err := r.registry.Get(name)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = p
	r.name = name
	return nil
}

// Active returns the currently active provider and its config name.
// Returns an error if no provider has been set yet.
func (r *Router) Active() (Provider, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.active == nil {
		return nil, "", fmt.Errorf("no active provider; run `hikari --provider <name>` or set [default] provider in config.toml")
	}
	return r.active, r.name, nil
}

// ActiveName returns the config-key name of the active provider, or "".
func (r *Router) ActiveName() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.name
}

// MustActive is like Active but panics if no provider is set.
// Use only in contexts where a provider is guaranteed.
func (r *Router) MustActive() Provider {
	p, _, err := r.Active()
	if err != nil {
		panic(err)
	}
	return p
}

// AvailableNames returns all provider names in the registry.
func (r *Router) AvailableNames() []string {
	return r.registry.List()
}
