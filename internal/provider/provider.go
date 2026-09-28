// Package provider defines the abstraction layer between Hikari and AI backends.
//
// Every AI backend (OpenClaw, Hermes, Ollama, …) is represented as a Provider.
// The Provider interface is intentionally minimal: it covers the lowest common
// denominator of all backends. Richer capabilities are exposed through the
// Capabilities struct, which allows TUI and CLI code to conditionally enable
// features without risking runtime panics from calling unsupported methods.
package provider

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

// EventType classifies events coming from a provider's streaming channel.
type EventType string

const (
	// EventToken is a partial text chunk during streaming generation.
	EventToken EventType = "token"

	// EventDone signals that the model has finished generating.
	EventDone EventType = "done"

	// EventError carries an error from the provider.
	EventError EventType = "error"

	// EventStatus is an informational status message from the provider
	// (e.g. "thinking…", "running tool X").
	EventStatus EventType = "status"
)

// Event is a single message emitted by a provider's streaming channel.
type Event struct {
	Type    EventType
	Content string // text token or status message
	Error   error  // non-nil only when Type == EventError
	// Meta is optional provider-specific metadata (tool name, model, etc.).
	Meta map[string]any
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model represents an AI model available on a provider.
type Model struct {
	ID          string
	Name        string
	Description string
	Size        int64 // bytes, if known (e.g. Ollama local models)
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

// ProviderStatus describes the health of a provider connection.
type ProviderStatus struct {
	Connected bool
	Latency   time.Duration
	Message   string // human-readable status
}

// ---------------------------------------------------------------------------
// Provider interface
// ---------------------------------------------------------------------------

// Provider is the core abstraction for all AI backends.
//
// Implementations must be safe for concurrent use from a single goroutine
// (i.e. the TUI event loop). Chat() may be called while a previous call's
// channel is still being drained — implementations should handle this cleanly
// by cancelling the previous request via the context.
type Provider interface {
	// Connect establishes a connection to the backend and validates credentials.
	// It must be called before any other method.
	Connect(ctx context.Context) error

	// Close releases all resources held by the provider.
	Close() error

	// Chat sends a message and returns a channel of streaming events.
	// The caller is responsible for draining the channel until it is closed.
	// Cancelling ctx will cause the provider to close the channel and stop
	// generation.
	//
	// history is the conversation history to send for context.
	// message is the new user message.
	Chat(ctx context.Context, history []Message, message string) (<-chan Event, error)

	// Models returns the list of models available on this provider.
	// Returns an error if the provider does not support model listing.
	Models(ctx context.Context) ([]Model, error)

	// Status returns the current health status of this provider.
	Status(ctx context.Context) (ProviderStatus, error)

	// Capabilities returns the set of features this provider supports.
	// Callers must check Capabilities before calling optional features.
	Capabilities() Capabilities

	// Name returns the human-readable name of this provider implementation.
	Name() string
}

// ---------------------------------------------------------------------------
// Message (shared between provider and session)
// ---------------------------------------------------------------------------

// Role identifies who sent a message in a conversation.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message is a single turn in a conversation history.
type Message struct {
	Role    Role
	Content string
}
