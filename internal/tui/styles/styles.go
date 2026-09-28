// Package styles defines all Lip Gloss styles used by the Hikari TUI.
//
// Follows the tui-design skill guidelines:
// - Semantic color slots (Primary, Accent, Text, Muted, User, Assistant, System, Success, Error, Warning)
// - Visible focus indicators and structured visual hierarchy
// - Clean rounded box-drawing characters (╭ ╮ ╯ ╰ ─ │)
// - Accessible contrast and monochrome/NO_COLOR resilience
package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nubiv/hikari/internal/theme"
)

// Palette — base color tokens.
var (
	ColorPrimary    = lipgloss.Color("#7C3AED") // violet-600
	ColorAccent     = lipgloss.Color("#A78BFA") // violet-400
	ColorDim        = lipgloss.Color("#6B7280") // gray-500
	ColorSubtle     = lipgloss.Color("#1E1E2E") // dark slate surface
	ColorText       = lipgloss.Color("#F9FAFB") // crisp white
	ColorTextMuted  = lipgloss.Color("#9CA3AF") // light gray
	ColorUser       = lipgloss.Color("#34D399") // emerald-400
	ColorAssistant  = lipgloss.Color("#818CF8") // indigo-400
	ColorSystem     = lipgloss.Color("#FCD34D") // amber-300
	ColorSuccess    = lipgloss.Color("#10B981") // emerald-500
	ColorError      = lipgloss.Color("#F87171") // red-400
	ColorWarning    = lipgloss.Color("#FBBF24") // amber-400
	ColorBorder     = lipgloss.Color("#4C1D95") // violet-900 / dark border
	ColorBackground = lipgloss.Color("#0D0E15") // near-black depth
	ColorCardBg     = lipgloss.Color("#141522") // elevated card background
)

// ---------------------------------------------------------------------------
// Global Styles
// ---------------------------------------------------------------------------

var (
	// Layout
	App       lipgloss.Style
	Header    lipgloss.Style
	StatusBar lipgloss.Style
	ChatArea  lipgloss.Style

	// Header elements
	HeaderLogo       lipgloss.Style
	HeaderPill       lipgloss.Style
	HeaderPillActive lipgloss.Style
	HeaderPillDim    lipgloss.Style
	HeaderTag        lipgloss.Style
	Divider          lipgloss.Style

	// Messages & Cards
	UserHeader       lipgloss.Style
	UserCard         lipgloss.Style
	UserMessage      lipgloss.Style
	AssistantHeader  lipgloss.Style
	AssistantCard    lipgloss.Style
	AssistantMessage lipgloss.Style
	SystemHeader     lipgloss.Style
	SystemCard       lipgloss.Style
	StreamingCursor  lipgloss.Style
	CodeBlock        lipgloss.Style

	// Input Area
	InputBorder       lipgloss.Style
	InputFocused      lipgloss.Style
	InputPrompt       lipgloss.Style
	CommandPopupBox   lipgloss.Style
	CommandPopupItem  lipgloss.Style
	CommandPopupMatch lipgloss.Style

	// Sidebar / Context
	SidebarBox        lipgloss.Style
	SidebarHeader     lipgloss.Style
	SidebarSection    lipgloss.Style
	SidebarKey        lipgloss.Style
	SidebarVal        lipgloss.Style
	SidebarBadge      lipgloss.Style

	// Indicators & Status
	ProviderConnected    lipgloss.Style
	ProviderDisconnected lipgloss.Style
	PersonaLabel         lipgloss.Style

	// Selector Modal
	SelectorBox          lipgloss.Style
	SelectorTitle        lipgloss.Style
	SelectorItem         lipgloss.Style
	SelectorItemSelected lipgloss.Style
	SelectorItemActive   lipgloss.Style
	SelectorDesc         lipgloss.Style

	// Splash / Welcome View
	SplashTitle lipgloss.Style
	SplashSub   lipgloss.Style
	SplashCard  lipgloss.Style
	SplashKey   lipgloss.Style

	// Typography & Utilities
	Error   lipgloss.Style
	Warning lipgloss.Style
	Success lipgloss.Style
	Muted   lipgloss.Style
	Bold    lipgloss.Style
)

func init() {
	ApplyTheme(theme.DefaultTheme())
}

