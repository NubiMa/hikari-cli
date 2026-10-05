package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
  hikari config openclaw    Configure OpenClaw provider (auto-updates config.toml)
  hikari config hermes      Configure Hermes provider (auto-updates config.toml)
  hikari config ollama      Configure Ollama provider (auto-updates config.toml)
  hikari config provider    Manage AI providers interactively
  hikari config test        Ping all configured providers
  hikari config edit        Open config.toml in $EDITOR
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
// Provider configuration flows
// ---------------------------------------------------------------------------

func addProviderInteractive(cfg *config.Config) error {
	items := []struct{ id, label, sub string }{
		{"openclaw", "OpenClaw (Autonomous Agent)", "Tool execution & shell access on remote server"},
		{"hermes", "Hermes (Agent Pipeline)", "Multi-turn agent pipeline with tool calling & planning"},
		{"ollama", "Ollama (Local / Remote)", "Run models like llama3.2, mistral, qwen"},
	}
	chosen := pickFromList("Select Provider Type to Add", items, "")
	switch chosen {
	case "openclaw":
		return runConfigureOpenClaw("openclaw-vps", "", "", 180, false, false)
	case "hermes":
		return runConfigureHermes("hermes-local", "", "", 120, false, false)
	case "ollama":
		return runConfigureOllama("ollama-local", "", "", "", 120, false, false)
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

	existing, hasExisting := cfg.Providers[name]
	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure OpenClaw Provider"))
		fmt.Println(styles.Muted.Render("Autonomous agent backend with tool calling, shell commands & memory."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting = cfg.Providers[name]

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

	existing, hasExisting := cfg.Providers[name]
	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure Hermes Provider"))
		fmt.Println(styles.Muted.Render("Multi-turn agent pipeline with function calling & planning."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting = cfg.Providers[name]

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

	existing, hasExisting := cfg.Providers[name]
	interactive := isTerminal(os.Stdin) && endpoint == ""

	reader := bufio.NewReader(os.Stdin)
	if interactive {
		fmt.Println()
		fmt.Println(styles.Bold.Render("⚙  Configure Ollama Provider"))
		fmt.Println(styles.Muted.Render("Local or remote open-source LLM server."))
		fmt.Println()

		name = promptInput(reader, "Provider name", name)
		existing, hasExisting = cfg.Providers[name]

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
		{"hikari", "Hikari", "Friendly conversational assistant — warm and helpful"},
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

	cmd.AddCommand(configOpenClawCmd())
	cmd.AddCommand(configHermesCmd())
	cmd.AddCommand(configOllamaCmd())

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
