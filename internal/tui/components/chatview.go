package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
)

// ChatMessage is a rendered message entry in the chat view.
type ChatMessage struct {
	Role      provider.Role
	Name      string // persona name or "You"
	Content   string
	Timestamp time.Time
}

// ChatView holds and renders the conversation message list with visual polish.
type ChatView struct {
	Width         int
	Height        int
	Messages      []ChatMessage
	Streaming     string // current in-progress assistant token accumulator
	IsStreaming   bool
	scrollOffset  int
	AsciiBanner   string
	ActivePersona string
}

// AddMessage appends a completed message to the view.
func (c *ChatView) AddMessage(role provider.Role, name, content string) {
	c.Messages = append(c.Messages, ChatMessage{
		Role:      role,
		Name:      name,
		Content:   content,
		Timestamp: time.Now(),
	})
	// Reset scroll on new message to show latest content
	c.scrollOffset = 0
}

// AppendToken appends a streaming token to the current in-progress response.
func (c *ChatView) AppendToken(token string) {
	c.Streaming += token
	c.IsStreaming = true
}

// CommitStreaming moves the accumulated streaming content into Messages and
// resets the streaming state.
func (c *ChatView) CommitStreaming(assistantName string) {
	if c.Streaming != "" {
		c.Messages = append(c.Messages, ChatMessage{
			Role:      provider.RoleAssistant,
			Name:      assistantName,
			Content:   c.Streaming,
			Timestamp: time.Now(),
		})
	}
	c.Streaming = ""
	c.IsStreaming = false
}

// ClearMessages removes all messages from the view.
func (c *ChatView) ClearMessages() {
	c.Messages = nil
	c.Streaming = ""
	c.IsStreaming = false
	c.scrollOffset = 0
}

// ScrollUp scrolls the view up by n lines.
func (c *ChatView) ScrollUp(n int) {
	c.scrollOffset += n
}

// ScrollDown scrolls the view down by n lines.
func (c *ChatView) ScrollDown(n int) {
	c.scrollOffset -= n
	if c.scrollOffset < 0 {
		c.scrollOffset = 0
	}
}

// View renders the chat view as a string.
func (c ChatView) View() string {
	innerWidth := c.Width - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	// If no user messages yet, show the Welcome Splash Hero
	if len(c.Messages) == 0 && !c.IsStreaming {
		return c.renderSplash(innerWidth)
	}

	var lines []string
	for _, msg := range c.Messages {
		lines = append(lines, c.renderMessage(msg, innerWidth)...)
		lines = append(lines, "") // spacing between message blocks
	}

	// Append streaming response if active
	if c.IsStreaming {
		streamMsg := ChatMessage{
			Role:      provider.RoleAssistant,
			Name:      c.ActivePersona,
			Content:   c.Streaming,
			Timestamp: time.Now(),
		}
		lines = append(lines, c.renderStreaming(streamMsg, innerWidth)...)
	}

	// Calculate visible viewport slice
	totalLines := len(lines)
	visible := c.Height
	if visible <= 0 {
		visible = 20
	}

	startLine := totalLines - visible - c.scrollOffset
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + visible
	if endLine > totalLines {
		endLine = totalLines
	}

	visibleLines := lines
	if startLine < endLine {
		visibleLines = lines[startLine:endLine]
	}

	content := strings.Join(visibleLines, "\n")
	return styles.ChatArea.Width(c.Width).Render(content)
}

