// Package wizard implements Hikari's first-run interactive setup wizard.
//
// The wizard runs as a full-screen Bubble Tea program only when no config.toml
// exists. It guides the user through: provider type selection, provider
// configuration, a live connection test, persona selection, theme selection,
// and finally writes the config file.
package wizard

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/provider/hermes"
	"github.com/NubiMa/hikari-cli/internal/provider/ollama"
	"github.com/NubiMa/hikari-cli/internal/provider/openclaw"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------------------------------------------------------------------
// Colors (always use default theme — wizard runs before config is loaded)
// ---------------------------------------------------------------------------

var (
	colPrimary = lipgloss.Color("#7C3AED")
	colAccent  = lipgloss.Color("#A78BFA")
	colSuccess = lipgloss.Color("#10B981")
	colError   = lipgloss.Color("#F87171")
	colMuted   = lipgloss.Color("#9CA3AF")
	colText    = lipgloss.Color("#F9FAFB")
	colBorder  = lipgloss.Color("#4C1D95")
)

var (
	styleBold    = lipgloss.NewStyle().Bold(true).Foreground(colText)
	styleAccent  = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleMuted   = lipgloss.NewStyle().Foreground(colMuted)
	styleSuccess = lipgloss.NewStyle().Bold(true).Foreground(colSuccess)
	styleError   = lipgloss.NewStyle().Bold(true).Foreground(colError)
	styleBox     = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(1, 2)
	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(colText).
			Background(colPrimary).
			Padding(0, 1)
	styleUnselected = lipgloss.NewStyle().Foreground(colMuted).Padding(0, 1)
	styleTitle      = lipgloss.NewStyle().Bold(true).Foreground(colAccent).MarginBottom(1)
	styleHint       = lipgloss.NewStyle().Foreground(colMuted).Italic(true)
	styleLabelFocus = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleLabelDim   = lipgloss.NewStyle().Foreground(colMuted)
)

// ---------------------------------------------------------------------------
// Step constants
// ---------------------------------------------------------------------------

type stepID int

const (
	stepWelcome stepID = iota
	stepProviderType
	stepProviderForm
	stepPingTest
	stepPersona
	stepTheme
	stepSummary
	stepDone
)

// ---------------------------------------------------------------------------
// Provider type definitions
// ---------------------------------------------------------------------------

type providerType struct {
	id          string
	label       string
	description string
	defaults    providerDefaults
}

type providerDefaults struct {
	name     string
	endpoint string
	model    string
	timeout  string
}

var providerTypes = []providerType{
	{
		id:          "ollama-local",
		label:       "Ollama (Local)",
		description: "Run open-source LLMs locally — offline, private, free",
		defaults:    providerDefaults{name: "ollama-local", endpoint: "http://127.0.0.1:11434", model: "llama3.2", timeout: "120"},
	},
	{
		id:          "ollama-remote",
		label:       "Ollama (Remote VPS / LAN)",
		description: "Connect to Ollama on a remote server or GPU machine",
		defaults:    providerDefaults{name: "ollama-vps", endpoint: "https://ollama.yourserver.com", model: "llama3.2", timeout: "180"},
	},
	{
		id:          "openclaw",
		label:       "OpenClaw (Autonomous Agent)",
		description: "Autonomous agent with tool execution and shell access",
		defaults:    providerDefaults{name: "openclaw-vps", endpoint: "https://agent.yourserver.com", model: "", timeout: "180"},
	},
	{
		id:          "hermes",
		label:       "Hermes (Agent Pipeline)",
		description: "Multi-turn agent backend with tool calling and planning",
		defaults:    providerDefaults{name: "hermes-local", endpoint: "http://127.0.0.1:8080", model: "", timeout: "120"},
	},
	{
		id:          "skip",
		label:       "Skip for now",
		description: "I'll configure providers manually in config.toml",
		defaults:    providerDefaults{},
	},
}

