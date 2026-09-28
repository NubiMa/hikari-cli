package session_test

import (
	"testing"
	"time"

	"github.com/nubiv/hikari/internal/provider"
	"github.com/nubiv/hikari/internal/session"
)

func TestSessionCRUD(t *testing.T) {
	dir := t.TempDir()
	mgr, err := session.NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	// Create
	s, err := mgr.Create("ollama-local", "ollama", "nino", "llama3.2")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID == "" {
		t.Error("expected non-empty ID")
	}

	// Add messages
	s.AddMessage(provider.RoleUser, "Hello!")
	s.AddMessage(provider.RoleAssistant, "Hi there!")
	if err := mgr.Save(s); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load
	loaded, err := mgr.Load(s.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Hello!" {
		t.Errorf("wrong content: %q", loaded.Messages[0].Content)
	}

	// AutoTitle
	if loaded.Title == "" {
		t.Error("expected title to be auto-set from first user message")
	}

	// List
	entries, err := mgr.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}

	// Delete
	if err := mgr.Delete(s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	entries, _ = mgr.List()
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after delete, got %d", len(entries))
	}
}

func TestHistoryGroupByDate(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	twoDaysAgo := now.AddDate(0, 0, -2)

	entries := []session.ListEntry{
		{ID: "1", Title: "Today 1", UpdatedAt: now},
		{ID: "2", Title: "Today 2", UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "3", Title: "Yesterday", UpdatedAt: yesterday},
		{ID: "4", Title: "Old", UpdatedAt: twoDaysAgo},
	}

	groups := session.GroupByDate(entries)

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups, got %d", len(groups))
	}
	if groups[0].Label != "Today" {
		t.Errorf("expected 'Today', got %q", groups[0].Label)
	}
	if len(groups[0].Sessions) != 2 {
		t.Errorf("expected 2 sessions in Today, got %d", len(groups[0].Sessions))
	}
	if groups[1].Label != "Yesterday" {
		t.Errorf("expected 'Yesterday', got %q", groups[1].Label)
	}
}

func TestProviderHistory(t *testing.T) {
	s := &session.Session{}
	s.AddMessage(provider.RoleUser, "ping")
	s.AddMessage(provider.RoleAssistant, "pong")

	history := s.ProviderHistory()
	if len(history) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(history))
	}
	if history[0].Role != provider.RoleUser {
		t.Error("wrong role for first message")
	}
}
