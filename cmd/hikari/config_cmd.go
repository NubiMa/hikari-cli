package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/assets"
	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/persona"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/provider/custom"
	"github.com/NubiMa/hikari-cli/internal/provider/hermes"
	"github.com/NubiMa/hikari-cli/internal/provider/ollama"
	"github.com/NubiMa/hikari-cli/internal/provider/openclaw"
	"github.com/NubiMa/hikari-cli/internal/theme"
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
  hikari config openclaw    Configure OpenClaw provider (auto-updates config.toml)
  hikari config hermes      Configure Hermes provider (auto-updates config.toml)
  hikari config ollama      Configure Ollama provider (auto-updates config.toml)
  hikari config custom      Configure custom/OpenAI-compatible provider (auto-updates config.toml)
  hikari config provider    Manage AI providers interactively
  hikari config persona     Set or edit default persona (auto-updates config.toml)
  hikari config theme       Set or edit default UI theme (auto-updates config.toml)
  hikari config test        Ping all configured providers
  hikari config edit        Open config.toml, persona, or theme in $EDITOR
  hikari config path        Print the config file path
  hikari config setup       Re-run the first-run setup wizard`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigMenu()
		},
	}

	// Sub-sub commands
	root.AddCommand(configOpenClawCmd())
	root.AddCommand(configHermesCmd())
	root.AddCommand(configOllamaCmd())
	root.AddCommand(configCustomCmd())
	root.AddCommand(configProviderCmd())
	root.AddCommand(configPersonaCmd())
	root.AddCommand(configThemeCmd())
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
		{id: "edit-persona", label: "Edit Persona", sub: fmt.Sprintf("edit %s in $EDITOR", personaSub)},
		{id: "edit-theme", label: "Edit Theme", sub: fmt.Sprintf("edit %s in $EDITOR", themeSub)},
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
	case "edit-persona":
		return openPersonaEditor("")
	case "edit-theme":
		return openThemeEditor("")
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
			if err := addProviderInteractive(); err != nil {
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
// Provider configuration flows
// ---------------------------------------------------------------------------

func addProviderInteractive() error {
	items := []struct{ id, label, sub string }{
		{"openclaw", "OpenClaw (Autonomous Agent)", "Tool execution & shell access on remote server"},
		{"hermes", "Hermes (Agent Pipeline)", "Multi-turn agent pipeline with tool calling & planning"},
		{"ollama", "Ollama (Local / Remote)", "Run models like llama3.2, mistral, qwen"},
		{"custom", "Custom / OpenAI-Compatible", "Groq, Together AI, LM Studio, OpenRouter, vLLM, and more"},
	}
	chosen := pickFromList("Select Provider Type to Add", items, "")
	switch chosen {
	case "openclaw":
		return runConfigureOpenClaw("openclaw-vps", "", "", 180, false, false)
	case "hermes":
		return runConfigureHermes("hermes-local", "", "", 120, false, false)
	case "ollama":
		return runConfigureOllama("ollama-local", "", "", "", 120, false, false)
	case "custom":
		return runConfigureCustom("my-custom", "", "", "", "/models", 60, false, false)
	default:
		return nil
	}
}

func configOpenClawCmd() *cobra.Command {
	var (
		name     string
		endpoint string
		token    string
		timeout  int
		makeDef  bool
		skipTest bool
	)
	cmd := &cobra.Command{
		Use:   "openclaw",
		Short: "Configure OpenClaw provider (auto-updates config.toml)",
		Long: `Configure an OpenClaw provider (autonomous agent with tool execution & shell access).
Can be run interactively or with command-line flags. Automatically saves changes to config.toml.

Examples:
  hikari config openclaw
  hikari config openclaw --endpoint https://agent.yourserver.com --token mytoken --default
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigureOpenClaw(name, endpoint, token, timeout, makeDef, skipTest)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "openclaw-vps", "Provider name identifier in config")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "OpenClaw server endpoint URL")
	cmd.Flags().StringVarP(&token, "token", "t", "", "API token / bearer token")
	cmd.Flags().IntVar(&timeout, "timeout", 180, "Timeout in seconds")
	cmd.Flags().BoolVarP(&makeDef, "default", "d", false, "Set as active default provider")
	cmd.Flags().BoolVar(&skipTest, "skip-test", false, "Skip connection ping test")
	return cmd
}

