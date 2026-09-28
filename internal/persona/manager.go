package persona

import (
	"fmt"
	"sort"
	"strings"

	"github.com/NubiMa/hikari-cli/internal/config"
)

// Manager holds all loaded personas and tracks the active one.
type Manager struct {
	personas map[string]*Persona // key: lowercase name
	active   *Persona
}

// NewManager loads personas from bundled assets (builtinDir) and user config
// (userDir). User personas override builtin ones with the same name.
func NewManager(builtinDir, userDir string) (*Manager, error) {
	m := &Manager{
		personas: make(map[string]*Persona),
	}

	// Load builtin personas first.
	builtins, err := LoadDir(builtinDir, "builtin")
	if err != nil {
		return nil, err
	}
	for k, p := range builtins {
		m.personas[k] = p
	}

	// User personas override builtins.
	userPersonas, err := LoadDir(userDir, "user")
	if err != nil {
		return nil, err
	}
	for k, p := range userPersonas {
		m.personas[k] = p
	}

	// Ensure there is always at least a default fallback.
	if len(m.personas) == 0 {
		m.personas["default"] = defaultPersona()
	}

	return m, nil
}

// NewManagerDefault creates a Manager using standard asset and config paths.
func NewManagerDefault() (*Manager, error) {
	return NewManager("assets/personas", config.PersonasDir())
}

// SetActive sets the active persona by name (case-insensitive).
func (m *Manager) SetActive(name string) error {
	key := strings.ToLower(name)
	p, ok := m.personas[key]
	if !ok {
		return fmt.Errorf("persona %q not found", name)
	}
	m.active = p
	return nil
}

// Active returns the currently active persona.
// If none has been set, returns the "default" persona.
func (m *Manager) Active() *Persona {
	if m.active != nil {
		return m.active
	}
	if p, ok := m.personas["default"]; ok {
		return p
	}
	// Absolute fallback.
	return defaultPersona()
}

// List returns all persona names sorted alphabetically.
func (m *Manager) List() []string {
	names := make([]string, 0, len(m.personas))
	for _, p := range m.personas {
		names = append(names, p.Name)
	}
	sort.Strings(names)
	return names
}

// Get returns a persona by name (case-insensitive).
func (m *Manager) Get(name string) (*Persona, bool) {
	p, ok := m.personas[strings.ToLower(name)]
	return p, ok
}

// All returns all personas as a slice, sorted by name.
func (m *Manager) All() []*Persona {
	all := make([]*Persona, 0, len(m.personas))
	for _, p := range m.personas {
		all = append(all, p)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Name < all[j].Name
	})
	return all
}

// defaultPersona returns a minimal fallback persona used when none are loaded.
func defaultPersona() *Persona {
	return &Persona{
		Name:        "Default",
		Description: "A helpful AI assistant",
		Greeting:    "Hello! How can I help you today?",
		SystemPrompt: "You are a helpful AI assistant. " +
			"Be concise, accurate, and friendly.",
		Behavior: Behavior{
			Tone:     "neutral",
			Language: "English",
		},
		source: "builtin",
	}
}