var personaChoices = []struct{ id, label, desc string }{
	{"hikari", "Hikari", "Friendly conversational assistant — warm, helpful, human-like"},
	{"developer", "Developer", "Concise code-focused assistant — technical, direct, precise"},
	{"sysadmin", "SysAdmin", "System administration expert — commands, logs, infrastructure"},
	{"default", "Default", "Generic helpful assistant — neutral and balanced"},
}

var themeChoices = []struct{ id, label, desc string }{
	{"default", "Default (Violet)", "Deep violet and indigo — the classic Hikari look"},
	{"minimal", "Minimal", "Low-contrast, distraction-free monochrome"},
	{"tokyo-night", "Tokyo Night", "Inspired by the popular VS Code Tokyo Night palette"},
}

// ---------------------------------------------------------------------------
// Field form for provider configuration
// ---------------------------------------------------------------------------

type formField struct {
	label    string
	hint     string
	input    textinput.Model
	masked   bool // token fields
	required bool
}

func newField(label, placeholder, defaultVal, hint string, masked, required bool) formField {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.SetValue(defaultVal)
	ti.CharLimit = 512
	if masked {
		ti.EchoMode = textinput.EchoPassword
		ti.EchoCharacter = '•'
	}
	return formField{label: label, hint: hint, input: ti, masked: masked, required: required}
}

// ---------------------------------------------------------------------------
// Ping result message
// ---------------------------------------------------------------------------

type pingResultMsg struct {
	ok      bool
	message string
}

// ---------------------------------------------------------------------------
// Wizard model
// ---------------------------------------------------------------------------

// Model is the root Bubble Tea model for the setup wizard.
type Model struct {
	step   stepID
	width  int
	height int

	// Step: provider type
	provTypeCursor int

	// Step: provider form
	fields       []formField
	focusedField int

	// Step: ping
	pinging    bool
	pingResult *pingResultMsg

	// Step: persona
	personaCursor int

	// Step: theme
	themeCursor int

	// Collected values
	chosenProvType string // e.g. "ollama-local"
	providerName   string
	providerCfg    config.ProviderConfig

	// Final status
	savedPath string
	saveErr   error

	skippedProvider bool
}

// New creates a fresh wizard model.
func New() *Model {
	return &Model{step: stepWelcome}
}

// Run starts the wizard as a full-screen Bubble Tea program and returns the
// path to the written config file on success.
func Run() (string, error) {
	m := New()
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", err
	}
	fm := result.(Model)
	if fm.saveErr != nil {
		return "", fm.saveErr
	}
	return fm.savedPath, nil
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	return nil
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case pingResultMsg:
		m.pinging = false
		m.pingResult = &msg
		return m, nil

	case tea.KeyMsg:
		switch m.step {
		case stepWelcome:
			return m.updateWelcome(msg)
		case stepProviderType:
			return m.updateProviderType(msg)
		case stepProviderForm:
			return m.updateProviderForm(msg)
		case stepPingTest:
			return m.updatePingTest(msg)
		case stepPersona:
			return m.updatePersona(msg)
		case stepTheme:
			return m.updateTheme(msg)
		case stepSummary:
			return m.updateSummary(msg)
		case stepDone:
			return m, tea.Quit
		}
	}

	// Delegate input to focused form field
	if m.step == stepProviderForm && len(m.fields) > 0 {
		var cmd tea.Cmd
		m.fields[m.focusedField].input, cmd = m.fields[m.focusedField].input.Update(msg)
		return m, cmd
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Step updates
// ---------------------------------------------------------------------------

func (m Model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "enter", " ":
		m.step = stepProviderType
	}
	return m, nil
}

func (m Model) updateProviderType(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.provTypeCursor > 0 {
			m.provTypeCursor--
		}
	case "down", "j":
		if m.provTypeCursor < len(providerTypes)-1 {
			m.provTypeCursor++
		}
	case "enter":
		chosen := providerTypes[m.provTypeCursor]
		m.chosenProvType = chosen.id
		if chosen.id == "skip" {
			m.skippedProvider = true
			m.step = stepPersona
		} else {
			m.fields = buildFields(chosen)
			m.focusedField = 0
			m.fields[0].input.Focus()
			m.step = stepProviderForm
		}
	case "esc":
		m.step = stepWelcome
	}
	return m, nil
}

