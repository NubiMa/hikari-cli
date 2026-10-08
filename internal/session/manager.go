package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
)

// Manager handles session persistence and lifecycle.
type Manager struct {
	dir string // sessions directory
}

// NewManager creates a Manager that stores sessions in the given directory.
// Use config.SessionsDir() for the default location.
func NewManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("session manager: creating sessions dir: %w", err)
	}
	return &Manager{dir: dir}, nil
}

// NewManagerDefault creates a Manager using the default sessions directory.
func NewManagerDefault() (*Manager, error) {
	return NewManager(config.SessionsDir())
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

// Create creates a new Session with a generated ID and saves it.
func (m *Manager) Create(providerName, providerType, persona, model string) (*Session, error) {
	id := generateID()
	now := time.Now()
	s := &Session{
		ID:           id,
		Provider:     providerName,
		ProviderType: providerType,
		Persona:      persona,
		Model:        model,
		Messages:     []Message{},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := m.Save(s); err != nil {
		return nil, err
	}
	return s, nil
}

// Load reads a session by ID from disk.
func (m *Manager) Load(id string) (*Session, error) {
	path := m.pathFor(id)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("session load %s: %w", id, err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("session load %s: decoding: %w", id, err)
	}
	return &s, nil
}

// Save writes the session to disk.
func (m *Manager) Save(s *Session) error {
	s.AutoTitle()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session save %s: encoding: %w", s.ID, err)
	}
	path := m.pathFor(s.ID)
	return os.WriteFile(path, data, 0600)
}

// Delete removes a session file.
func (m *Manager) Delete(id string) error {
	path := m.pathFor(id)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("session delete %s: %w", id, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

// ListEntry is a lightweight summary used for displaying session lists.
type ListEntry struct {
	ID           string
	Title        string
	Provider     string
	Persona      string
	Model        string
	MessageCount int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Rename updates the title of an existing session.
func (m *Manager) Rename(id string, newTitle string) error {
	s, err := m.Load(id)
	if err != nil {
		return err
	}
	s.Title = strings.TrimSpace(newTitle)
	s.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("session rename %s: encoding: %w", s.ID, err)
	}
	path := m.pathFor(s.ID)
	return os.WriteFile(path, data, 0600)
}

// List returns all sessions sorted by UpdatedAt descending (most recent first).
func (m *Manager) List() ([]ListEntry, error) {
	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("session list: %w", err)
	}

	var result []ListEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".json")
		s, err := m.Load(id)
		if err != nil {
			continue // skip corrupted sessions
		}
		result = append(result, ListEntry{
			ID:           s.ID,
			Title:        s.Title,
			Provider:     s.Provider,
			Persona:      s.Persona,
			Model:        s.Model,
			MessageCount: len(s.Messages),
			CreatedAt:    s.CreatedAt,
			UpdatedAt:    s.UpdatedAt,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

// Search filters sessions whose title, metadata, or message content matches query.
func (m *Manager) Search(query string) ([]ListEntry, error) {
	all, err := m.List()
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return all, nil
	}

	var matched []ListEntry
	for _, entry := range all {
		if strings.Contains(strings.ToLower(entry.Title), q) ||
			strings.Contains(strings.ToLower(entry.Provider), q) ||
			strings.Contains(strings.ToLower(entry.Persona), q) ||
			strings.Contains(strings.ToLower(entry.Model), q) {
			matched = append(matched, entry)
			continue
		}
		s, err := m.Load(entry.ID)
		if err != nil {
			continue
		}
		msgMatch := false
		for _, msg := range s.Messages {
			if strings.Contains(strings.ToLower(msg.Content), q) {
				msgMatch = true
				break
			}
		}
		if msgMatch {
			matched = append(matched, entry)
		}
	}
	return matched, nil
}

// ExportMarkdown writes a formatted markdown transcript of the session to outPath.
func (m *Manager) ExportMarkdown(id string, outPath string) error {
	s, err := m.Load(id)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", s.Title))
	b.WriteString(fmt.Sprintf("- **Date:** %s\n", s.CreatedAt.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("- **Provider:** %s\n", s.Provider))
	b.WriteString(fmt.Sprintf("- **Persona:** %s\n", s.Persona))
	if s.Model != "" {
		b.WriteString(fmt.Sprintf("- **Model:** %s\n", s.Model))
	}
	b.WriteString("\n---\n\n")

	for _, msg := range s.Messages {
		roleName := "User"
		switch msg.Role {
		case "assistant":
			roleName = s.Persona
			if roleName == "" {
				roleName = "Hikari"
			}
		case "system":
			roleName = "System"
		}
		b.WriteString(fmt.Sprintf("### %s (%s)\n\n", roleName, msg.Timestamp.Format("15:04:05")))
		b.WriteString(msg.Content)
		b.WriteString("\n\n")
	}

	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating export dir: %w", err)
	}
	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m *Manager) pathFor(id string) string {
	return filepath.Join(m.dir, id+".json")
}

// generateID creates a time-based unique ID.
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