func (c ChatView) renderSplash(width int) string {
	var b strings.Builder

	// ASCII Art banner
	banner := c.AsciiBanner
	if banner == "" {
		banner = "    ██╗  ██╗██╗██╗  ██╗ █████╗ ██████╗ ██╗\n" +
			"    ██║  ██║██║██║ ██╔╝██╔══██╗██╔══██╗██║\n" +
			"    ███████║██║█████╔╝ ███████║██████╔╝██║\n" +
			"    ██╔══██║██║██╔═██╗ ██╔══██║██╔══██╗██║\n" +
			"    ██║  ██║██║██║  ██╗██║  ██║██║  ██║██║\n" +
			"    ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝"
	}

	b.WriteString(styles.SplashTitle.Render(banner))
	b.WriteString("\n")
	b.WriteString(styles.SplashSub.Render("One terminal platform, unlimited AI personalities."))
	b.WriteString("\n\n")

	// Quick start card
	cardWidth := width - 4
	if cardWidth > 60 {
		cardWidth = 60
	}

	cardContent := fmt.Sprintf(
		"%s\n\n"+
			"  %s  Select AI Provider (Ollama, OpenClaw, Hermes)\n"+
			"  %s  Select Persona (Nino, Developer, SysAdmin)\n"+
			"  %s  Select Color Theme (Default, Minimal, Tokyo Night)\n"+
			"  %s  View all commands & keybindings\n\n"+
			"%s",
		styles.Bold.Render("⚡ GETTING STARTED"),
		styles.SplashKey.Render(" [1] /provider "),
		styles.SplashKey.Render(" [2] /persona  "),
		styles.SplashKey.Render(" [3] /theme    "),
		styles.SplashKey.Render(" [?] /help     "),
		styles.Muted.Render("Type a message below or press Tab to inspect context."),
	)

	b.WriteString(styles.SplashCard.Width(cardWidth).Render(cardContent))
	return styles.ChatArea.Width(c.Width).Render(b.String())
}

func (c ChatView) renderMessage(msg ChatMessage, width int) []string {
	var result []string

	ts := ""
	if !msg.Timestamp.IsZero() {
		ts = "  " + styles.Muted.Render(msg.Timestamp.Format("15:04"))
	}

	// 1. Header with icon and role badge
	switch msg.Role {
	case provider.RoleUser:
		header := fmt.Sprintf("  %s %s%s", styles.UserHeader.Render("❯"), styles.Bold.Render("You"), ts)
		result = append(result, header)
		bar := styles.UserHeader.Render("  │ ")
		for _, line := range splitWrap(msg.Content, width-6) {
			result = append(result, bar+styles.UserMessage.Render(line))
		}

	case provider.RoleAssistant:
		name := msg.Name
		if name == "" {
			name = "Hikari"
		}
		header := fmt.Sprintf("  %s %s%s", styles.AssistantHeader.Render("◈"), styles.AssistantHeader.Render(name), ts)
		result = append(result, header)
		bar := styles.AssistantHeader.Render("  │ ")
		for _, line := range splitWrap(msg.Content, width-6) {
			result = append(result, bar+styles.AssistantMessage.Render(line))
		}

	case provider.RoleSystem:
		header := fmt.Sprintf("  %s %s%s", styles.SystemHeader.Render("ℹ"), styles.SystemHeader.Render("System"), ts)
		result = append(result, header)
		bar := styles.SystemHeader.Render("  │ ")
		for _, line := range splitWrap(msg.Content, width-6) {
			result = append(result, bar+styles.Muted.Render(line))
		}
	}

	return result
}

func (c ChatView) renderStreaming(msg ChatMessage, width int) []string {
	var result []string
	name := msg.Name
	if name == "" {
		name = "Hikari"
	}

	header := fmt.Sprintf("  %s %s  %s", styles.AssistantHeader.Render("◈"), styles.AssistantHeader.Render(name), styles.StreamingCursor.Render("thinking…"))
	result = append(result, header)
	bar := styles.AssistantHeader.Render("  │ ")

	wrapped := splitWrap(msg.Content, width-6)
	if len(wrapped) == 0 {
		result = append(result, bar+styles.StreamingCursor.Render("▋"))
	} else {
		for i, line := range wrapped {
			if i == len(wrapped)-1 {
				result = append(result, bar+styles.AssistantMessage.Render(line)+styles.StreamingCursor.Render("▋"))
			} else {
				result = append(result, bar+styles.AssistantMessage.Render(line))
			}
		}
	}

	return result
}

// splitWrap splits text by existing newlines and wraps long paragraphs.
func splitWrap(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	var out []string
	lines := strings.Split(s, "\n")
	for _, l := range lines {
		if len(l) <= width {
			out = append(out, l)
			continue
		}
		// Word wrap line
		words := strings.Fields(l)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		curr := words[0]
		for _, w := range words[1:] {
			if len(curr)+1+len(w) <= width {
				curr += " " + w
			} else {
				out = append(out, curr)
				curr = w
			}
		}
		out = append(out, curr)
	}
	return out
}
