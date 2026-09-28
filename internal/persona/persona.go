// Package persona manages AI persona definitions.
//
// Personas are YAML files stored under ~/.config/hikari/personas/ or
// bundled in assets/personas/. They define the character, tone, greeting,
// and system prompt injected at the start of each conversation.
package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Persona struct
// ---------------------------------------------------------------------------

// Behavior holds behavioral parameters for a persona.
type Behavior struct {
	Tone     string `yaml:"tone"`
	Language string `yaml:"language"`
}

// Persona defines an AI character's identity and behavior.
type Persona struct {
	// Name is the display name of the persona (e.g. "Nino").
	Name string `yaml:"name"`

	// Description is a short human-readable description shown in the picker.
	Description string `yaml:"description"`

	// Greeting is the first message the persona sends when a session starts.
	Greeting string `yaml:"greeting"`

	// SystemPrompt is injected as a system-role message at the start of every
	// conversation. This is how the persona shapes the AI's behavior.
	SystemPrompt string `yaml:"system_prompt"`

	// Behavior controls tone, language, and style.
	Behavior Behavior `yaml:"behavior"`

	// source tracks where this persona was loaded from (for display).
	source string
}

// IsBuiltin reports whether the persona was loaded from bundled assets.
func (p *Persona) IsBuiltin() bool { return p.source == "builtin" }

// ---------------------------------------------------------------------------
// Load
// ---------------------------------------------------------------------------

// LoadFromFile reads and parses a single persona YAML file.
func LoadFromFile(path string) (*Persona, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persona: reading %s: %w", path, err)
	}

	var p Persona
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("persona: parsing %s: %w", path, err)
	}

	if p.Name == "" {
		// Fall back to filename without extension.
		base := filepath.Base(path)
		p.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &p, nil
}

// LoadDir reads all *.yaml files from a directory and returns a map of
// lowercase-name → Persona.
func LoadDir(dir string, source string) (map[string]*Persona, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("persona: reading dir %s: %w", dir, err)
	}

	result := make(map[string]*Persona)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		p, err := LoadFromFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		p.source = source
		key := strings.ToLower(p.Name)
		result[key] = p
	}
	return result, nil
}