func buildFields(pt providerType) []formField {
	d := pt.defaults
	fields := []formField{
		newField("Provider name", d.name, d.name, "A unique identifier for this provider (e.g. ollama-local)", false, true),
		newField("Endpoint URL", d.endpoint, d.endpoint, "The base URL of the provider API", false, true),
	}
	if pt.id == "ollama-local" || pt.id == "ollama-remote" {
		fields = append(fields, newField("Model", d.model, d.model, "Default model to use (e.g. llama3.2, qwen2.5:7b)", false, true))
	}
	fields = append(fields, newField("API token / bearer", "", "", "Optional. Leave blank if not required. Or set HIKARI_<NAME>_TOKEN env var", true, false))
	fields = append(fields, newField("Timeout (seconds)", d.timeout, d.timeout, "HTTP timeout for requests to this provider", false, false))
	return fields
}

func (m Model) updateProviderForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.step = stepProviderType
		return m, nil
	case "tab", "down":
		m.fields[m.focusedField].input.Blur()
		m.focusedField = (m.focusedField + 1) % len(m.fields)
		m.fields[m.focusedField].input.Focus()
		return m, textinput.Blink
	case "shift+tab", "up":
		m.fields[m.focusedField].input.Blur()
		m.focusedField = (m.focusedField - 1 + len(m.fields)) % len(m.fields)
		m.fields[m.focusedField].input.Focus()
		return m, textinput.Blink
	case "enter":
		// On last field: advance; otherwise move to next field
		if m.focusedField == len(m.fields)-1 {
			// Validate required fields
			if err := m.validateFields(); err != nil {
				// Stay on form — show error via statusMsg (reuse last field hint)
				m.fields[m.focusedField].hint = "⚠ " + err.Error()
				return m, nil
			}
			m.collectProviderConfig()
			m.step = stepPingTest
			m.pinging = true
			return m, m.doPing()
		}
		m.fields[m.focusedField].input.Blur()
		m.focusedField++
		m.fields[m.focusedField].input.Focus()
		return m, textinput.Blink
	}

	var cmd tea.Cmd
	m.fields[m.focusedField].input, cmd = m.fields[m.focusedField].input.Update(msg)
	return m, cmd
}

func (m *Model) validateFields() error {
	for _, f := range m.fields {
		if f.required && strings.TrimSpace(f.input.Value()) == "" {
			return fmt.Errorf("%s is required", f.label)
		}
	}
	return nil
}

func (m *Model) collectProviderConfig() {
	fieldMap := make(map[string]string)
	for _, f := range m.fields {
		fieldMap[f.label] = strings.TrimSpace(f.input.Value())
	}

	m.providerName = fieldMap["Provider name"]
	if m.providerName == "" {
		m.providerName = providerTypes[m.provTypeCursor].defaults.name
	}

	timeout := 120
	if t := fieldMap["Timeout (seconds)"]; t != "" {
		if n, err := strconv.Atoi(t); err == nil {
			timeout = n
		}
	}

	provType := "ollama"
	switch m.chosenProvType {
	case "openclaw":
		provType = "openclaw"
	case "hermes":
		provType = "hermes"
	}

	m.providerCfg = config.ProviderConfig{
		Type:           provType,
		Endpoint:       fieldMap["Endpoint URL"],
		Model:          fieldMap["Model"],
		Token:          fieldMap["API token / bearer"],
		TimeoutSeconds: timeout,
	}
}