func configHermesCmd() *cobra.Command {
	var (
		name     string
		endpoint string
		token    string
		timeout  int
		makeDef  bool
		skipTest bool
	)
	cmd := &cobra.Command{
		Use:   "hermes",
		Short: "Configure Hermes provider (auto-updates config.toml)",
		Long: `Configure a Hermes provider (multi-turn agent pipeline with tool calling & planning).
Can be run interactively or with command-line flags. Automatically saves changes to config.toml.

Examples:
  hikari config hermes
  hikari config hermes --endpoint http://127.0.0.1:8080 --default
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigureHermes(name, endpoint, token, timeout, makeDef, skipTest)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "hermes-local", "Provider name identifier in config")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "Hermes server endpoint URL")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Optional API token / bearer token")
	cmd.Flags().IntVar(&timeout, "timeout", 120, "Timeout in seconds")
	cmd.Flags().BoolVarP(&makeDef, "default", "d", false, "Set as active default provider")
	cmd.Flags().BoolVar(&skipTest, "skip-test", false, "Skip connection ping test")
	return cmd
}

func configOllamaCmd() *cobra.Command {
	var (
		name     string
		endpoint string
		model    string
		token    string
		timeout  int
		makeDef  bool
		skipTest bool
	)
	cmd := &cobra.Command{
		Use:   "ollama",
		Short: "Configure Ollama provider (auto-updates config.toml)",
		Long: `Configure an Ollama provider (local or remote LLM server).
Can be run interactively or with command-line flags. Automatically saves changes to config.toml.

Examples:
  hikari config ollama
  hikari config ollama --endpoint http://127.0.0.1:11434 --model llama3.2 --default
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigureOllama(name, endpoint, model, token, timeout, makeDef, skipTest)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "ollama-local", "Provider name identifier in config")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "Ollama server endpoint URL")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model name (e.g. llama3.2, mistral)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Optional API token")
	cmd.Flags().IntVar(&timeout, "timeout", 120, "Timeout in seconds")
	cmd.Flags().BoolVarP(&makeDef, "default", "d", false, "Set as active default provider")
	cmd.Flags().BoolVar(&skipTest, "skip-test", false, "Skip connection ping test")
	return cmd
}

func runConfigureOpenClaw(name, endpoint, token string, timeout int, makeDef, skipTest bool) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.ProviderConfig),
		}
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	if name == "" {
		name = "openclaw-vps"
	}

	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure OpenClaw Provider"))
		fmt.Println(styles.Muted.Render("Autonomous agent backend with tool calling, shell commands & memory."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting := cfg.Providers[name]

		defEndpoint := "https://agent.yourserver.com"
		if hasExisting && existing.Endpoint != "" {
			defEndpoint = existing.Endpoint
		}
		if endpoint != "" {
			defEndpoint = endpoint
		}
		endpoint = promptInput(reader, "Endpoint URL", defEndpoint)

		tokenPrompt := "API token / bearer key"
		defToken := ""
		if hasExisting && existing.Token != "" {
			tokenPrompt = "API token (press Enter to keep existing token)"
			defToken = existing.Token
		}
		enteredToken := promptInput(reader, tokenPrompt, "")
		if enteredToken != "" {
			token = enteredToken
		} else if defToken != "" {
			token = defToken
		}

		defTimeout := "180"
		if hasExisting && existing.TimeoutSeconds > 0 {
			defTimeout = strconv.Itoa(existing.TimeoutSeconds)
		} else if timeout > 0 {
			defTimeout = strconv.Itoa(timeout)
		}
		tStr := promptInput(reader, "Timeout in seconds", defTimeout)
		if n, err := strconv.Atoi(tStr); err == nil && n > 0 {
			timeout = n
		}

		defIsDefault := cfg.Default.Provider == "" || cfg.Default.Provider == name
		if !makeDef {
			makeDef = promptConfirm(reader, "Set as default provider?", defIsDefault)
		}
		fmt.Println()
	}

	if endpoint == "" {
		endpoint = "https://agent.yourserver.com"
	}
	if timeout <= 0 {
		timeout = 180
	}

	pcfg := config.ProviderConfig{
		Type:           "openclaw",
		Endpoint:       endpoint,
		Token:          token,
		TimeoutSeconds: timeout,
	}

	if !skipTest {
		fmt.Printf("Testing connection to %s (%s)…\n", styles.Bold.Render(name), pcfg.Endpoint)
		ok, msg := pingProvider(name, pcfg)
		if ok {
			fmt.Printf("  %s Connected%s\n\n", styles.Success.Render("✓"), msg)
		} else {
			fmt.Printf("  %s %s\n", styles.Error.Render("✗"), msg)
			if interactive {
				if !promptConfirm(reader, "Save configuration anyway?", true) {
					return fmt.Errorf("configuration aborted by user")
				}
				fmt.Println()
			}
		}
	}

	return saveProviderConfig(cfg, name, pcfg, makeDef)
}

