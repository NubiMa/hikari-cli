package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/provider/hermes"
	"github.com/NubiMa/hikari-cli/internal/provider/ollama"
	"github.com/NubiMa/hikari-cli/internal/provider/openclaw"
	"github.com/NubiMa/hikari-cli/internal/tui/styles"
	"github.com/NubiMa/hikari-cli/internal/wizard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// configCmd returns the `hikari config` cobra subcommand tree.
func configCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "Manage Hikari configuration interactively",
		Long: `Open the interactive configuration manager.

Manage providers, personas, and themes without editing config.toml by hand.

Subcommands:
  hikari config provider    Manage AI providers
  hikari config test        Ping all configured providers
  hikari config edit        Open config.toml in $EDITOR
  hikari config path        Print the config file path
  hikari config setup       Re-run the first-run setup wizard`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigMenu()
		},
	}

	// Sub-sub commands
	root.AddCommand(configProviderCmd())
	root.AddCommand(configTestCmd())
	root.AddCommand(configEditCmd())
	root.AddCommand(configPathCmd())
	root.AddCommand(configSetupCmd())

	return root
}

// ---------------------------------------------------------------------------
// hikari config (interactive menu)
// ---------------------------------------------------------------------------

type configMenuOption struct {
	id    string
	label string
	sub   string
}

type configMenuModel struct {
	options []configMenuOption
	cursor  int
	width   int
	height  int
	done    string // chosen option id, or "quit"
}

func runConfigMenu() error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	provCount := len(cfg.Providers)
	provSub := fmt.Sprintf("%d configured", provCount)
	themeSub := cfg.UI.Theme
	if themeSub == "" {
		themeSub = "default"
	}
	personaSub := cfg.Default.Persona
	if personaSub == "" {
		personaSub = "default"
	}

	options := []configMenuOption{
		{id: "providers", label: "Providers", sub: provSub},
		{id: "test", label: "Test Connections", sub: "ping all configured providers"},
		{id: "persona", label: "Default Persona", sub: personaSub},
		{id: "theme", label: "Default Theme", sub: themeSub},
		{id: "edit", label: "Edit config.toml", sub: "open in $EDITOR"},
		{id: "path", label: "Show config path", sub: config.ConfigFile()},
		{id: "setup", label: "Re-run setup wizard", sub: "guided first-run setup"},
		{id: "quit", label: "Exit", sub: ""},
	}

	m := configMenuModel{options: options}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return err
	}

	chosen := result.(configMenuModel).done
	return dispatchConfigAction(chosen)
}

func dispatchConfigAction(id string) error {
	switch id {
	case "providers":
		return runProviderMenu()
	case "test":
		return runTestAll()
	case "persona":
		return runSetPersona()
	case "theme":
		return runSetTheme()
	case "edit":
		return openEditor()
	case "path":
		fmt.Println(config.ConfigFile())
		return nil
	case "setup":
		_, err := wizard.Run()
		return err
	case "quit", "":
		return nil
	}
	return nil
}

func (m configMenuModel) Init() tea.Cmd { return nil }

