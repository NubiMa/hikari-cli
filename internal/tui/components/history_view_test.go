package components_test

import (
	"testing"
	"time"

	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestHistoryBrowserPreviewAndResume(t *testing.T) {
	now := time.Now()
	entries := []session.ListEntry{
		{ID: "sess-1", Title: "Session One", UpdatedAt: now},
		{ID: "sess-2", Title: "Session Two", UpdatedAt: now.Add(-2 * time.Hour)},
	}

	loader := func(id string) (*session.Session, error) {
		s := &session.Session{
			ID:        id,
			Title:     "Title for " + id,
			Provider:  "ollama",
			Persona:   "hikari",
			CreatedAt: now,
		}
		s.AddMessage(provider.RoleUser, "What is Hikari?")
		s.AddMessage(provider.RoleAssistant, "Hikari is a modern AI terminal.")
		return s, nil
	}

	hb := components.NewHistoryBrowser(entries, "sess-1", loader, 80, 24)

	// View rendering
	v := hb.View()
	if v == "" {
		t.Fatal("expected non-empty history browser view")
	}

	// Move cursor down to sess-2
	hb, _ = hb.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Enter to resume
	hb, cmd := hb.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}

	msg := cmd()
	resumeMsg, ok := msg.(components.HistoryResumeMsg)
	if !ok {
		t.Fatalf("expected HistoryResumeMsg, got %T", msg)
	}
	if resumeMsg.ID != "sess-2" {
		t.Errorf("expected sess-2, got %s", resumeMsg.ID)
	}

	// Press 'e' to export
	_, exportCmd := hb.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if exportCmd == nil {
		t.Fatal("expected command on 'e'")
	}
	if eMsg, ok := exportCmd().(components.HistoryExportMsg); !ok || eMsg.ID != "sess-2" {
		t.Errorf("unexpected export msg: %+v", exportCmd())
	}
}
