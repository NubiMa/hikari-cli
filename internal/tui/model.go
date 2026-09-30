// Package tui implements the Hikari interactive terminal UI using Bubble Tea.
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/app"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/session"
	"github.com/NubiMa/hikari-cli/internal/stream"
	"github.com/NubiMa/hikari-cli/internal/tui/commands"
	"github.com/NubiMa/hikari-cli/internal/tui/components"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// View modes
// ---------------------------------------------------------------------------

type viewMode int

const (
	modeChat viewMode = iota
	modeProviderSelector
	modePersonaSelector
	modeModelSelector
	modeSessionSelector
	modeThemeSelector
	modeHelp
	modeStatus
)

// ---------------------------------------------------------------------------
// Root Model
// ---------------------------------------------------------------------------

// Model is the root Bubble Tea model for the Hikari TUI.
type Model struct {
	app     *app.App
	session *session.Session

	// Layout
	width       int
	height      int
	showSidebar bool

	// Components
	chatView components.ChatView
	input    components.InputModel
	selector components.Selector

	// State
	mode        viewMode
	connected   bool
	statusMsg   string
	isError     bool
	eventCancel context.CancelFunc // cancels the current streaming request

	// Streaming
	eventCh     <-chan provider.Event
	isStreaming bool
}

// New creates a new TUI Model. Call Start() to run it.
func New(a *app.App) (*Model, error) {
	sess, err := a.NewSession()
	if err != nil {
		return nil, fmt.Errorf("creating session: %w", err)
	}

	// Load ASCII banner
	asciiName := "default"
	if a.Config != nil && a.Config.UI.ASCII != "" {
		asciiName = a.Config.UI.ASCII
	}
	asciiBanner := a.Themes.LoadASCII(asciiName, "")

	activePersona := a.Personas.Active()

	m := &Model{
		app:         a,
		session:     sess,
		input:       components.NewInputModel(),
		mode:        modeChat,
		showSidebar: true,
		chatView: components.ChatView{
			AsciiBanner:   asciiBanner,
			ActivePersona: activePersona.Name,
		},
	}

	return m, nil
}

// Start runs the Bubble Tea program.
func Start(a *app.App) error {
	m, err := New(a)
	if err != nil {
		return err
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	return err
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	persona := m.app.Personas.Active()

	// Show greeting if persona has one
	if persona.Greeting != "" {
		m.chatView.AddMessage(provider.RoleAssistant, persona.Name, persona.Greeting)
	}

	return tea.Batch(
		m.input.Init(),
		connectCmd(m.app),
	)
}

// connectCmd attempts to connect the active provider in the background.
func connectCmd(a *app.App) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := a.ConnectActive(ctx)
		return connectedMsg{err: err}
	}
}

type connectedMsg struct{ err error }