func (m configMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.done = "quit"
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.done = m.options[m.cursor].id
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m configMenuModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	var b strings.Builder
	b.WriteString(styles.SelectorTitle.Render("◈ HIKARI CONFIG"))
	b.WriteString("\n\n")

	for i, opt := range m.options {
		var line string
		if i == m.cursor {
			label := opt.label
			if opt.sub != "" {
				label += "  " + styles.Muted.Render(opt.sub)
			}
			line = styles.SelectorItemSelected.Render("› " + label)
		} else {
			label := opt.label
			if opt.sub != "" {
				label += "  " + styles.Muted.Render(opt.sub)
			}
			line = styles.SelectorItem.Render("  " + label)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [q/Esc] Exit"))

	content := styles.SelectorBox.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// ---------------------------------------------------------------------------
// Provider management menu
// ---------------------------------------------------------------------------

type providerMenuItem struct {
	name string
	cfg  config.ProviderConfig
}

func runProviderMenu() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	for {
		items := buildProviderItems(cfg)
		action, selected := showProviderMenu(items, cfg.Default.Provider)
		switch action {
		case "add":
			if err := addProviderInteractive(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			// reload
			cfg, _ = config.Load()
		case "delete":
			if selected != "" {
				delete(cfg.Providers, selected)
				if cfg.Default.Provider == selected {
					cfg.Default.Provider = ""
				}
				if err := config.WriteFromStruct(cfg); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving: %v\n", err)
				} else {
					fmt.Printf("Removed provider %q\n", selected)
				}
			}
		case "default":
			if selected != "" {
				cfg.Default.Provider = selected
				_ = config.WriteFromStruct(cfg)
				fmt.Printf("Default provider set to %q\n", selected)
			}
		case "test":
			if selected != "" {
				testProvider(selected, cfg.Providers[selected])
			}
		case "back", "quit", "":
			return nil
		}
	}
}

func buildProviderItems(cfg *config.Config) []providerMenuItem {
	var items []providerMenuItem
	for name, pcfg := range cfg.Providers {
		items = append(items, providerMenuItem{name: name, cfg: pcfg})
	}
	return items
}

type providerMenuModel struct {
	items     []providerMenuItem
	cursor    int
	width     int
	height    int
	activeID  string
	doneAct   string
	selectedN string
}

func showProviderMenu(items []providerMenuItem, activeDefault string) (action, selected string) {
	m := providerMenuModel{items: items, activeID: activeDefault}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, _ := p.Run()
	rm := result.(providerMenuModel)
	return rm.doneAct, rm.selectedN
}

func (m providerMenuModel) Init() tea.Cmd { return nil }

func (m providerMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.doneAct = "back"
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items) {
				m.cursor++
			}
		case "enter":
			if m.cursor < len(m.items) {
				m.selectedN = m.items[m.cursor].name
				m.doneAct = "test"
			} else {
				m.doneAct = "add"
			}
			return m, tea.Quit
		case "d":
			if m.cursor < len(m.items) {
				m.selectedN = m.items[m.cursor].name
				m.doneAct = "delete"
				return m, tea.Quit
			}
		case "s":
			if m.cursor < len(m.items) {
				m.selectedN = m.items[m.cursor].name
				m.doneAct = "default"
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m providerMenuModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	var b strings.Builder
	b.WriteString(styles.SelectorTitle.Render("◈ PROVIDERS"))
	b.WriteString("\n\n")

	for i, item := range m.items {
		isDefault := item.name == m.activeID
		label := item.name
		if isDefault {
			label += " " + styles.Success.Render("[default]")
		}
		sub := fmt.Sprintf("type: %s  endpoint: %s", item.cfg.Type, item.cfg.Endpoint)
		if item.cfg.Model != "" {
			sub += "  model: " + item.cfg.Model
		}

		if i == m.cursor {
			b.WriteString(styles.SelectorItemSelected.Render("› " + label))
			b.WriteString("\n")
			b.WriteString(styles.Muted.Render("    " + sub))
		} else {
			b.WriteString(styles.SelectorItem.Render("  " + label))
			b.WriteString("\n")
			b.WriteString(styles.Muted.Render("    " + sub))
		}
		b.WriteString("\n")
	}

	// "Add new" row
	addLabel := "  + Add new provider"
	if m.cursor == len(m.items) {
		addLabel = styles.SelectorItemSelected.Render("› + Add new provider")
	} else {
		addLabel = styles.SelectorItem.Render(addLabel)
	}
	b.WriteString(addLabel)
	b.WriteString("\n\n")
	b.WriteString(styles.Muted.Render("[↑/↓] Navigate  ·  [Enter] Test  ·  [s] Set default  ·  [d] Delete  ·  [Esc] Back"))

	content := styles.SelectorBox.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// ---------------------------------------------------------------------------
// Add provider flow (calls into wizard internals)
// ---------------------------------------------------------------------------

func addProviderInteractive(_ *config.Config) error {
	// Guide user to the wizard or manual edit — full form TUI is in hikari config setup.
	fmt.Println()
	fmt.Println(styles.Bold.Render("Adding a new provider"))
	fmt.Println(styles.Muted.Render("Run `hikari config setup` to use the guided wizard, or"))
	fmt.Println(styles.Muted.Render("edit " + config.ConfigFile() + " directly."))
	fmt.Println()
	return nil
}

// testProvider pings a single provider and prints the result.
func testProvider(name string, pcfg config.ProviderConfig) {
	fmt.Printf("Testing %s (%s)…\n", name, pcfg.Endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	var prov provider.Provider
	var err error
	switch pcfg.Type {
	case "ollama":
		prov, err = ollama.New(name, pcfg)
	case "openclaw":
		prov, err = openclaw.New(name, pcfg)
	case "hermes":
		prov, err = hermes.New(name, pcfg)
	default:
		fmt.Printf("  %s Unknown provider type %q\n", styles.Error.Render("✗"), pcfg.Type)
		return
	}
	if err != nil {
		fmt.Printf("  %s %v\n", styles.Error.Render("✗"), err)
		return
	}

	if err := prov.Connect(ctx); err != nil {
		fmt.Printf("  %s %v\n", styles.Error.Render("✗"), err)
		return
	}

	status, err := prov.Status(ctx)
	if err != nil {
		fmt.Printf("  %s %v\n", styles.Error.Render("✗"), err)
		return
	}

	if status.Connected {
		latency := ""
		if status.Latency > 0 {
			latency = fmt.Sprintf(" (%dms)", status.Latency.Milliseconds())
		}
		fmt.Printf("  %s Connected%s\n", styles.Success.Render("✓"), latency)
	} else {
		fmt.Printf("  %s Offline: %s\n", styles.Error.Render("✗"), status.Message)
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// Persona/Theme selectors (uses TUI selector)
// ---------------------------------------------------------------------------

type simplePickerModel struct {
	title    string
	items    []struct{ id, label, sub string }
	cursor   int
	activeID string
	chosen   string
	width    int
	height   int
}

func (m simplePickerModel) Init() tea.Cmd { return nil }

func (m simplePickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.chosen = ""
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.items[m.cursor].id
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m simplePickerModel) View() string {
	if m.width == 0 {
		return "Loading…"
	}
	var b strings.Builder
	b.WriteString(styles.SelectorTitle.Render("◈ " + strings.ToUpper(m.title)))
	b.WriteString("\n\n")

	for i, item := range m.items {
		isActive := item.id == m.activeID
		label := item.label
		if isActive {
			label += " " + styles.Success.Render("[active]")
		}
		if i == m.cursor {
			b.WriteString(styles.SelectorItemSelected.Render("› " + label))
			if item.sub != "" {
				b.WriteString("\n")
				b.WriteString(styles.Muted.Render("    " + item.sub))
			}
		} else {
			b.WriteString(styles.SelectorItem.Render("  " + label))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.Muted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [Esc] Cancel"))

	content := styles.SelectorBox.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func pickFromList(title string, items []struct{ id, label, sub string }, activeID string) string {
	m := simplePickerModel{title: title, items: items, activeID: activeID}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, _ := p.Run()
	return result.(simplePickerModel).chosen
}

func runSetPersona() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	items := []struct{ id, label, sub string }{
		{"nino", "Nino", "Friendly conversational assistant — warm and helpful"},
		{"developer", "Developer", "Code-focused assistant — concise, technical, precise"},
		{"sysadmin", "SysAdmin", "System admin expert — commands, logs, infrastructure"},
		{"default", "Default", "Generic helpful assistant — neutral and balanced"},
	}

	chosen := pickFromList("Select Default Persona", items, cfg.Default.Persona)
	if chosen == "" {
		return nil
	}

	cfg.Default.Persona = chosen
	if err := config.WriteFromStruct(cfg); err != nil {
		return err
	}
	fmt.Printf("%s Default persona set to %q\n", styles.Success.Render("✓"), chosen)
	return nil
}

func runSetTheme() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	items := []struct{ id, label, sub string }{
		{"default", "Default (Violet)", "Deep violet and indigo — the classic Hikari look"},
		{"minimal", "Minimal", "Low-contrast, distraction-free monochrome"},
		{"tokyo-night", "Tokyo Night", "Inspired by the popular VS Code Tokyo Night palette"},
	}

	chosen := pickFromList("Select Default Theme", items, cfg.UI.Theme)
	if chosen == "" {
		return nil
	}

	cfg.UI.Theme = chosen
	if err := config.WriteFromStruct(cfg); err != nil {
		return err
	}
	fmt.Printf("%s Theme set to %q\n", styles.Success.Render("✓"), chosen)
	return nil
}

// ---------------------------------------------------------------------------
// Test all providers
// ---------------------------------------------------------------------------

func runTestAll() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Providers) == 0 {
		fmt.Println(styles.Muted.Render("No providers configured. Run `hikari config setup` to add one."))
		return nil
	}

	fmt.Println(styles.Bold.Render("Testing all configured providers…"))
	fmt.Println()

	for name, pcfg := range cfg.Providers {
		testProvider(name, pcfg)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Open editor
// ---------------------------------------------------------------------------

func openEditor() error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	cfgPath := config.ConfigFile()
	if !config.Exists() {
		if err := config.WriteDefault(); err != nil {
			return err
		}
	}

	cmd := exec.Command(editor, cfgPath) //nolint:gosec // editor is from user env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ---------------------------------------------------------------------------
// Subcommand wiring
// ---------------------------------------------------------------------------

func configProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Manage AI providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProviderMenu()
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listProviders()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "test [name]",
		Short: "Ping a provider (or all if no name given)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				return runTestAll()
			}
			pcfg, ok := cfg.Providers[args[0]]
			if !ok {
				return fmt.Errorf("provider %q not found in config", args[0])
			}
			testProvider(args[0], pcfg)
			return nil
		},
	})

	return cmd
}

func listProviders() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if len(cfg.Providers) == 0 {
		fmt.Println(styles.Muted.Render("No providers configured."))
		fmt.Println(styles.Muted.Render("Run `hikari config setup` to add one, or edit " + config.ConfigFile()))
		return nil
	}

	fmt.Println(styles.Bold.Render("Configured providers:"))
	fmt.Println()

	for name, pcfg := range cfg.Providers {
		isDefault := name == cfg.Default.Provider
		def := ""
		if isDefault {
			def = " " + styles.Success.Render("[default]")
		}
		fmt.Printf("  %s%s\n", styles.Bold.Render(name), def)
		fmt.Printf("    Type:     %s\n", pcfg.Type)
		fmt.Printf("    Endpoint: %s\n", pcfg.Endpoint)
		if pcfg.Model != "" {
			fmt.Printf("    Model:    %s\n", pcfg.Model)
		}
		fmt.Println()
	}
	return nil
}

func configTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Ping all configured providers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTestAll()
		},
	}
}

func configEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open config.toml in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			return openEditor()
		},
	}
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(config.ConfigFile())
		},
	}
}

func configSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Re-run the first-run setup wizard",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := wizard.Run()
			if err != nil {
				return err
			}
			if path != "" {
				fmt.Printf("%s Config written to %s\n", styles.Success.Render("✓"), path)
			}
			return nil
		},
	}
}
