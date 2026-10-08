package components

import (
	"fmt"
	"strings"

	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HistoryBrowser Messages
type HistoryResumeMsg struct {
	ID string
}

type HistoryExportMsg struct {
	ID string
}

type HistoryBrowserCloseMsg struct{}

// HistoryBrowser provides a rich dual-pane chronological conversation browser with live transcript preview.
type HistoryBrowser struct {
	allEntries    []session.ListEntry
	filtered      []session.ListEntry
	groups        []session.HistoryGroup
	cursor        int
	activeSessID  string
	loadedSession *session.Session
	sessionLoader func(id string) (*session.Session, error)
	searchInput   textinput.Model
	isSearching   bool
	focusPreview  bool
	previewScroll int
	Width         int
	Height        int
}

// NewHistoryBrowser constructs the HistoryBrowser component.
func NewHistoryBrowser(
	entries []session.ListEntry,
	activeSessID string,
	loader func(id string) (*session.Session, error),
	width, height int,
) HistoryBrowser {
	ti := textinput.New()
	ti.Placeholder = "Type to search past conversations..."
	ti.CharLimit = 60

	hb := HistoryBrowser{
		allEntries:    entries,
		filtered:      entries,
		activeSessID:  activeSessID,
		sessionLoader: loader,
		searchInput:   ti,
		Width:         width,
		Height:        height,
	}

	hb.rebuildGroups()

	// Initial cursor position
	for i, e := range hb.filtered {
		if e.ID == activeSessID {
			hb.cursor = i
			break
		}
	}

	hb.loadCurrentPreview()
	return hb
}

func (h *HistoryBrowser) rebuildGroups() {
	h.groups = session.GroupByDate(h.filtered)
}

func (h *HistoryBrowser) loadCurrentPreview() {
	h.previewScroll = 0
	if len(h.filtered) == 0 || h.cursor >= len(h.filtered) {
		h.loadedSession = nil
		return
	}
	id := h.filtered[h.cursor].ID
	if h.sessionLoader != nil {
		s, err := h.sessionLoader(id)
		if err == nil {
			h.loadedSession = s
			return
		}
	}
	h.loadedSession = nil
}

func (h HistoryBrowser) Init() tea.Cmd {
	return nil
}

func (h HistoryBrowser) Update(msg tea.Msg) (HistoryBrowser, tea.Cmd) {
	if h.isSearching {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "esc":
				h.isSearching = false
				h.searchInput.Blur()
				return h, nil
			}
		}
		var cmd tea.Cmd
		h.searchInput, cmd = h.searchInput.Update(msg)
		// Filter list
		query := strings.ToLower(strings.TrimSpace(h.searchInput.Value()))
		if query == "" {
			h.filtered = h.allEntries
		} else {
			var res []session.ListEntry
			for _, e := range h.allEntries {
				if strings.Contains(strings.ToLower(e.Title), query) ||
					strings.Contains(strings.ToLower(e.Provider), query) ||
					strings.Contains(strings.ToLower(e.Persona), query) ||
					strings.Contains(strings.ToLower(e.Model), query) {
					res = append(res, e)
				}
			}
			h.filtered = res
		}
		h.cursor = 0
		h.rebuildGroups()
		h.loadCurrentPreview()
		return h, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			h.isSearching = true
			h.searchInput.Focus()
			return h, textinput.Blink

		case "tab":
			h.focusPreview = !h.focusPreview
			return h, nil

		case "up", "k":
			if h.focusPreview {
				if h.previewScroll > 0 {
					h.previewScroll--
				}
			} else {
				if h.cursor > 0 {
					h.cursor--
					h.loadCurrentPreview()
				} else if len(h.filtered) > 0 {
					h.cursor = len(h.filtered) - 1
					h.loadCurrentPreview()
				}
			}

		case "down", "j":
			if h.focusPreview {
				h.previewScroll++
			} else {
				if h.cursor < len(h.filtered)-1 {
					h.cursor++
					h.loadCurrentPreview()
				} else if len(h.filtered) > 0 {
					h.cursor = 0
					h.loadCurrentPreview()
				}
			}

		case "pgup":
			if h.previewScroll >= 5 {
				h.previewScroll -= 5
			} else {
				h.previewScroll = 0
			}

		case "pgdown":
			h.previewScroll += 5

		case "enter":
			if len(h.filtered) > h.cursor {
				id := h.filtered[h.cursor].ID
				return h, func() tea.Msg {
					return HistoryResumeMsg{ID: id}
				}
			}

		case "e":
			if len(h.filtered) > h.cursor {
				id := h.filtered[h.cursor].ID
				return h, func() tea.Msg {
					return HistoryExportMsg{ID: id}
				}
			}

		case "esc", "q", "ctrl+c":
			return h, func() tea.Msg {
				return HistoryBrowserCloseMsg{}
			}
		}
	}

	return h, nil
}