// streamReadyMsg is sent once a Chat() call succeeds and carries the event
// channel back to Update() so it can be stored on the real Model value.
// This is necessary because handleSubmit runs inside a tea.Cmd closure where
// mutations to m are invisible to Bubble Tea.
type streamReadyMsg struct{ ch <-chan provider.Event }

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// -- Window resize --
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.SetWidth(msg.Width)
		m.selector.Width = msg.Width
		m.selector.Height = msg.Height
		return m, nil

	// -- Provider connection result --
	case connectedMsg:
		if msg.err != nil {
			m.connected = false
			m.statusMsg = "Provider unavailable: " + sanitiseError(msg.err)
			m.isError = true
		} else {
			m.connected = true
			m.statusMsg = ""
			m.isError = false
		}
		return m, nil

	// -- Stream channel handoff --
	// streamReadyMsg arrives once Chat() returns successfully. We store the
	// channel on the real model here (inside Update) so that subsequent
	// NextEvent calls actually read from it.
	case streamReadyMsg:
		m.eventCh = msg.ch
		return m, stream.NextEvent(m.eventCh)

	// -- Streaming events --
	case stream.TokenMsg:
		m.chatView.AppendToken(msg.Content)
		return m, stream.NextEvent(m.eventCh)

	case stream.DoneMsg:
		persona := m.app.Personas.Active()
		m.chatView.CommitStreaming(persona.Name)
		m.isStreaming = false
		m.input.SetDisabled(false)
		m.statusMsg = ""
		// Save assistant response to session
		if len(m.chatView.Messages) > 0 {
			last := m.chatView.Messages[len(m.chatView.Messages)-1]
			m.session.AddMessage(provider.RoleAssistant, last.Content)
			_ = m.app.Sessions.Save(m.session)
		}
		return m, nil

	case stream.ErrorMsg:
		m.chatView.CommitStreaming(m.app.Personas.Active().Name)
		m.isStreaming = false
		m.input.SetDisabled(false)
		m.statusMsg = sanitiseError(msg.Err)
		m.isError = true
		return m, nil

	case stream.StatusMsg:
		m.statusMsg = msg.Content
		return m, stream.NextEvent(m.eventCh)

	// -- Global keyboard shortcuts --
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if m.isStreaming && m.eventCancel != nil {
				m.eventCancel()
				m.isStreaming = false
				m.input.SetDisabled(false)
				m.chatView.CommitStreaming(m.app.Personas.Active().Name)
				m.statusMsg = "Generation cancelled"
				return m, nil
			}
			return m, tea.Quit

		case "tab":
			// Toggle sidebar context view
			if m.mode == modeChat && !strings.HasPrefix(m.input.Value(), "/") {
				m.showSidebar = !m.showSidebar
				return m, nil
			}

		case "ctrl+l":
			m.chatView.ClearMessages()
			m.statusMsg = ""
			return m, nil

		case "pgup":
			m.chatView.ScrollUp(5)
			return m, nil

		case "pgdown":
			m.chatView.ScrollDown(5)
			return m, nil
		}

		// Delegate keyboard to active modal picker if open
		if m.mode != modeChat {
			return m.updateSelector(msg)
		}

	// -- Input events --
	case components.SubmitMsg:
		return m.handleSubmit(msg.Value)

	case components.CommandMsg:
		return m.handleCommand(msg)

	// -- Selector events --
	case components.SelectorChosenMsg:
		return m.handleSelectorChosen(msg)

	case components.SelectorCancelledMsg:
		m.mode = modeChat
		m.statusMsg = ""
		return m, nil
	}

	// Delegate to input component when in chat mode
	if m.mode == modeChat {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	// Delegate to selector in picker modes
	return m.updateSelector(msg)
}

// ---------------------------------------------------------------------------
// Submit (send message to provider)
// ---------------------------------------------------------------------------

func (m Model) handleSubmit(text string) (tea.Model, tea.Cmd) {
	if text == "" || m.isStreaming {
		return m, nil
	}

	p, _, err := m.app.Router.Active()
	if err != nil {
		m.statusMsg = err.Error()
		m.isError = true
		return m, nil
	}

	persona := m.app.Personas.Active()

	// Add user message to view and session
	m.chatView.AddMessage(provider.RoleUser, "You", text)
	m.session.AddMessage(provider.RoleUser, text)
	_ = m.app.Sessions.Save(m.session)

	// Build history with system prompt prepended
	history := buildHistory(persona.SystemPrompt, m.session)

	m.isStreaming = true
	m.input.SetDisabled(true)
	m.statusMsg = fmt.Sprintf("%s is thinking…", persona.Name)
	m.isError = false

	ctx, cancel := context.WithCancel(context.Background())
	m.eventCancel = cancel

	return m, func() tea.Msg {
		ch, err := p.Chat(ctx, history, text)
		if err != nil {
			cancel()
			return stream.ErrorMsg{Err: err}
		}
		// Return streamReadyMsg so Update() can store ch on the real Model.
		// We must NOT touch m.eventCh here — this closure captures a copy of m.
		return streamReadyMsg{ch: ch}
	}
}

