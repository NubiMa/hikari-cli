package components

import (
	"fmt"
	"strings"

	"github.com/NubiMa/hikari-cli/internal/persona"
	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
)

// Sidebar renders the contextual information panel required by PRD Section 16.
type Sidebar struct {
	Width        int
	Height       int
	ProviderName string
	ProviderType string
	ModelName    string
	Connected    bool
	Persona      *persona.Persona
	Session      *session.Session
}

// View renders the sidebar context view.
func (s Sidebar) View() string {
	if s.Width <= 0 || s.Height <= 0 {
		return ""
	}

	contentW := s.Width - 4
	if contentW < 12 {
		return ""
	}

	var b strings.Builder

	// Header
	b.WriteString(styles.SidebarHeader.Render("◈ CONTEXT"))
	b.WriteString("\n\n")

	// Section 1: Persona
	b.WriteString(styles.SidebarSection.Render("PERSONA"))
	b.WriteString("\n")
	pName := "Default"
	pDesc := ""
	if s.Persona != nil {
		if s.Persona.Name != "" {
			pName = s.Persona.Name
		}
		pDesc = s.Persona.Description
	}
	b.WriteString(styles.PersonaLabel.Render("  " + pName))
	b.WriteString("\n")
	if pDesc != "" {
		desc := pDesc
		if len(desc) > contentW {
			desc = desc[:contentW-3] + "…"
		}
		b.WriteString(styles.Muted.Render("  " + desc))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Section 2: Provider & Model
	b.WriteString(styles.SidebarSection.Render("PROVIDER"))
	b.WriteString("\n")
	provName := s.ProviderName
	if provName == "" {
		provName = "None"
	}

	statusDot := styles.ProviderDisconnected.Render("○ Offline")
	if s.Connected {
		statusDot = styles.ProviderConnected.Render("● Connected")
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", styles.Bold.Render(provName), statusDot))

	if s.ModelName != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Muted.Render("Model:"), styles.SidebarVal.Render(s.ModelName)))
	}
	if s.ProviderType != "" {
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.Muted.Render("Type: "), styles.Muted.Render(s.ProviderType)))
	}
	b.WriteString("\n")

	// Section 3: Session
	b.WriteString(styles.SidebarSection.Render("SESSION"))
	b.WriteString("\n")
	if s.Session != nil {
		title := s.Session.Title
		if len(title) > contentW {
			title = title[:contentW-3] + "…"
		}
		b.WriteString(fmt.Sprintf("  %s\n", styles.SidebarVal.Render(title)))
		b.WriteString(fmt.Sprintf("  %s\n", styles.Muted.Render(fmt.Sprintf("%d messages", len(s.Session.Messages)))))
	} else {
		b.WriteString(styles.Muted.Render("  New Session\n"))
	}
	b.WriteString("\n")

	// Section 4: Quick Actions
	b.WriteString(styles.SidebarSection.Render("ACTIONS"))
	b.WriteString("\n")
	actions := []struct{ cmd, desc string }{
		{"/provider", "Switch AI"},
		{"/persona", "Switch role"},
		{"/model", "Switch model"},
		{"/theme", "Change theme"},
		{"/history", "Sessions"},
		{"/status", "Health test"},
	}
	for _, a := range actions {
		b.WriteString(fmt.Sprintf("  %s %s\n", styles.SplashKey.Render(a.cmd), styles.Muted.Render(a.desc)))
	}
	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("  [Tab] Toggle"))

	rendered := b.String()
	boxHeight := s.Height
	if boxHeight < 5 {
		boxHeight = 5
	}

	return styles.SidebarBox.
		Width(s.Width).
		Height(boxHeight).
		Render(rendered)
}