func (m Model) doPing() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		var prov provider.Provider
		var err error

		switch m.chosenProvType {
		case "ollama-local", "ollama-remote":
			prov, err = ollama.New(m.providerName, m.providerCfg)
		case "openclaw":
			prov, err = openclaw.New(m.providerName, m.providerCfg)
		case "hermes":
			prov, err = hermes.New(m.providerName, m.providerCfg)
		default:
			return pingResultMsg{ok: false, message: "unknown provider type"}
		}

		if err != nil {
			return pingResultMsg{ok: false, message: err.Error()}
		}

		if err := prov.Connect(ctx); err != nil {
			return pingResultMsg{ok: false, message: err.Error()}
		}

		status, err := prov.Status(ctx)
		if err != nil {
			return pingResultMsg{ok: false, message: err.Error()}
		}
		if !status.Connected {
			msg := status.Message
			if msg == "" {
				msg = "provider offline"
			}
			return pingResultMsg{ok: false, message: msg}
		}

		msg := "Connected"
		if status.Latency > 0 {
			msg += fmt.Sprintf(" (%dms)", status.Latency.Milliseconds())
		}
		return pingResultMsg{ok: true, message: msg}
	}
}

func (m Model) updatePingTest(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pinging {
		return m, nil // wait for ping
	}
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter", " ":
		m.step = stepPersona
	case "esc":
		m.step = stepProviderForm
		m.pingResult = nil
	case "r":
		m.pinging = true
		m.pingResult = nil
		return m, m.doPing()
	}
	return m, nil
}

func (m Model) updatePersona(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.personaCursor > 0 {
			m.personaCursor--
		}
	case "down", "j":
		if m.personaCursor < len(personaChoices)-1 {
			m.personaCursor++
		}
	case "enter":
		m.step = stepTheme
	case "esc":
		if m.skippedProvider {
			m.step = stepProviderType
		} else {
			m.step = stepPingTest
		}
	}
	return m, nil
}

func (m Model) updateTheme(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.themeCursor > 0 {
			m.themeCursor--
		}
	case "down", "j":
		if m.themeCursor < len(themeChoices)-1 {
			m.themeCursor++
		}
	case "enter":
		m.step = stepSummary
	case "esc":
		m.step = stepPersona
	}
	return m, nil
}

func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "enter", "y", "Y":
		return m.doSave()
	case "n", "N", "esc":
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) doSave() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		if err := config.EnsureDirs(); err != nil {
			m.saveErr = err
			return nil
		}

		cfg := buildConfig(m)
		if err := config.WriteFromStruct(cfg); err != nil {
			m.saveErr = err
			return nil
		}
		m.savedPath = config.ConfigFile()
		m.step = stepDone
		return nil
	}
}

