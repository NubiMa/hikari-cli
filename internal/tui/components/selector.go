package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nubiv/hikari/internal/tui/styles"
)

// SelectorItem represents a single option in a Selector.
type SelectorItem struct {
	ID    string // unique identifier
	Label string // display name
	Sub   string // optional subtitle
	Badge string // optional pill tag
}

// SelectorChosenMsg is sent when the user selects an item.
type SelectorChosenMsg struct {
	Context string       // e.g. "provider", "persona", "model", "theme"
	Item    SelectorItem
}

// SelectorCancelledMsg is sent when the user presses Escape.
type SelectorCancelledMsg struct{}

// Selector is a reusable modal picker used for /provider, /persona, /model, /theme, /session.
type Selector struct {
	Title    string
	Context  string // passed back in SelectorChosenMsg
	Items    []SelectorItem
	ActiveID string // ID of currently active element to mark
	cursor   int
	Width    int
	Height   int
}

func (s Selector) Init() tea.Cmd { return nil }

func (s Selector) Update(msg tea.Msg) (Selector, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			} else {
				s.cursor = len(s.Items) - 1
			}
		case "down", "j":
			if s.cursor < len(s.Items)-1 {
				s.cursor++
			} else {
				s.cursor = 0
			}
		case "enter":
			if len(s.Items) > 0 {
				item := s.Items[s.cursor]
				return s, func() tea.Msg {
					return SelectorChosenMsg{Context: s.Context, Item: item}
				}
			}
		case "esc", "q", "ctrl+c":
			return s, func() tea.Msg { return SelectorCancelledMsg{} }
		}
	}
	return s, nil
}

func (s Selector) View() string {
	boxW := s.Width - 12
	if boxW > 68 {
		boxW = 68
	}
	if boxW < 36 {
		boxW = 36
	}

	if len(s.Items) == 0 {
		emptyContent := fmt.Sprintf("%s\n\n%s\n\n%s",
			styles.SelectorTitle.Render("╭─ [ "+s.Title+" ] "),
			styles.Muted.Render("  No items available."),
			styles.Muted.Render("  [Esc] Cancel"),
		)
		return styles.SelectorBox.Width(boxW).Render(emptyContent)
	}

	var b strings.Builder
	b.WriteString(styles.SelectorTitle.Render("◈ " + strings.ToUpper(s.Title)))
	b.WriteString("\n\n")

	for i, item := range s.Items {
		isCursor := i == s.cursor
		isActive := s.ActiveID != "" && item.ID == s.ActiveID

		prefix := "  "
		if isCursor {
			prefix = "› "
		}

		label := item.Label
		if isActive {
			label += " " + styles.Success.Render("[Active]")
		}
		if item.Badge != "" {
			label += " " + styles.Muted.Render("("+item.Badge+")")
		}

		var line string
		if isCursor {
			line = styles.SelectorItemSelected.Render(prefix + label)
		} else {
			line = styles.SelectorItem.Render(prefix + label)
		}

		b.WriteString(line)
		b.WriteString("\n")

		if item.Sub != "" {
			subIndent := "    "
			b.WriteString(styles.Muted.Render(subIndent + item.Sub))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.Muted.Render(" [↑/↓/j/k] Navigate  ·  [Enter] Select  ·  [Esc] Close"))

	return styles.SelectorBox.Width(boxW).Render(b.String())
}