func runConfigureHermes(name, endpoint, token string, timeout int, makeDef, skipTest bool) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.ProviderConfig),
		}
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	if name == "" {
		name = "hermes-local"
	}

	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure Hermes Provider"))
		fmt.Println(styles.Muted.Render("Multi-turn agent pipeline with function calling & planning."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting := cfg.Providers[name]

		defEndpoint := "http://127.0.0.1:8080"
		if hasExisting && existing.Endpoint != "" {
			defEndpoint = existing.Endpoint
		}
		if endpoint != "" {
			defEndpoint = endpoint
		}
		endpoint = promptInput(reader, "Endpoint URL", defEndpoint)

		tokenPrompt := "API token (optional, press Enter to skip)"
		defToken := ""
		if hasExisting && existing.Token != "" {
			tokenPrompt = "API token (press Enter to keep existing token)"
			defToken = existing.Token
		}
		enteredToken := promptInput(reader, tokenPrompt, "")
		if enteredToken != "" {
			token = enteredToken
		} else if defToken != "" {
			token = defToken
		}

		defTimeout := "120"
		if hasExisting && existing.TimeoutSeconds > 0 {
			defTimeout = strconv.Itoa(existing.TimeoutSeconds)
		} else if timeout > 0 {
			defTimeout = strconv.Itoa(timeout)
		}
		tStr := promptInput(reader, "Timeout in seconds", defTimeout)
		if n, err := strconv.Atoi(tStr); err == nil && n > 0 {
			timeout = n
		}

		defIsDefault := cfg.Default.Provider == "" || cfg.Default.Provider == name
		if !makeDef {
			makeDef = promptConfirm(reader, "Set as default provider?", defIsDefault)
		}
		fmt.Println()
	}

	if endpoint == "" {
		endpoint = "http://127.0.0.1:8080"
	}
	if timeout <= 0 {
		timeout = 120
	}

	pcfg := config.ProviderConfig{
		Type:           "hermes",
		Endpoint:       endpoint,
		Token:          token,
		TimeoutSeconds: timeout,
	}

	if !skipTest {
		fmt.Printf("Testing connection to %s (%s)…\n", styles.Bold.Render(name), pcfg.Endpoint)
		ok, msg := pingProvider(name, pcfg)
		if ok {
			fmt.Printf("  %s Connected%s\n\n", styles.Success.Render("✓"), msg)
		} else {
			fmt.Printf("  %s %s\n", styles.Error.Render("✗"), msg)
			if interactive {
				if !promptConfirm(reader, "Save configuration anyway?", true) {
					return fmt.Errorf("configuration aborted by user")
				}
				fmt.Println()
			}
		}
	}

	return saveProviderConfig(cfg, name, pcfg, makeDef)
}

