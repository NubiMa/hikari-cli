package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// SessionManager Messages
type SessionSwitchMsg struct {
	ID string
}

type SessionNewMsg struct{}

type SessionRenameMsg struct {
	ID       string
	NewTitle string
}

type SessionDeleteMsg struct {
	ID string
}

type SessionManagerCloseMsg struct{}

type sessionMgrState int

const (
	mgrStateList sessionMgrState = iota
	mgrStateConfirmDelete
	mgrStateRename
)

// SessionManager is a dedicated interactive modal for managing session lifecycle.
type SessionManager struct {
	Entries     []session.ListEntry
	ActiveID    string
	cursor      int
	Width       int
	Height      int
	state       sessionMgrState
	renameInput textinput.Model
}

// NewSessionManager creates a new SessionManager component.
func NewSessionManager(entries []session.ListEntry, activeID string, width, height int) SessionManager {
	ti := textinput.New()
	ti.Placeholder = "Enter session title..."
	ti.CharLimit = 64

	// Find cursor matching active session
	cursor := 0
	for i, e := range entries {
		if e.ID == activeID {
			cursor = i
			break
		}
	}

	return SessionManager{
		Entries:     entries,
		ActiveID:    activeID,
		cursor:      cursor,
		Width:       width,
		Height:      height,
		state:       mgrStateList,
		renameInput: ti,
	}
}

func (s SessionManager) Init() tea.Cmd {
	return nil
}

func (s SessionManager) Update(msg tea.Msg) (SessionManager, tea.Cmd) {
	switch s.state {
	case mgrStateRename:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				val := strings.TrimSpace(s.renameInput.Value())
				s.state = mgrStateList
				if val != "" && len(s.Entries) > s.cursor {
					id := s.Entries[s.cursor].ID
					s.Entries[s.cursor].Title = val
					return s, func() tea.Msg {
						return SessionRenameMsg{ID: id, NewTitle: val}
					}
				}
				return s, nil
			case "esc":
				s.state = mgrStateList
				return s, nil
			}
		}
		var cmd tea.Cmd
		s.renameInput, cmd = s.renameInput.Update(msg)
		return s, cmd

	case mgrStateConfirmDelete:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y", "enter":
				s.state = mgrStateList
				if len(s.Entries) > s.cursor {
					deletedID := s.Entries[s.cursor].ID
					// Remove from local list
					s.Entries = append(s.Entries[:s.cursor], s.Entries[s.cursor+1:]...)
					if s.cursor >= len(s.Entries) && s.cursor > 0 {
						s.cursor--
					}
					return s, func() tea.Msg {
						return SessionDeleteMsg{ID: deletedID}
					}
				}
				return s, nil
			case "n", "N", "esc", "q":
				s.state = mgrStateList
				return s, nil
			}
		}
		return s, nil

	case mgrStateList:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if s.cursor > 0 {
					s.cursor--
				} else if len(s.Entries) > 0 {
					s.cursor = len(s.Entries) - 1
				}
			case "down", "j":
				if s.cursor < len(s.Entries)-1 {
					s.cursor++
				} else {
					s.cursor = 0
				}
			case "enter":
				if len(s.Entries) > s.cursor {
					id := s.Entries[s.cursor].ID
					return s, func() tea.Msg {
						return SessionSwitchMsg{ID: id}
					}
				}
			case "n":
				return s, func() tea.Msg {
					return SessionNewMsg{}
				}
			case "r":
				if len(s.Entries) > s.cursor {
					s.state = mgrStateRename
					s.renameInput.SetValue(s.Entries[s.cursor].Title)
					s.renameInput.Focus()
					return s, textinput.Blink
				}
			case "d":
				if len(s.Entries) > s.cursor {
					s.state = mgrStateConfirmDelete
					return s, nil
				}
			case "esc", "q", "ctrl+c":
				return s, func() tea.Msg {
					return SessionManagerCloseMsg{}
				}
			}
		}
	}

	return s, nil
}

