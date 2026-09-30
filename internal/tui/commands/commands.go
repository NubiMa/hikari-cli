// Package commands handles internal TUI commands (those starting with /).
//
// Commands are intercepted by the input component before being sent to the
// provider. Each command returns a tea.Cmd that performs the appropriate
// action (switching provider, clearing chat, etc.).
package commands

// KnownCommands lists all supported internal commands for /help display.
var KnownCommands = []CommandDef{
	{Name: "help", Description: "Show available commands"},
	{Name: "clear", Description: "Clear conversation and start fresh"},
	{Name: "home", Description: "Exit current session and return to start screen  (alias: /new)"},
	{Name: "exit", Description: "Exit Hikari"},
	{Name: "status", Description: "Show provider connection status (with model list for Ollama)"},
	{Name: "log", Description: "Show the application log file path and recent entries"},
	{Name: "provider", Description: "Switch active provider"},
	{Name: "model", Description: "Switch active model"},
	{Name: "persona", Description: "Switch active persona"},
	{Name: "theme", Description: "Switch color theme"},
	{Name: "session", Description: "Manage sessions"},
	{Name: "history", Description: "Browse conversation history"},
}

// CommandDef describes a single internal command.
type CommandDef struct {
	Name        string
	Description string
}

// HelpText returns a formatted list of commands for /help.
func HelpText() string {
	out := "Available commands:\n\n"
	for _, c := range KnownCommands {
		out += "  /" + c.Name
		// Pad to align descriptions
		pad := 12 - len(c.Name)
		for i := 0; i < pad; i++ {
			out += " "
		}
		out += c.Description + "\n"
	}
	return out
}