func runConfigureOllama(name, endpoint, model, token string, timeout int, makeDef, skipTest bool) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.ProviderConfig),
		}
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	if name == "" {
		name = "ollama-local"
	}

	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure Ollama Provider"))
		fmt.Println(styles.Muted.Render("Local or remote open-source LLM server."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting := cfg.Providers[name]

		defEndpoint := "http://127.0.0.1:11434"
		if hasExisting && existing.Endpoint != "" {
			defEndpoint = existing.Endpoint
		}
		if endpoint != "" {
			defEndpoint = endpoint
		}
		endpoint = promptInput(reader, "Endpoint URL", defEndpoint)

		defModel := "llama3.2"
		if hasExisting && existing.Model != "" {
			defModel = existing.Model
		}
		if model != "" {
			defModel = model
		}
		model = promptInput(reader, "Model name", defModel)

		tokenPrompt := "API token (optional, press Enter to skip)"
		defToken := ""
		if hasExisting && existing.Token != "" {
			tokenPrompt = "API token (press Enter to keep existing token)"
			defToken = existing.Token
		}
		enteredToken := promptInput(reader, tokenPrompt, "")
		if enteredToken != "" {
			token = enteredToken
		} else if defToken != "" {
			token = defToken
		}

		defTimeout := "120"
		if hasExisting && existing.TimeoutSeconds > 0 {
			defTimeout = strconv.Itoa(existing.TimeoutSeconds)
		} else if timeout > 0 {
			defTimeout = strconv.Itoa(timeout)
		}
		tStr := promptInput(reader, "Timeout in seconds", defTimeout)
		if n, err := strconv.Atoi(tStr); err == nil && n > 0 {
			timeout = n
		}

		defIsDefault := cfg.Default.Provider == "" || cfg.Default.Provider == name
		if !makeDef {
			makeDef = promptConfirm(reader, "Set as default provider?", defIsDefault)
		}
		fmt.Println()
	}

	if endpoint == "" {
		endpoint = "http://127.0.0.1:11434"
	}
	if model == "" {
		model = "llama3.2"
	}
	if timeout <= 0 {
		timeout = 120
	}

	pcfg := config.ProviderConfig{
		Type:           "ollama",
		Endpoint:       endpoint,
		Model:          model,
		Token:          token,
		TimeoutSeconds: timeout,
	}

	if !skipTest {
		fmt.Printf("Testing connection to %s (%s)…\n", styles.Bold.Render(name), pcfg.Endpoint)
		ok, msg := pingProvider(name, pcfg)
		if ok {
			fmt.Printf("  %s Connected%s\n\n", styles.Success.Render("✓"), msg)
		} else {
			fmt.Printf("  %s %s\n", styles.Error.Render("✗"), msg)
			if interactive {
				if !promptConfirm(reader, "Save configuration anyway?", true) {
					return fmt.Errorf("configuration aborted by user")
				}
				fmt.Println()
			}
		}
	}

	return saveProviderConfig(cfg, name, pcfg, makeDef)
}

func promptInput(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultVal
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptConfirm(reader *bufio.Reader, label string, defaultYes bool) bool {
	hint := "[Y/n]"
	if !defaultYes {
		hint = "[y/N]"
	}
	fmt.Printf("  %s %s: ", label, hint)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}

func saveProviderConfig(cfg *config.Config, name string, pcfg config.ProviderConfig, makeDefault bool) error {
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}
	cfg.Providers[name] = pcfg
	if makeDefault || cfg.Default.Provider == "" {
		cfg.Default.Provider = name
	}
	if err := config.WriteFromStruct(cfg); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("%s Provider %s (%s) saved!\n", styles.Success.Render("✓"), styles.Bold.Render(name), pcfg.Type)
	if cfg.Default.Provider == name {
		fmt.Printf("  Active default: %s\n", styles.Bold.Render(name))
	}
	fmt.Printf("  Config file:    %s\n\n", config.ConfigFile())
	return nil
}

func pingProvider(name string, pcfg config.ProviderConfig) (bool, string) {
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
	case "custom":
		prov, err = custom.New(name, pcfg)
	default:
		return false, fmt.Sprintf("unknown provider type %q", pcfg.Type)
	}
	if err != nil {
		return false, err.Error()
	}

	if err := prov.Connect(ctx); err != nil {
		return false, err.Error()
	}

	status, err := prov.Status(ctx)
	if err != nil {
		return false, err.Error()
	}

	if status.Connected {
		latency := ""
		if status.Latency > 0 {
			latency = fmt.Sprintf(" (%dms)", status.Latency.Milliseconds())
		}
		return true, latency
	}
	return false, "offline: " + status.Message
}

// testProvider pings a single provider and prints the result.
func testProvider(name string, pcfg config.ProviderConfig) {
	fmt.Printf("Testing %s (%s)…\n", name, pcfg.Endpoint)
	ok, msg := pingProvider(name, pcfg)
	if ok {
		fmt.Printf("  %s Connected%s\n", styles.Success.Render("✓"), msg)
	} else {
		fmt.Printf("  %s %s\n", styles.Error.Render("✗"), msg)
	}
	fmt.Println()
}

// ---------------------------------------------------------------------------
// Persona/Theme selectors (uses TUI selector)
// ---------------------------------------------------------------------------

