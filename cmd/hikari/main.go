package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/app"
	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/stream"
	"github.com/NubiMa/hikari-cli/internal/tui"
	cbterm "github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
)

// Build-time variables injected by GoReleaser / ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var (
		providerFlag string
		personaFlag  string
		modelFlag    string
		noStream     bool
		versionFlag  bool
		initFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "hikari [message]",
		Short: "Hikari — Terminal AI Platform",
		Long: `Hikari is a terminal-native AI platform.

Usage:
  hikari                    Open interactive TUI
  hikari "your prompt"      One-shot: print response and exit
  cat file | hikari "..."   Pipe/stdin mode`,

		SilenceUsage:  true,
		SilenceErrors: true,

		RunE: func(cmd *cobra.Command, args []string) error {
			if versionFlag {
				fmt.Printf("hikari %s (commit %s, built %s)\n", version, commit, date)
				return nil
			}

			if initFlag {
				return runInit()
			}

			// Bootstrap application
			a, err := buildApp(providerFlag, personaFlag, modelFlag)
			if err != nil {
				return err
			}

			// Determine run mode
			stdinIsTTY := isTerminal(os.Stdin)
			hasArgs := len(args) > 0

			switch {
			case !stdinIsTTY:
				// Pipe / stdin mode
				return runPipe(a, args, noStream)
			case hasArgs:
				// One-shot mode
				return runOneShot(a, strings.Join(args, " "), noStream)
			default:
				// Interactive TUI
				return tui.Start(a)
			}
		},
	}

	cmd.Flags().StringVar(&providerFlag, "provider", "", "Override active provider (must match a name in config.toml)")
	cmd.Flags().StringVar(&personaFlag, "persona", "", "Override active persona")
	cmd.Flags().StringVar(&modelFlag, "model", "", "Override active model")
	cmd.Flags().BoolVar(&noStream, "no-stream", false, "Disable streaming output (wait for full response)")
	cmd.Flags().BoolVarP(&versionFlag, "version", "v", false, "Print version and exit")
	cmd.Flags().BoolVar(&initFlag, "init", false, "Create default config file and exit")

	cmd.AddCommand(updateCmd())

	return cmd
}

// ---------------------------------------------------------------------------
// App bootstrap
// ---------------------------------------------------------------------------

func buildApp(providerName, personaName, modelName string) (*app.App, error) {
	a, err := app.New()
	if err != nil {
		return nil, fmt.Errorf("hikari: %w", err)
	}

	// CLI flag overrides
	if providerName != "" {
		if err := a.Router.SetActive(providerName); err != nil {
			return nil, err
		}
	}
	if personaName != "" {
		if err := a.Personas.SetActive(personaName); err != nil {
			return nil, err
		}
	}
	// Model override: update the active provider config in-memory
	// Full in-config model override is post-MVP
	_ = modelName

	return a, nil
}

// ---------------------------------------------------------------------------
// One-shot mode
// ---------------------------------------------------------------------------

func runOneShot(a *app.App, message string, noStream bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := a.ConnectActive(ctx); err != nil {
		return fmt.Errorf("connecting to provider: %w", err)
	}

	p, _, err := a.Router.Active()
	if err != nil {
		return err
	}

	persona := a.Personas.Active()
	history := []provider.Message{}
	if persona.SystemPrompt != "" {
		history = append(history, provider.Message{
			Role:    provider.RoleSystem,
			Content: persona.SystemPrompt,
		})
	}

	ch, err := p.Chat(ctx, history, message)
	if err != nil {
		return fmt.Errorf("chat: %w", err)
	}

	if noStream || !isTerminal(os.Stdout) {
		// Non-streaming: collect then print
		response, err := stream.DrainToString(ch)
		if err != nil {
			return err
		}
		fmt.Print(response)
		if !strings.HasSuffix(response, "\n") {
			fmt.Println()
		}
		return nil
	}

	// Streaming: print tokens as they arrive
	for evt := range ch {
		switch evt.Type {
		case provider.EventToken:
			fmt.Print(evt.Content)
		case provider.EventDone:
			fmt.Println()
			return nil
		case provider.EventError:
			fmt.Println()
			return evt.Error
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Pipe / stdin mode
// ---------------------------------------------------------------------------

func runPipe(a *app.App, args []string, noStream bool) error {
	// Read all stdin
	stdinBytes, err := io.ReadAll(bufio.NewReader(os.Stdin))
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	stdinContent := strings.TrimSpace(string(stdinBytes))

	// Combine: "<user prompt>\n\n<stdin content>" or just stdin if no args
	var message string
	if len(args) > 0 {
		message = strings.Join(args, " ") + "\n\n" + stdinContent
	} else {
		message = stdinContent
	}

	if message == "" {
		return fmt.Errorf("no input provided")
	}

	return runOneShot(a, message, noStream)
}

// ---------------------------------------------------------------------------
// Init mode
// ---------------------------------------------------------------------------

func runInit() error {
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	if err := config.WriteDefault(); err != nil {
		return err
	}
	cfgPath := config.ConfigFile()
	fmt.Printf("✓ Hikari config initialised at %s\n", cfgPath)
	fmt.Printf("  Edit the file to add your providers, then run: hikari\n")
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isTerminal reports whether the given file is an interactive terminal.
// Uses charmbracelet/x/term which is already a transitive dependency and
// works correctly on Linux, macOS, and Windows (including ConPTY).
func isTerminal(f *os.File) bool {
	return cbterm.IsTerminal(f.Fd())
}