func (h HistoryBrowser) View() string {
	boxW := h.Width - 6
	if boxW > 110 {
		boxW = 110
	}
	if boxW < 50 {
		boxW = 50
	}

	boxH := h.Height - 6
	if boxH < 14 {
		boxH = 14
	}

	innerH := boxH - 4
	leftW := int(float64(boxW) * 0.40)
	if leftW < 24 {
		leftW = 24
	}
	rightW := boxW - leftW - 3
	if rightW < 22 {
		rightW = 22
	}

	// 1. Left Pane: Search bar + Date-grouped list
	var left strings.Builder
	left.WriteString(styles.SelectorTitle.Render("◈ SESSIONS"))
	left.WriteString("\n")

	// Search field
	searchPrefix := "[/] Search: "
	if h.isSearching {
		left.WriteString(styles.Bold.Render("Search: ") + h.searchInput.View() + "\n\n")
	} else if h.searchInput.Value() != "" {
		left.WriteString(styles.Muted.Render(searchPrefix+h.searchInput.Value()) + "\n\n")
	} else {
		left.WriteString(styles.Muted.Render(searchPrefix+"(press / to filter)") + "\n\n")
	}

	if len(h.filtered) == 0 {
		left.WriteString(styles.Muted.Render("  No matching conversations."))
	} else {
		// Flatten groups with headers
		flatIdx := 0
		maxLines := innerH - 4
		linesCount := 0

		for _, grp := range h.groups {
			if linesCount >= maxLines {
				break
			}
			left.WriteString(styles.Bold.Render("▼ " + grp.Label))
			left.WriteString("\n")
			linesCount++

			for _, entry := range grp.Sessions {
				if linesCount >= maxLines {
					break
				}
				isCursor := flatIdx == h.cursor
				isActive := entry.ID == h.activeSessID

				prefix := "  "
				if isCursor {
					prefix = "› "
				}

				title := entry.Title
				if title == "" {
					title = "Untitled"
				}
				maxT := leftW - 8
				if len(title) > maxT && maxT > 4 {
					title = title[:maxT-2] + "…"
				}

				activeTag := ""
				if isActive {
					activeTag = " *"
				}

				itemLine := prefix + title + activeTag
				if isCursor && !h.focusPreview {
					itemLine = styles.SelectorItemSelected.Render(itemLine)
				} else if isCursor {
					itemLine = styles.SelectorItemActive.Render(itemLine)
				} else {
					itemLine = styles.SelectorItem.Render(itemLine)
				}
				left.WriteString(itemLine + "\n")
				linesCount++
				flatIdx++
			}
		}
	}

	leftBox := lipgloss.NewStyle().
		Width(leftW).
		Height(innerH).
		Render(left.String())

	// 2. Right Pane: Transcript preview
	var right strings.Builder
	if h.loadedSession == nil {
		right.WriteString(styles.Muted.Render("Select a conversation to preview transcript."))
	} else {
		s := h.loadedSession
		right.WriteString(styles.Bold.Render("◈ " + s.Title))
		right.WriteString("\n")
		meta := fmt.Sprintf("%s · %s · %d turns · %s",
			s.Provider, s.Persona, len(s.Messages), s.CreatedAt.Format("Jan 2 15:04"))
		right.WriteString(styles.Muted.Render(meta))
		right.WriteString("\n\n")

		var previewLines []string
		for _, m := range s.Messages {
			roleLabel := "You"
			roleStyle := styles.UserHeader
			if m.Role == provider.RoleAssistant {
				roleLabel = s.Persona
				if roleLabel == "" {
					roleLabel = "Hikari"
				}
				roleStyle = styles.AssistantHeader
			} else if m.Role == provider.RoleSystem {
				roleLabel = "System"
				roleStyle = styles.SystemHeader
			}

			previewLines = append(previewLines, roleStyle.Render("["+roleLabel+"] "+m.Timestamp.Format("15:04")))
			// Wrap message lines
			msgContent := strings.TrimSpace(m.Content)
			if msgContent == "" {
				msgContent = "(empty message)"
			}
			for _, line := range strings.Split(msgContent, "\n") {
				previewLines = append(previewLines, "  "+line)
			}
			previewLines = append(previewLines, "")
		}

		// Apply scroll
		total := len(previewLines)
		maxVis := innerH - 4
		if maxVis < 3 {
			maxVis = 3
		}
		start := h.previewScroll
		if start >= total {
			start = total - 1
		}
		if start < 0 {
			start = 0
		}
		end := start + maxVis
		if end > total {
			end = total
		}

		for i := start; i < end; i++ {
			right.WriteString(previewLines[i] + "\n")
		}

		if total > maxVis {
			scrollInfo := fmt.Sprintf("Scroll: %d/%d (Tab then ↑/↓/PgUp/PgDn)", start+1, total)
			right.WriteString(styles.Muted.Render(scrollInfo))
		}
	}

	rightBorder := styles.ColorDim
	if h.focusPreview {
		rightBorder = styles.ColorPrimary
	}
	rightBox := lipgloss.NewStyle().
		Width(rightW).
		Height(innerH).
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(rightBorder).
		PaddingLeft(1).
		Render(right.String())

	content := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)

	// Footer
	footer := styles.Muted.Render(" [↑/↓] Navigate  ·  [Tab] Toggle Pane  ·  [/] Search  ·  [Enter] Resume  ·  [e] Export  ·  [Esc] Close")

	modalContent := lipgloss.JoinVertical(lipgloss.Left, content, "\n"+footer)
	return styles.SelectorBox.Width(boxW).Render(modalContent)
}