type simplePickerModel struct {
	title     string
	items     []struct{ id, label, sub string }
	cursor    int
	activeID  string
	chosen    string
	action    string // "select", "edit", or "cancel"
	allowEdit bool
	width     int
	height    int
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
			m.action = "cancel"
			return m, tea.Quit
		case "e":
			if m.allowEdit && len(m.items) > 0 && m.cursor < len(m.items) {
				m.chosen = m.items[m.cursor].id
				m.action = "edit"
				return m, tea.Quit
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.items) > 0 && m.cursor < len(m.items) {
				m.chosen = m.items[m.cursor].id
				m.action = "select"
			}
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
	if m.allowEdit {
		b.WriteString(styles.Muted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [e] Edit file  ·  [Esc] Cancel"))
	} else {
		b.WriteString(styles.Muted.Render("[↑/↓] Navigate  ·  [Enter] Select  ·  [Esc] Cancel"))
	}

	content := styles.SelectorBox.Render(b.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func pickFromList(title string, items []struct{ id, label, sub string }, activeID string) string {
	chosen, _ := pickFromListWithAction(title, items, activeID, false)
	return chosen
}

func pickFromListWithAction(title string, items []struct{ id, label, sub string }, activeID string, allowEdit bool) (string, string) {
	m := simplePickerModel{title: title, items: items, activeID: activeID, allowEdit: allowEdit}
	p := tea.NewProgram(m, tea.WithAltScreen())
	result, err := p.Run()
	if err != nil {
		return "", "cancel"
	}
	res := result.(simplePickerModel)
	return res.chosen, res.action
}

func getPersonaChoices() []struct{ id, label, sub string } {
	items := []struct{ id, label, sub string }{
		{"hikari", "Hikari", "Friendly conversational assistant — warm and helpful"},
		{"developer", "Developer", "Code-focused assistant — concise, technical, precise"},
		{"sysadmin", "SysAdmin", "System admin expert — commands, logs, infrastructure"},
		{"default", "Default", "Generic helpful assistant — neutral and balanced"},
	}

	mgr, err := persona.NewManagerDefault()
	if err == nil {
		known := map[string]bool{"hikari": true, "developer": true, "sysadmin": true, "default": true}
		for _, p := range mgr.All() {
			id := strings.ToLower(p.Name)
			if !known[id] {
				items = append(items, struct{ id, label, sub string }{
					id:    id,
					label: p.Name,
					sub:   p.Description,
				})
				known[id] = true
			}
		}
	}
	return items
}

func getThemeChoices() []struct{ id, label, sub string } {
	items := []struct{ id, label, sub string }{
		{"default", "Default (Violet)", "Deep violet and indigo — the classic Hikari look"},
		{"minimal", "Minimal", "Low-contrast, distraction-free monochrome"},
		{"tokyo-night", "Tokyo Night", "Inspired by the popular VS Code Tokyo Night palette"},
	}

	tm, err := theme.NewManagerDefault()
	if err == nil {
		known := map[string]bool{"default": true, "minimal": true, "tokyo-night": true}
		for _, t := range tm.All() {
			id := strings.ToLower(t.Name)
			if !known[id] {
				sub := t.Description
				if sub == "" {
					sub = fmt.Sprintf("User theme: %s", t.Name)
				}
				items = append(items, struct{ id, label, sub string }{
					id:    id,
					label: t.Name,
					sub:   sub,
				})
				known[id] = true
			}
		}
	}
	return items
}

func runSetPersona(target ...string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	var chosen string
	if len(target) > 0 && strings.TrimSpace(target[0]) != "" {
		chosen = strings.TrimSpace(strings.ToLower(target[0]))
	} else {
		items := getPersonaChoices()
		var action string
		chosen, action = pickFromListWithAction("Select Default Persona", items, cfg.Default.Persona, true)
		if action == "edit" {
			return openPersonaEditor(chosen)
		}
		if chosen == "" || action == "cancel" {
			return nil
		}
	}

	cfg.Default.Persona = chosen
	if err := config.WriteFromStruct(cfg); err != nil {
		return err
	}
	fmt.Printf("%s Default persona set to %q\n", styles.Success.Render("✓"), chosen)
	return nil
}

func runSetTheme(target ...string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	var chosen string
	if len(target) > 0 && strings.TrimSpace(target[0]) != "" {
		chosen = strings.TrimSpace(strings.ToLower(target[0]))
	} else {
		items := getThemeChoices()
		var action string
		chosen, action = pickFromListWithAction("Select Default Theme", items, cfg.UI.Theme, true)
		if action == "edit" {
			return openThemeEditor(chosen)
		}
		if chosen == "" || action == "cancel" {
			return nil
		}
	}

	cfg.UI.Theme = chosen
	if err := config.WriteFromStruct(cfg); err != nil {
		return err
	}
	fmt.Printf("%s Theme set to %q\n", styles.Success.Render("✓"), chosen)
	return nil
}

func listPersonas() error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	items := getPersonaChoices()
	fmt.Println(styles.Bold.Render("Available Personas:"))
	fmt.Println()
	for _, it := range items {
		active := ""
		if strings.EqualFold(it.id, cfg.Default.Persona) || (cfg.Default.Persona == "" && it.id == "default") {
			active = " " + styles.Success.Render("[active]")
		}
		fmt.Printf("  • %s (%s)%s\n", styles.Bold.Render(it.label), it.id, active)
		if it.sub != "" {
			fmt.Printf("    %s\n", styles.Muted.Render(it.sub))
		}
	}
	fmt.Println()
	return nil
}

func listThemes() error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	items := getThemeChoices()
	fmt.Println(styles.Bold.Render("Available Themes:"))
	fmt.Println()
	for _, it := range items {
		active := ""
		if strings.EqualFold(it.id, cfg.UI.Theme) || (cfg.UI.Theme == "" && it.id == "default") {
			active = " " + styles.Success.Render("[active]")
		}
		fmt.Printf("  • %s (%s)%s\n", styles.Bold.Render(it.label), it.id, active)
		if it.sub != "" {
			fmt.Printf("    %s\n", styles.Muted.Render(it.sub))
		}
	}
	fmt.Println()
	return nil
}

func openPersonaEditor(name string) error {
	if err := config.EnsureDirs(); err != nil {
		return fmt.Errorf("ensuring config dirs: %w", err)
	}

	cfg, _ := config.Load()
	if name == "" && cfg != nil && cfg.Default.Persona != "" {
		name = cfg.Default.Persona
	}
	if name == "" {
		name = "hikari"
	}
	name = strings.ToLower(name)

	targetPath := filepath.Join(config.PersonasDir(), name+".yaml")
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		embedPath := "personas/" + name + ".yaml"
		var content []byte
		if data, err := assets.Personas.ReadFile(embedPath); err == nil {
			content = data
		} else {
			content = []byte(fmt.Sprintf(`name: %s
description: Custom persona
greeting: "Hello! How can I help you today?"
system_prompt: |
  You are a helpful AI assistant.
behavior:
  tone: casual
  language: English
`, name))
		}
		if err := os.WriteFile(targetPath, content, 0600); err != nil {
			return fmt.Errorf("creating persona file: %w", err)
		}
		fmt.Printf("Created user persona template at %s\n", targetPath)
	}

	return openEditor(targetPath)
}

