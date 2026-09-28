// Package session manages conversation sessions and their persistence.
//
// A Session is a single conversation thread. It records all messages,
// the active provider and persona at the time of creation, and timestamps.
// Sessions are stored as JSON files under ~/.config/hikari/sessions/.
package session

import (
	"time"

	"github.com/NubiMa/hikari-cli/internal/provider"
)

// ---------------------------------------------------------------------------
// Core types
// ---------------------------------------------------------------------------

// Message is a single turn in a conversation, stored as part of a Session.
type Message struct {
	Role      provider.Role `json:"role"`
	Content   string        `json:"content"`
	Timestamp time.Time     `json:"timestamp"`
}

// Session represents a single conversation thread.
type Session struct {
	// ID is a UUID-style identifier, used as the filename.
	ID string `json:"id"`

	// Title is a human-readable name for the session.
	// Auto-generated from the first user message if not set.
	Title string `json:"title"`

	// Provider is the config-key name of the provider used in this session.
	Provider string `json:"provider"`

	// ProviderType is the provider type (e.g. "ollama") for future-proofing
	// when provider names change.
	ProviderType string `json:"provider_type"`

	// Persona is the name of the active persona.
	Persona string `json:"persona"`

	// Model is the model used in this session (may be empty for agent providers).
	Model string `json:"model"`

	// Messages is the ordered conversation history.
	Messages []Message `json:"messages"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// ProviderHistory returns the session messages in the format expected by the
// Provider.Chat() method (without timestamps, which are Hikari-internal).
func (s *Session) ProviderHistory() []provider.Message {
	msgs := make([]provider.Message, 0, len(s.Messages))
	for _, m := range s.Messages {
		msgs = append(msgs, provider.Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return msgs
}

// AddMessage appends a message to the session and updates UpdatedAt.
func (s *Session) AddMessage(role provider.Role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
	s.UpdatedAt = time.Now()
}

// AutoTitle sets Title from the first user message if Title is still empty.
func (s *Session) AutoTitle() {
	if s.Title != "" {
		return
	}
	for _, m := range s.Messages {
		if m.Role == provider.RoleUser {
			title := m.Content
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			s.Title = title
			return
		}
	}
}