// buildConfig assembles a Config struct from the wizard's collected values.
func buildConfig(m Model) *config.Config {
	persona := personaChoices[m.personaCursor].id
	thm := themeChoices[m.themeCursor].id

	cfg := &config.Config{
		Default: config.DefaultConfig{
			Persona: persona,
		},
		Providers: make(map[string]config.ProviderConfig),
		UI: config.UIConfig{
			Theme: thm,
			ASCII: "default",
		},
	}

	if !m.skippedProvider && m.providerName != "" {
		cfg.Providers[m.providerName] = m.providerCfg
		cfg.Default.Provider = m.providerName
		cfg.Default.Model = m.providerCfg.Model
	}

	return cfg
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	var content string
	switch m.step {
	case stepWelcome:
		content = m.viewWelcome()
	case stepProviderType:
		content = m.viewProviderType()
	case stepProviderForm:
		content = m.viewProviderForm()
	case stepPingTest:
		content = m.viewPingTest()
	case stepPersona:
		content = m.viewPersona()
	case stepTheme:
		content = m.viewTheme()
	case stepSummary:
		content = m.viewSummary()
	case stepDone:
		content = m.viewDone()
	}

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

const hikariBanner = `    ██╗  ██╗██╗██╗  ██╗ █████╗ ██████╗ ██╗
    ██║  ██║██║██║ ██╔╝██╔══██╗██╔══██╗██║
    ███████║██║█████═╝ ███████║██████╔╝██║
    ██╔══██║██║██╔═██╗ ██╔══██║██╔══██╗██║
    ██║  ██║██║██║  ██╗██║  ██║██║  ██║██║
    ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝`

func (m Model) viewWelcome() string {
	banner := styleAccent.Render(hikariBanner)

	var b strings.Builder
	b.WriteString(banner)
	b.WriteString("\n\n")
	b.WriteString(styleTitle.Render("Welcome to Hikari!"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Looks like this is your first run. Let's get you set up in under 2 minutes."))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("We'll help you:"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Connect to your AI provider (Ollama, OpenClaw, or Hermes)"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Choose a default persona"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Pick a UI theme"))
	b.WriteString("\n\n")
	b.WriteString(styleHint.Render("[Enter] Get started  ·  [q / Ctrl+C] Skip & quit"))

	return styleBox.Render(b.String())
}

func (m Model) viewProviderType() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Step 1 / 4  ·  Choose Provider Type"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Which AI backend do you want to connect to?"))
	b.WriteString("\n\n")

	for i, pt := range providerTypes {
		var row string
		if i == m.provTypeCursor {
			row = styleSelected.Render("› " + pt.label)
			b.WriteString(row)
			b.WriteString("\n")
			b.WriteString(styleHint.Padding(0, 3).Render(pt.description))
		} else {
			row = styleUnselected.Render("  " + pt.label)
			b.WriteString(row)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [Esc] Back"))

	return styleBox.Render(b.String())
}

func (m Model) viewProviderForm() string {
	chosen := providerTypes[m.provTypeCursor]

	var b strings.Builder
	b.WriteString(styleTitle.Render(fmt.Sprintf("Step 2 / 4  ·  Configure %s", chosen.label)))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Fill in the details below. Tab / Enter to move between fields."))
	b.WriteString("\n\n")

	for i, f := range m.fields {
		focused := i == m.focusedField
		var lbl string
		if focused {
			lbl = styleLabelFocus.Render("  ❯ " + f.label)
		} else {
			lbl = styleLabelDim.Render("    " + f.label)
		}
		b.WriteString(lbl)
		b.WriteString("\n")

		inputStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
		if focused {
			inputStyle = inputStyle.BorderForeground(colPrimary)
		} else {
			inputStyle = inputStyle.BorderForeground(colBorder)
		}

		b.WriteString(inputStyle.Width(52).Render(f.input.View()))
		b.WriteString("\n")

		if f.hint != "" {
			b.WriteString(styleHint.Render("      " + f.hint))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styleMuted.Render("[Tab/Enter] Next field  ·  [Enter on last] Test connection  ·  [Esc] Back"))
	return styleBox.Render(b.String())
}

func (m Model) viewPingTest() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Step 3 / 4  ·  Testing Connection"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(fmt.Sprintf("Connecting to: %s", m.providerCfg.Endpoint)))
	b.WriteString("\n\n")

	if m.pinging {
		b.WriteString(styleAccent.Render("  ⠿ Testing connection…"))
		b.WriteString("\n\n")
		b.WriteString(styleMuted.Render("Please wait…"))
	} else if m.pingResult != nil {
		if m.pingResult.ok {
			b.WriteString(styleSuccess.Render("  ✓ " + m.pingResult.message))
			b.WriteString("\n\n")
			b.WriteString(styleMuted.Render("Provider is reachable and ready."))
			b.WriteString("\n\n")
			b.WriteString(styleMuted.Render("[Enter] Continue  ·  [r] Retry  ·  [Esc] Edit settings"))
		} else {
			b.WriteString(styleError.Render("  ✗ Connection failed"))
			b.WriteString("\n")
			b.WriteString(styleMuted.Render("  " + m.pingResult.message))
			b.WriteString("\n\n")
			b.WriteString(styleMuted.Render("You can still continue — Hikari will retry when you start a session."))
			b.WriteString("\n\n")
			b.WriteString(styleMuted.Render("[Enter] Continue anyway  ·  [r] Retry  ·  [Esc] Edit settings"))
		}
	}

	return styleBox.Render(b.String())
}

func (m Model) viewPersona() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Step 3 / 4  ·  Choose Default Persona"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Personas shape the AI's personality and system prompt."))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("You can always switch with /persona in the TUI."))
	b.WriteString("\n\n")

	for i, p := range personaChoices {
		if i == m.personaCursor {
			b.WriteString(styleSelected.Render("› " + p.label))
			b.WriteString("\n")
			b.WriteString(styleHint.Padding(0, 3).Render(p.desc))
		} else {
			b.WriteString(styleUnselected.Render("  " + p.label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [Esc] Back"))
	return styleBox.Render(b.String())
}

func (m Model) viewTheme() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Step 4 / 4  ·  Choose UI Theme"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("Pick the visual style for the Hikari TUI."))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("You can always switch with /theme in the TUI."))
	b.WriteString("\n\n")

	for i, t := range themeChoices {
		if i == m.themeCursor {
			b.WriteString(styleSelected.Render("› " + t.label))
			b.WriteString("\n")
			b.WriteString(styleHint.Padding(0, 3).Render(t.desc))
		} else {
			b.WriteString(styleUnselected.Render("  " + t.label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleMuted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [Esc] Back"))
	return styleBox.Render(b.String())
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(styleTitle.Render("Almost done! Here's what will be saved:"))
	b.WriteString("\n\n")

	cfgPath := config.ConfigFile()
	b.WriteString(styleMuted.Render("Config file: "))
	b.WriteString(styleAccent.Render(cfgPath))
	b.WriteString("\n\n")

	b.WriteString(styleBold.Render("  Provider"))
	b.WriteString("\n")
	if m.skippedProvider {
		b.WriteString(styleMuted.Render("    None — skipped (add providers later with `hikari config`)"))
	} else {
		b.WriteString(styleMuted.Render(fmt.Sprintf("    Name:     %s", m.providerName)))
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(fmt.Sprintf("    Type:     %s", m.providerCfg.Type)))
		b.WriteString("\n")
		b.WriteString(styleMuted.Render(fmt.Sprintf("    Endpoint: %s", m.providerCfg.Endpoint)))
		if m.providerCfg.Model != "" {
			b.WriteString("\n")
			b.WriteString(styleMuted.Render(fmt.Sprintf("    Model:    %s", m.providerCfg.Model)))
		}
	}
	b.WriteString("\n\n")

	b.WriteString(styleBold.Render("  Persona"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("    " + personaChoices[m.personaCursor].label))
	b.WriteString("\n\n")

	b.WriteString(styleBold.Render("  Theme"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("    " + themeChoices[m.themeCursor].label))
	b.WriteString("\n\n")

	b.WriteString(styleMuted.Render("Save configuration?  "))
	b.WriteString(styleAccent.Render("[Enter / y]"))
	b.WriteString(styleMuted.Render(" Yes   "))
	b.WriteString(styleMuted.Render("[n / Esc]"))
	b.WriteString(styleMuted.Render(" No, quit"))
	return styleBox.Render(b.String())
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(styleSuccess.Render("  ✓ Configuration saved!"))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("Config written to: "))
	b.WriteString(styleAccent.Render(m.savedPath))
	b.WriteString("\n\n")
	b.WriteString(styleBold.Render("You're all set. Press any key to start Hikari."))
	b.WriteString("\n\n")
	b.WriteString(styleMuted.Render("Tips:"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Type /provider to switch providers"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Type /persona to switch personas"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Type /theme to switch themes"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render("  • Run `hikari config` anytime to change settings"))
	return styleBox.Render(b.String())
}

// RunHeadless runs the wizard in a non-interactive terminal (e.g. piped),
// printing messages to stderr and writing a minimal config.
func RunHeadless() error {
	fmt.Fprintln(os.Stderr, "No config found. Run `hikari --init` or `hikari config` to set up.")
	return nil
}