func openThemeEditor(name string) error {
	if err := config.EnsureDirs(); err != nil {
		return fmt.Errorf("ensuring config dirs: %w", err)
	}

	cfg, _ := config.Load()
	if name == "" && cfg != nil && cfg.UI.Theme != "" {
		name = cfg.UI.Theme
	}
	if name == "" {
		name = "default"
	}
	name = strings.ToLower(name)

	targetPath := filepath.Join(config.ThemesDir(), name+".toml")
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		embedPath := "themes/" + name + ".toml"
		var content []byte
		if data, err := assets.Themes.ReadFile(embedPath); err == nil {
			content = data
		} else {
			content = []byte(fmt.Sprintf(`name = "%s"
description = "Custom Hikari theme"

[colors]
primary     = "#7C3AED"
accent      = "#A78BFA"
dim         = "#6B7280"
subtle      = "#374151"
text        = "#F9FAFB"
text_muted  = "#9CA3AF"
user        = "#34D399"
assistant   = "#A78BFA"
system      = "#FCD34D"
success     = "#10B981"
error       = "#F87171"
warning     = "#FBBF24"
border      = "#4C1D95"
background  = "#0F0F1A"
`, name))
		}
		if err := os.WriteFile(targetPath, content, 0600); err != nil {
			return fmt.Errorf("creating theme file: %w", err)
		}
		fmt.Printf("Created user theme template at %s\n", targetPath)
	}

	return openEditor(targetPath)
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