// buildHistory constructs the message list to send to the provider,
// prepending the system prompt if one is defined.
func buildHistory(systemPrompt string, sess *session.Session) []provider.Message {
	var history []provider.Message
	if systemPrompt != "" {
		history = append(history, provider.Message{
			Role:    provider.RoleSystem,
			Content: systemPrompt,
		})
	}
	history = append(history, sess.ProviderHistory()...)
	return history
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (m Model) handleCommand(msg components.CommandMsg) (tea.Model, tea.Cmd) {
	switch msg.Command {
	case "help":
		m.chatView.AddMessage(provider.RoleSystem, "Hikari", commands.HelpText())
		return m, nil

	case "clear":
		m.chatView.ClearMessages()
		m.session.Messages = nil
		_ = m.app.Sessions.Save(m.session)
		persona := m.app.Personas.Active()
		if persona.Greeting != "" {
			m.chatView.AddMessage(provider.RoleAssistant, persona.Name, persona.Greeting)
		}
		return m, nil

	case "exit":
		return m, tea.Quit

	case "status":
		return m.showStatus()

	case "provider":
		return m.openProviderSelector()

	case "persona":
		return m.openPersonaSelector()

	case "model":
		return m.openModelSelector()

	case "session", "history":
		return m.openSessionSelector()

	case "home", "new":
		return m.homeSession()

	case "log":
		return m.showLog()

	case "theme":
		return m.openThemeSelector()

	default:
		m.statusMsg = fmt.Sprintf("Unknown command: /%s  (try /help)", msg.Command)
		m.isError = true
		return m, nil
	}
}

// showStatus checks health for all configured providers with rich diagnostics.
func (m Model) showStatus() (tea.Model, tea.Cmd) {
	var b strings.Builder
	b.WriteString(styles.Bold.Render("Provider Status"))
	b.WriteString("\n\n")

	for _, name := range m.app.Router.AvailableNames() {
		prov, err := m.app.Registry.Get(name)
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		status, statusErr := prov.Status(ctx)
		cancel()

		caps := prov.Capabilities()

		// Status dot + label
		var statusLabel string
		if statusErr != nil || !status.Connected {
			statusLabel = styles.ProviderDisconnected.Render("○ Offline")
		} else {
			statusLabel = styles.ProviderConnected.Render("● Online")
		}

		latencyStr := ""
		if status.Latency > 0 {
			latencyStr = fmt.Sprintf("  %dms", status.Latency.Milliseconds())
		}

		// Header line: name + status + latency
		b.WriteString(fmt.Sprintf("  %-16s %s%s\n", styles.Bold.Render(name), statusLabel, styles.Muted.Render(latencyStr)))

		// Endpoint
		if status.Endpoint != "" {
			b.WriteString(styles.Muted.Render(fmt.Sprintf("    Endpoint: %s\n", status.Endpoint)))
		}

		// Status message
		if status.Message != "" {
			b.WriteString(styles.Muted.Render(fmt.Sprintf("    Status:   %s\n", status.Message)))
		}

		// Detailed diagnostic lines (models list, auth hints, etc.)
		for _, d := range status.Details {
			b.WriteString(styles.Muted.Render("    " + d + "\n"))
		}

		// Capabilities summary
		b.WriteString(styles.Muted.Render(fmt.Sprintf(
			"    Caps:     streaming=%v  tools=%v  memory=%v  agent=%v\n",
			caps.Streaming, caps.ToolCalling, caps.Memory, caps.AgentExecution)))
		b.WriteString("\n")
	}

	m.chatView.AddMessage(provider.RoleSystem, "Hikari", b.String())
	return m, nil
}

// showLog displays the path to the log file and the last few entries.
func (m Model) showLog() (tea.Model, tea.Cmd) {
	var b strings.Builder
	b.WriteString(styles.Bold.Render("Application Log"))
	b.WriteString("\n\n")

	logPath := m.app.LogPath
	b.WriteString(styles.Muted.Render("Log file: " + logPath))
	b.WriteString("\n\n")

	// Read the last ~20 lines of the log for quick inspection.
	data, err := os.ReadFile(logPath)
	if err != nil {
		b.WriteString(styles.Muted.Render("(could not read log file: " + err.Error() + ")"))
	} else {
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		const tail = 20
		if len(lines) > tail {
			b.WriteString(styles.Muted.Render(fmt.Sprintf("(showing last %d of %d lines)\n\n", tail, len(lines))))
			lines = lines[len(lines)-tail:]
		}
		for _, l := range lines {
			b.WriteString(l)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("Tip: tail -f " + logPath + "  to monitor live"))

	m.chatView.AddMessage(provider.RoleSystem, "Hikari", b.String())
	if m.app.Logger != nil {
		m.app.Logger.Printf("user ran /log")
	}
	return m, nil
}

// homeSession discards the current session and starts a fresh one,
// effectively returning the user to the "home" / welcome screen.
func (m Model) homeSession() (tea.Model, tea.Cmd) {
	if m.isStreaming && m.eventCancel != nil {
		m.eventCancel()
		m.isStreaming = false
		m.input.SetDisabled(false)
	}

	newSess, err := m.app.NewSession()
	if err != nil {
		m.statusMsg = "Could not create new session: " + sanitiseError(err)
		m.isError = true
		return m, nil
	}

	m.session = newSess
	m.chatView.ClearMessages()
	m.statusMsg = ""
	m.isError = false

	// Show greeting for the active persona on the fresh session.
	persona := m.app.Personas.Active()
	if persona.Greeting != "" {
		m.chatView.AddMessage(provider.RoleAssistant, persona.Name, persona.Greeting)
	}

	if m.app.Logger != nil {
		m.app.Logger.Printf("user returned home (new session %s)", newSess.ID)
	}
	return m, nil
}

func (m Model) openProviderSelector() (tea.Model, tea.Cmd) {
	names := m.app.Router.AvailableNames()
	items := make([]components.SelectorItem, len(names))
	for i, n := range names {
		pType := "unknown"
		model := ""
		if m.app.Config != nil {
			if cfg, ok := m.app.Config.Providers[n]; ok {
				pType = cfg.Type
				model = cfg.Model
			}
		}
		sub := fmt.Sprintf("Type: %s", pType)
		if model != "" {
			sub += fmt.Sprintf("  ·  Model: %s", model)
		}
		items[i] = components.SelectorItem{
			ID:    n,
			Label: n,
			Sub:   sub,
		}
	}
	m.selector = components.Selector{
		Title:    "Select Provider",
		Context:  "provider",
		Items:    items,
		ActiveID: m.app.Router.ActiveName(),
		Width:    m.width,
		Height:   m.height,
	}
	m.mode = modeProviderSelector
	return m, nil
}

func (m Model) openPersonaSelector() (tea.Model, tea.Cmd) {
	personas := m.app.Personas.All()
	items := make([]components.SelectorItem, len(personas))
	for i, p := range personas {
		items[i] = components.SelectorItem{
			ID:    p.Name,
			Label: p.Name,
			Sub:   p.Description,
		}
	}
	m.selector = components.Selector{
		Title:    "Select Persona",
		Context:  "persona",
		Items:    items,
		ActiveID: m.app.Personas.Active().Name,
		Width:    m.width,
		Height:   m.height,
	}
	m.mode = modePersonaSelector
	return m, nil
}

func (m Model) openModelSelector() (tea.Model, tea.Cmd) {
	p, _, err := m.app.Router.Active()
	if err != nil || !p.Capabilities().Models {
		m.statusMsg = "Active provider does not support model listing"
		m.isError = true
		return m, nil
	}

	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		models, err := p.Models(ctx)
		if err != nil {
			return stream.ErrorMsg{Err: err}
		}
		items := make([]components.SelectorItem, len(models))
		for i, mod := range models {
			items[i] = components.SelectorItem{ID: mod.ID, Label: mod.Name, Sub: mod.Description}
		}
		return modelsLoadedMsg{items: items}
	}
}

type modelsLoadedMsg struct{ items []components.SelectorItem }

func (m Model) openSessionSelector() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		entries, err := m.app.Sessions.List()
		if err != nil {
			return stream.ErrorMsg{Err: err}
		}

		groups := session.GroupByDate(entries)
		var items []components.SelectorItem
		for _, g := range groups {
			for _, e := range g.Sessions {
				items = append(items, components.SelectorItem{
					ID:    e.ID,
					Label: e.Title,
					Badge: g.Label,
					Sub:   fmt.Sprintf("%s  ·  %s", e.Provider, e.UpdatedAt.Format("Jan 2, 15:04")),
				})
			}
		}

		return sessionsLoadedMsg{items: items}
	}
}

