package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/NubiMa/hikari-cli/internal/tui/commands"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
)

// SubmitMsg is sent when the user presses Enter with non-empty input.
type SubmitMsg struct {
	Value string
}

// CommandMsg is sent when the user submits a /command.
type CommandMsg struct {
	Command string // e.g. "provider", "persona", "help"
	Args    string // remainder after the command word
}

// InputModel wraps the textinput component with command completion and rich styling.
type InputModel struct {
	input    textinput.Model
	Width    int
	Disabled bool // true while streaming (prevent sending new message)
}

// NewInputModel creates an initialised InputModel.
func NewInputModel() InputModel {
	ti := textinput.New()
	ti.Placeholder = "Type a message… (/ for commands, Tab to complete/toggle)"
	ti.Focus()
	ti.CharLimit = 4096

	return InputModel{input: ti}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	if m.Disabled {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		val := m.input.Value()

		// Tab completion for commands
		if msg.Type == tea.KeyTab && strings.HasPrefix(val, "/") && !strings.Contains(val, " ") {
			prefix := strings.TrimPrefix(val, "/")
			for _, c := range commands.KnownCommands {
				if strings.HasPrefix(c.Name, prefix) {
					m.input.SetValue("/" + c.Name + " ")
					m.input.SetCursor(len("/" + c.Name + " "))
					return m, nil
				}
			}
		}

		if msg.Type == tea.KeyEnter {
			trimmed := strings.TrimSpace(val)
			if trimmed == "" {
				return m, nil
			}
			m.input.SetValue("")

			if strings.HasPrefix(trimmed, "/") {
				parts := strings.SplitN(trimmed[1:], " ", 2)
				cmd := strings.ToLower(parts[0])
				args := ""
				if len(parts) > 1 {
					args = parts[1]
				}
				return m, func() tea.Msg { return CommandMsg{Command: cmd, Args: args} }
			}
			return m, func() tea.Msg { return SubmitMsg{Value: trimmed} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	val := m.input.Value()

	prompt := styles.InputPrompt.Render("❯ ")
	if m.Disabled {
		prompt = styles.Muted.Render("⏳ ")
	}

	field := m.input.View()
	inner := prompt + field

	borderStyle := styles.InputBorder
	if m.input.Focused() && !m.Disabled {
		borderStyle = styles.InputFocused
	}

	boxW := m.Width - 2
	if boxW < 20 {
		boxW = 20
	}
	inputRendered := borderStyle.Width(boxW).Render(inner)

	// Command completion helper overlay when input starts with '/'
	if strings.HasPrefix(val, "/") && !strings.Contains(val, " ") {
		prefix := strings.TrimPrefix(val, "/")
		var matches []string
		for _, c := range commands.KnownCommands {
			if strings.HasPrefix(c.Name, prefix) {
				match := fmt.Sprintf("%s %s", styles.SplashKey.Render("/"+c.Name), styles.Muted.Render(c.Description))
				matches = append(matches, match)
			}
		}
		if len(matches) > 0 {
			var preview string
			if len(matches) > 4 {
				preview = strings.Join(matches[:4], "  ·  ") + " …"
			} else {
				preview = strings.Join(matches, "  ·  ")
			}
			popup := styles.CommandPopupBox.Width(boxW).Render(preview)
			return popup + "\n" + inputRendered
		}
	}

	return inputRendered
}

// SetWidth updates the width of the input (called on window resize).
func (m *InputModel) SetWidth(w int) {
	m.Width = w
	m.input.Width = w - 6 // account for border + prompt
}

// SetDisabled enables or disables the input field.
func (m *InputModel) SetDisabled(d bool) {
	m.Disabled = d
	if d {
		m.input.Blur()
	} else {
		m.input.Focus()
	}
}

// Value returns current input text.
func (m InputModel) Value() string {
	return m.input.Value()
}