func openEditor(filePath ...string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "nano"
	}

	target := config.ConfigFile()
	if len(filePath) > 0 && filePath[0] != "" {
		target = filePath[0]
	} else if !config.Exists() {
		if err := config.WriteDefault(); err != nil {
			return err
		}
	}

	cmd := exec.Command(editor, target) //nolint:gosec // editor is from user env
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

	cmd.AddCommand(configOpenClawCmd())
	cmd.AddCommand(configHermesCmd())
	cmd.AddCommand(configOllamaCmd())
	cmd.AddCommand(configCustomCmd())

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
		Use:   "edit [target]",
		Short: "Open config file in $EDITOR (config.toml, persona, or theme)",
		Long: `Open configuration files in $EDITOR.

Targets:
  hikari config edit            Open config.toml
  hikari config edit persona    Open active persona configuration file
  hikari config edit theme      Open active theme configuration file
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return openEditor()
			}
			target := strings.ToLower(args[0])
			switch target {
			case "persona", "personas":
				return openPersonaEditor("")
			case "theme", "themes":
				return openThemeEditor("")
			case "config", "config.toml":
				return openEditor()
			default:
				return fmt.Errorf("unknown edit target %q; expected 'persona', 'theme', or leave empty for config.toml", args[0])
			}
		},
	}
}

func configPersonaCmd() *cobra.Command {
	var (
		name string
		list bool
		edit bool
	)
	cmd := &cobra.Command{
		Use:   "persona [name]",
		Short: "Set or edit the default persona (auto-updates config.toml)",
		Long: `Configure, select, or edit personas.
Can be run interactively or with command-line flags/arguments.

Examples:
  hikari config persona               # Interactive picker ([Enter] select, [e] edit)
  hikari config persona developer     # Set default persona to developer
  hikari config persona --edit        # Edit active persona in $EDITOR
  hikari config persona --list        # List available personas
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return listPersonas()
			}
			target := name
			if len(args) > 0 {
				target = args[0]
			}
			if edit {
				return openPersonaEditor(target)
			}
			return runSetPersona(target)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "Persona name to set as default")
	cmd.Flags().BoolVarP(&list, "list", "l", false, "List available personas")
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "Open persona configuration file in $EDITOR")
	return cmd
}

func configThemeCmd() *cobra.Command {
	var (
		name string
		list bool
		edit bool
	)
	cmd := &cobra.Command{
		Use:   "theme [name]",
		Short: "Set or edit the default UI theme (auto-updates config.toml)",
		Long: `Configure, select, or edit UI themes.
Can be run interactively or with command-line flags/arguments.

Examples:
  hikari config theme               # Interactive picker ([Enter] select, [e] edit)
  hikari config theme tokyo-night   # Set default theme to tokyo-night
  hikari config theme --edit        # Edit active theme in $EDITOR
  hikari config theme --list        # List available themes
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if list {
				return listThemes()
			}
			target := name
			if len(args) > 0 {
				target = args[0]
			}
			if edit {
				return openThemeEditor(target)
			}
			return runSetTheme(target)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "Theme name to set as default")
	cmd.Flags().BoolVarP(&list, "list", "l", false, "List available themes")
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "Open theme configuration file in $EDITOR")
	return cmd
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

// ---------------------------------------------------------------------------
// hikari config custom — configure a generic OpenAI-compatible provider
// ---------------------------------------------------------------------------

func configCustomCmd() *cobra.Command {
	var (
		name       string
		endpoint   string
		token      string
		model      string
		healthPath string
		timeout    int
		makeDef    bool
		skipTest   bool
	)
	cmd := &cobra.Command{
		Use:   "custom",
		Short: "Configure a custom/OpenAI-compatible provider (auto-updates config.toml)",
		Long: `Configure any OpenAI-compatible HTTP API as a Hikari provider.
Works with Groq, Together AI, LM Studio, OpenRouter, vLLM, Mistral, and more.
Can be run interactively or with command-line flags. Automatically saves to config.toml.

Examples:
  hikari config custom
  hikari config custom --name my-groq --endpoint https://api.groq.com/openai/v1 --token gsk_... --model llama-3.1-70b-versatile --default
  hikari config custom --name lm-studio --endpoint http://127.0.0.1:1234/v1 --model local-model
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigureCustom(name, endpoint, token, model, healthPath, timeout, makeDef, skipTest)
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "my-custom", "Provider name identifier in config")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "API base URL (e.g. https://api.groq.com/openai/v1)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "API key / bearer token (optional for local servers)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model ID to use (e.g. llama-3.1-70b-versatile)")
	cmd.Flags().StringVar(&healthPath, "health-path", "/models", "Path used to test connectivity")
	cmd.Flags().IntVar(&timeout, "timeout", 60, "Timeout in seconds")
	cmd.Flags().BoolVarP(&makeDef, "default", "d", false, "Set as active default provider")
	cmd.Flags().BoolVar(&skipTest, "skip-test", false, "Skip connection ping test")
	return cmd
}