type sessionsLoadedMsg struct{ items []components.SelectorItem }

func (m Model) openThemeSelector() (tea.Model, tea.Cmd) {
	themes := m.app.Themes.All()
	items := make([]components.SelectorItem, len(themes))
	for i, t := range themes {
		items[i] = components.SelectorItem{ID: t.Name, Label: t.Name, Sub: t.Description}
	}
	m.selector = components.Selector{
		Title:    "Select Theme",
		Context:  "theme",
		Items:    items,
		ActiveID: m.app.Themes.Active().Name,
		Width:    m.width,
		Height:   m.height,
	}
	m.mode = modeThemeSelector
	return m, nil
}

func (m Model) updateSelector(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle async loads
	switch msg := msg.(type) {
	case modelsLoadedMsg:
		m.selector = components.Selector{
			Title:   "Select Model",
			Context: "model",
			Items:   msg.items,
			Width:   m.width,
			Height:  m.height,
		}
		m.mode = modeModelSelector
		return m, nil

	case sessionsLoadedMsg:
		m.selector = components.Selector{
			Title:    "Session History",
			Context:  "session",
			Items:    msg.items,
			ActiveID: m.session.ID,
			Width:    m.width,
			Height:   m.height,
		}
		m.mode = modeSessionSelector
		return m, nil
	}

	var cmd tea.Cmd
	m.selector, cmd = m.selector.Update(msg)
	return m, cmd
}