func (s SessionManager) View() string {
	boxW := s.Width - 10
	if boxW > 74 {
		boxW = 74
	}
	if boxW < 42 {
		boxW = 42
	}

	var b strings.Builder
	b.WriteString(styles.SelectorTitle.Render("◈ SESSION MANAGER"))
	b.WriteString("\n\n")

	if len(s.Entries) == 0 {
		b.WriteString(styles.Muted.Render("  No saved sessions found."))
		b.WriteString("\n\n")
		b.WriteString(styles.Muted.Render("  [n] Create New Session  ·  [Esc] Close"))
		return styles.SelectorBox.Width(boxW).Render(b.String())
	}

	// In delete confirmation mode
	if s.state == mgrStateConfirmDelete {
		target := s.Entries[s.cursor]
		title := target.Title
		if len(title) > 40 {
			title = title[:37] + "..."
		}
		confirmText := fmt.Sprintf("%s\n\nDelete session %q?\n%s\n\n%s",
			styles.Warning.Render("⚠ CONFIRM DELETION"),
			title,
			styles.Muted.Render("This action cannot be undone."),
			styles.Error.Render(" [y] Confirm Delete ") + "  " + styles.Muted.Render("[n/Esc] Cancel"),
		)
		b.WriteString(confirmText)
		return styles.SelectorBox.Width(boxW).Render(b.String())
	}

	// In rename mode
	if s.state == mgrStateRename {
		b.WriteString(styles.Bold.Render("Rename Session:"))
		b.WriteString("\n\n")
		b.WriteString(s.renameInput.View())
		b.WriteString("\n\n")
		b.WriteString(styles.Muted.Render(" [Enter] Save  ·  [Esc] Cancel"))
		return styles.SelectorBox.Width(boxW).Render(b.String())
	}

	// Calculate visible slice for pagination
	maxVisible := 7
	startIdx := 0
	if s.cursor >= maxVisible {
		startIdx = s.cursor - maxVisible + 1
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(s.Entries) {
		endIdx = len(s.Entries)
	}

	for i := startIdx; i < endIdx; i++ {
		entry := s.Entries[i]
		isCursor := i == s.cursor
		isActive := s.ActiveID != "" && entry.ID == s.ActiveID

		prefix := "  "
		if isCursor {
			prefix = "› "
		}

		title := entry.Title
		if title == "" {
			title = "Untitled Session"
		}
		maxTitleW := boxW - 18
		if len(title) > maxTitleW && maxTitleW > 10 {
			title = title[:maxTitleW-3] + "..."
		}

		activeTag := ""
		if isActive {
			activeTag = " " + styles.Success.Render("[Active]")
		}

		line := prefix + title + activeTag
		if isCursor {
			line = styles.SelectorItemSelected.Render(line)
		} else {
			line = styles.SelectorItem.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")

		// Subtitle: Provider, Model, Message count, Updated time
		var metaParts []string
		if entry.Provider != "" {
			metaParts = append(metaParts, entry.Provider)
		}
		if entry.Model != "" {
			metaParts = append(metaParts, entry.Model)
		}
		if entry.MessageCount > 0 {
			metaParts = append(metaParts, fmt.Sprintf("%d msgs", entry.MessageCount))
		}
		metaParts = append(metaParts, formatRelativeTime(entry.UpdatedAt))

		sub := strings.Join(metaParts, " · ")
		b.WriteString(styles.Muted.Render("    " + sub))
		b.WriteString("\n\n")
	}

	// Footer hotkeys
	b.WriteString(styles.Muted.Render(" [Enter] Switch  ·  [n] New  ·  [r] Rename  ·  [d] Delete  ·  [Esc] Close"))

	return styles.SelectorBox.Width(boxW).Render(b.String())
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	}
	if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	}
	if diff < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
	return t.Format("Jan 2")
}