func runConfigureCustom(name, endpoint, token, model, healthPath string, timeout int, makeDef, skipTest bool) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{
			Providers: make(map[string]config.ProviderConfig),
		}
	}
	if cfg.Providers == nil {
		cfg.Providers = make(map[string]config.ProviderConfig)
	}

	if name == "" {
		name = "my-custom"
	}

	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure Custom / OpenAI-Compatible Provider"))
		fmt.Println(styles.Muted.Render("Connect any OpenAI-compatible API: Groq, Together AI, LM Studio, OpenRouter, vLLM, and more."))
		fmt.Println()
		fmt.Println(styles.Muted.Render("  Compatible services:"))
		fmt.Println(styles.Muted.Render("    Groq          → https://api.groq.com/openai/v1"))
		fmt.Println(styles.Muted.Render("    Together AI   → https://api.together.xyz/v1"))
		fmt.Println(styles.Muted.Render("    OpenRouter    → https://openrouter.ai/api/v1"))
		fmt.Println(styles.Muted.Render("    LM Studio     → http://127.0.0.1:1234/v1"))
		fmt.Println(styles.Muted.Render("    vLLM          → http://your-server:8000/v1"))
		fmt.Println(styles.Muted.Render("    Mistral       → https://api.mistral.ai/v1"))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting := cfg.Providers[name]

		defEndpoint := "https://"
		if hasExisting && existing.Endpoint != "" {
			defEndpoint = existing.Endpoint
		}
		if endpoint != "" {
			defEndpoint = endpoint
		}
		endpoint = promptInput(reader, "API Base URL (include /v1 if needed)", defEndpoint)

		tokenPrompt := "API Key / Bearer Token (optional, press Enter to skip)"
		defToken := ""
		if hasExisting && existing.Token != "" {
			tokenPrompt = "API Key (press Enter to keep existing key)"
			defToken = existing.Token
		}
		enteredToken := promptInput(reader, tokenPrompt, "")
		if enteredToken != "" {
			token = enteredToken
		} else if defToken != "" {
			token = defToken
		}

		defModel := ""
		if hasExisting && existing.Model != "" {
			defModel = existing.Model
		}
		if model != "" {
			defModel = model
		}
		model = promptInput(reader, "Model ID (e.g. llama-3.1-70b-versatile)", defModel)

		defHealthPath := "/models"
		if hasExisting && existing.HealthPath != "" {
			defHealthPath = existing.HealthPath
		}
		if healthPath != "" && healthPath != "/models" {
			defHealthPath = healthPath
		}
		healthPath = promptInput(reader, "Health check path (used for ping test)", defHealthPath)

		defTimeout := "60"
		if hasExisting && existing.TimeoutSeconds > 0 {
			defTimeout = strconv.Itoa(existing.TimeoutSeconds)
		} else if timeout > 0 {
			defTimeout = strconv.Itoa(timeout)
		}
		tStr := promptInput(reader, "Timeout in seconds", defTimeout)
		if n, err := strconv.Atoi(tStr); err == nil && n > 0 {
			timeout = n
		}

		defIsDefault := cfg.Default.Provider == "" || cfg.Default.Provider == name
		if !makeDef {
			makeDef = promptConfirm(reader, "Set as default provider?", defIsDefault)
		}
		fmt.Println()
	}

	if endpoint == "" {
		endpoint = "https://"
	}
	if healthPath == "" {
		healthPath = "/models"
	}
	if timeout <= 0 {
		timeout = 60
	}

	pcfg := config.ProviderConfig{
		Type:           "custom",
		Endpoint:       endpoint,
		Token:          token,
		Model:          model,
		HealthPath:     healthPath,
		Compatibility:  "openai",
		TimeoutSeconds: timeout,
	}

	if !skipTest {
		fmt.Printf("Testing connection to %s (%s)…\n", styles.Bold.Render(name), pcfg.Endpoint)
		ok, msg := pingProvider(name, pcfg)
		if ok {
			fmt.Printf("  %s Connected%s\n\n", styles.Success.Render("✓"), msg)
		} else {
			fmt.Printf("  %s %s\n", styles.Error.Render("✗"), msg)
			if interactive {
				if !promptConfirm(reader, "Save configuration anyway?", true) {
					return fmt.Errorf("configuration aborted by user")
				}
				fmt.Println()
			}
		}
	}

	return saveProviderConfig(cfg, name, pcfg, makeDef)
}
