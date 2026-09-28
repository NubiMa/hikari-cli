package components

import (
	"fmt"
	"strings"

	"github.com/NubiMa/hikari-cli/internal/tui/styles"
	"github.com/charmbracelet/lipgloss"
)

// Header renders the top bar showing the app name, active provider,
// persona, model, and connection status as required by PRD Section 16.
type Header struct {
	Width        int
	ProviderName string
	PersonaName  string
	ModelName    string
	SessionTitle string
	Connected    bool
	ShowSidebar  bool
}

// View renders the header bar to a string.
func (h Header) View() string {
	if h.Width <= 0 {
		return ""
	}

	// 1. Left brand logo
	logo := styles.HeaderLogo.Render("◈ HIKARI")

	// 2. Middle pills: Persona, Provider, Model, Session
	var pills []string

	if h.PersonaName != "" {
		pills = append(pills, styles.HeaderPill.Render(fmt.Sprintf("👤 %s", styles.PersonaLabel.Render(h.PersonaName))))
	}

	if h.ProviderName != "" {
		dot := styles.ProviderDisconnected.Render("○")
		if h.Connected {
			dot = styles.ProviderConnected.Render("●")
		}
		pills = append(pills, styles.HeaderPill.Render(fmt.Sprintf("%s %s", dot, h.ProviderName)))
	}

	if h.ModelName != "" {
		pills = append(pills, styles.HeaderPill.Render(fmt.Sprintf("⚡ %s", h.ModelName)))
	}

	if h.SessionTitle != "" && h.Width > 90 {
		title := h.SessionTitle
		if len(title) > 20 {
			title = title[:17] + "…"
		}
		pills = append(pills, styles.HeaderPill.Render(fmt.Sprintf("📁 %s", title)))
	}

	middleContent := strings.Join(pills, " ")

	// 3. Right help & toggle hints
	sidebarHint := "Sidebar: ON"
	if !h.ShowSidebar {
		sidebarHint = "Sidebar: OFF"
	}
	right := styles.HeaderPillDim.Render(fmt.Sprintf("[Tab] %s  ·  [?] /help", sidebarHint))

	// Layout spacing
	leftWidth := lipgloss.Width(logo) + 1 + lipgloss.Width(middleContent)
	rightWidth := lipgloss.Width(right)
	padding := h.Width - leftWidth - rightWidth - 2
	if padding < 0 {
		padding = 0
	}
	spacer := strings.Repeat(" ", padding)

	content := logo + " " + middleContent + spacer + right
	return styles.Header.Width(h.Width).Render(content)
}

// StatusBar renders the bottom status bar.
type StatusBar struct {
	Width       int
	Message     string
	IsError     bool
	IsStreaming bool
}

func (s StatusBar) View() string {
	if s.Width <= 0 {
		return ""
	}

	// Mode badge
	modeBadge := styles.HeaderPillActive.Render(" NORMAL ")
	if s.IsStreaming {
		modeBadge = styles.SidebarBadge.Render(" STREAMING ")
	} else if s.IsError {
		modeBadge = styles.Error.Background(styles.ColorSubtle).Render(" ERROR ")
	}

	// Message
	msg := s.Message
	if msg == "" {
		msg = "Ready"
	}
	msgText := styles.Muted.Render(msg)
	if s.IsError {
		msgText = styles.Error.Render("✗ " + msg)
	}

	// Right key hints
	hints := styles.Muted.Render("[Enter] Send  ·  [/] Commands  ·  [Ctrl+C] Exit")

	leftW := lipgloss.Width(modeBadge) + 1 + lipgloss.Width(msgText)
	rightW := lipgloss.Width(hints)
	pad := s.Width - leftW - rightW - 2
	if pad < 0 {
		pad = 0
	}
	spacer := strings.Repeat(" ", pad)

	content := modeBadge + " " + msgText + spacer + hints
	return styles.StatusBar.Width(s.Width).Render(content)
}