func (m Model) handleSelectorChosen(msg components.SelectorChosenMsg) (tea.Model, tea.Cmd) {
	m.mode = modeChat

	switch msg.Context {
	case "provider":
		if err := m.app.Router.SetActive(msg.Item.ID); err != nil {
			m.statusMsg = err.Error()
			m.isError = true
			return m, nil
		}
		m.connected = false
		m.statusMsg = fmt.Sprintf("Connecting to %s…", msg.Item.ID)
		return m, connectCmd(m.app)

	case "persona":
		if err := m.app.Personas.SetActive(msg.Item.ID); err != nil {
			m.statusMsg = err.Error()
			m.isError = true
			return m, nil
		}
		p := m.app.Personas.Active()
		m.chatView.ActivePersona = p.Name
		m.statusMsg = fmt.Sprintf("Switched to persona: %s", msg.Item.ID)
		if p.Greeting != "" {
			m.chatView.AddMessage(provider.RoleAssistant, p.Name, p.Greeting)
		}

	case "model":
		m.statusMsg = fmt.Sprintf("Model set to: %s", msg.Item.ID)

	case "session":
		sess, err := m.app.Sessions.Load(msg.Item.ID)
		if err != nil {
			m.statusMsg = "Failed to load session: " + sanitiseError(err)
			m.isError = true
			return m, nil
		}
		m.session = sess
		m.chatView.ClearMessages()
		// Replay history into chat view
		persona := m.app.Personas.Active()
		for _, msg := range sess.Messages {
			switch msg.Role {
			case provider.RoleUser:
				m.chatView.AddMessage(provider.RoleUser, "You", msg.Content)
			case provider.RoleAssistant:
				m.chatView.AddMessage(provider.RoleAssistant, persona.Name, msg.Content)
			}
		}
		m.statusMsg = fmt.Sprintf("Resumed session: %s", sess.Title)

	case "theme":
		if err := m.app.Themes.SetActive(msg.Item.ID); err != nil {
			m.statusMsg = err.Error()
			m.isError = true
			return m, nil
		}
		styles.ApplyTheme(m.app.Themes.Active())
		m.statusMsg = fmt.Sprintf("Theme: %s", msg.Item.ID)
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if m.width == 0 {
		return "Loading Hikari…"
	}

	activeProv := m.app.Router.ActiveName()
	activePersona := m.app.Personas.Active().Name

	activeModel := ""
	activeType := ""
	if m.app.Config != nil {
		if pcfg, ok := m.app.Config.Providers[activeProv]; ok {
			activeModel = pcfg.Model
			activeType = pcfg.Type
		}
	}

	// 1. Header Bar
	header := components.Header{
		Width:        m.width,
		ProviderName: activeProv,
		PersonaName:  activePersona,
		ModelName:    activeModel,
		SessionTitle: m.session.Title,
		Connected:    m.connected,
		ShowSidebar:  m.showSidebar,
	}.View()

	// 2. Status Bar
	statusBar := components.StatusBar{
		Width:       m.width,
		Message:     m.statusMsg,
		IsError:     m.isError,
		IsStreaming: m.isStreaming,
	}.View()

	// 3. Modal Overlay Dialog (if active)
	if m.mode != modeChat {
		selectorView := m.selector.View()
		centered := lipgloss.Place(m.width, m.height-2, lipgloss.Center, lipgloss.Center, selectorView)
		return header + "\n" + centered + "\n" + statusBar
	}

	// 4. Middle Content (Chat View + optional Sidebar)
	sidebarW := 0
	if m.showSidebar && m.width >= 90 {
		sidebarW = 28
	}

	chatW := m.width - sidebarW
	m.chatView.Width = chatW
	m.chatView.Height = m.chatHeight()
	chatStr := m.chatView.View()

	var middleStr string
	if sidebarW > 0 {
		sidebarView := components.Sidebar{
			Width:        sidebarW,
			Height:       m.chatHeight(),
			ProviderName: activeProv,
			ProviderType: activeType,
			ModelName:    activeModel,
			Connected:    m.connected,
			Persona:      m.app.Personas.Active(),
			Session:      m.session,
		}.View()
		middleStr = lipgloss.JoinHorizontal(lipgloss.Top, chatStr, sidebarView)
	} else {
		middleStr = chatStr
	}

	// 5. Input Field
	m.input.SetWidth(m.width)
	inputStr := m.input.View()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		middleStr,
		inputStr,
		statusBar,
	)
}

// chatHeight returns the number of lines available for the chat area.
func (m Model) chatHeight() int {
	// header(1) + statusbar(1) + input(3) + padding
	reserved := 6
	h := m.height - reserved
	if h < 2 {
		h = 2
	}
	return h
}

// sanitiseError trims and cleans sensitive error text.
func sanitiseError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
