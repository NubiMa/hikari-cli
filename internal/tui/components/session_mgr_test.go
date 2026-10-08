package components_test

import (
	"testing"
	"time"

	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/tui/components"
	tea "github.com/charmbracelet/bubbletea"
)

func TestSessionManagerNavigationAndSwitch(t *testing.T) {
	entries := []session.ListEntry{
		{ID: "sess-1", Title: "Session One", UpdatedAt: time.Now()},
		{ID: "sess-2", Title: "Session Two", UpdatedAt: time.Now().Add(-1 * time.Hour)},
	}

	mgr := components.NewSessionManager(entries, "sess-1", 80, 24)

	// Initial view rendering
	view := mgr.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}

	// Move cursor down
	mgr, _ = mgr.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Enter to switch
	mgr, cmd := mgr.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command on Enter")
	}

	msg := cmd()
	switchMsg, ok := msg.(components.SessionSwitchMsg)
	if !ok {
		t.Fatalf("expected SessionSwitchMsg, got %T", msg)
	}
	if switchMsg.ID != "sess-2" {
		t.Errorf("expected sess-2, got %s", switchMsg.ID)
	}
}

func TestSessionManagerActions(t *testing.T) {
	entries := []session.ListEntry{
		{ID: "sess-1", Title: "Original Title", UpdatedAt: time.Now()},
	}

	mgr := components.NewSessionManager(entries, "sess-1", 80, 24)

	// Test 'n' for new session
	mgr, cmd := mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cmd == nil {
		t.Fatal("expected command on 'n'")
	}
	if _, ok := cmd().(components.SessionNewMsg); !ok {
		t.Errorf("expected SessionNewMsg")
	}

	// Test 'r' for rename
	mgr, _ = mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	// Type Enter in rename mode
	mgr, renameCmd := mgr.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if renameCmd != nil {
		msg := renameCmd()
		if rMsg, ok := msg.(components.SessionRenameMsg); ok {
			if rMsg.ID != "sess-1" || rMsg.NewTitle != "Original Title" {
				t.Errorf("unexpected rename msg: %+v", rMsg)
			}
		}
	}

	// Test 'd' for delete
	mgr, _ = mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	// Confirm delete with 'y'
	mgr, delCmd := mgr.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if delCmd == nil {
		t.Fatal("expected delete command")
	}
	if dMsg, ok := delCmd().(components.SessionDeleteMsg); !ok || dMsg.ID != "sess-1" {
		t.Errorf("unexpected delete msg: %+v", delCmd())
	}
}