// ApplyTheme rebuilds all styles using the colors from the provided theme.
func ApplyTheme(t theme.Theme) {
	c := t.Colors

	if c.Primary != "" {
		ColorPrimary = lipgloss.Color(c.Primary)
	}
	if c.Accent != "" {
		ColorAccent = lipgloss.Color(c.Accent)
	}
	if c.Dim != "" {
		ColorDim = lipgloss.Color(c.Dim)
	}
	if c.Subtle != "" {
		ColorSubtle = lipgloss.Color(c.Subtle)
	}
	if c.Text != "" {
		ColorText = lipgloss.Color(c.Text)
	}
	if c.TextMuted != "" {
		ColorTextMuted = lipgloss.Color(c.TextMuted)
	}
	if c.User != "" {
		ColorUser = lipgloss.Color(c.User)
	}
	if c.Assistant != "" {
		ColorAssistant = lipgloss.Color(c.Assistant)
	}
	if c.System != "" {
		ColorSystem = lipgloss.Color(c.System)
	}
	if c.Success != "" {
		ColorSuccess = lipgloss.Color(c.Success)
	}
	if c.Error != "" {
		ColorError = lipgloss.Color(c.Error)
	}
	if c.Warning != "" {
		ColorWarning = lipgloss.Color(c.Warning)
	}
	if c.Border != "" {
		ColorBorder = lipgloss.Color(c.Border)
	}
	if c.Background != "" {
		ColorBackground = lipgloss.Color(c.Background)
	}

	App = lipgloss.NewStyle().
		Background(ColorBackground)

	// Modern Header Bar
	Header = lipgloss.NewStyle().
		Background(ColorSubtle).
		Foreground(ColorText).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	HeaderLogo = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorBackground).
		Background(ColorPrimary).
		Padding(0, 1)

	HeaderPill = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Background(ColorBackground).
		Padding(0, 1)

	HeaderPillActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess).
		Background(ColorBackground).
		Padding(0, 1)

	HeaderPillDim = lipgloss.NewStyle().
		Foreground(ColorDim).
		Background(ColorBackground).
		Padding(0, 1)

	HeaderTag = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent)

	Divider = lipgloss.NewStyle().
		Foreground(ColorDim)

	// Status Bar
	StatusBar = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Background(ColorSubtle).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	// Chat Area
	ChatArea = lipgloss.NewStyle().
		Padding(0, 1)

	// Message cards with rounded borders and colored accent bars
	UserHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorUser)

	UserCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorUser).
		Padding(0, 1).
		MarginBottom(1)

	UserMessage = lipgloss.NewStyle().
		Foreground(ColorText)

	AssistantHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAssistant)

	AssistantCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		MarginBottom(1)

	AssistantMessage = lipgloss.NewStyle().
		Foreground(ColorText)

	SystemHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSystem)

	SystemCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorWarning).
		Padding(0, 1).
		MarginBottom(1)

	StreamingCursor = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	CodeBlock = lipgloss.NewStyle().
		Background(ColorSubtle).
		Foreground(ColorAccent).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	// Input Box
	InputBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorDim).
		Padding(0, 1)

	InputFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1)

	InputPrompt = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	// Command Suggestion Popup
	CommandPopupBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorSubtle).
		Padding(0, 1)

	CommandPopupItem = lipgloss.NewStyle().
		Foreground(ColorTextMuted)

	CommandPopupMatch = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent)

	// Sidebar / Context
	SidebarBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(ColorBorder).
		Padding(0, 1)

	SidebarHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		MarginBottom(1)

	SidebarSection = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorDim).
		MarginTop(1)

	SidebarKey = lipgloss.NewStyle().
		Foreground(ColorTextMuted)

	SidebarVal = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)

	SidebarBadge = lipgloss.NewStyle().
		Foreground(ColorBackground).
		Background(ColorPrimary).
		Padding(0, 1)

	// Indicators
	ProviderConnected = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	ProviderDisconnected = lipgloss.NewStyle().
		Foreground(ColorError)

	PersonaLabel = lipgloss.NewStyle().
		Foreground(ColorAccent).
		Bold(true)

	// Selector Modal Dialog
	SelectorBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Background(ColorBackground).
		Padding(1, 2)

	SelectorTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1).
		MarginBottom(1)

	SelectorItem = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Padding(0, 1)

	SelectorItemSelected = lipgloss.NewStyle().
		Foreground(ColorText).
		Background(ColorPrimary).
		Bold(true).
		Padding(0, 1)

	SelectorItemActive = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	SelectorDesc = lipgloss.NewStyle().
		Foreground(ColorDim)

	// Splash / Welcome
	SplashTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

	SplashSub = lipgloss.NewStyle().
		Foreground(ColorTextMuted).
		Italic(true)

	SplashCard = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		MarginTop(1)

	SplashKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent)

	// Utilities
	Error = lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true)

	Warning = lipgloss.NewStyle().
		Foreground(ColorWarning)

	Success = lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Bold(true)

	Muted = lipgloss.NewStyle().
		Foreground(ColorTextMuted)

	Bold = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorText)
}
