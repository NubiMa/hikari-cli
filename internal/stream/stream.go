// Package stream bridges provider event channels to Bubble Tea commands.
//
// Providers emit events on a <-chan provider.Event channel. Bubble Tea
// requires commands (tea.Cmd) that return tea.Msg values. This package
// provides the glue between the two.
package stream

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nubiv/hikari/internal/provider"
)

// ---------------------------------------------------------------------------
// Messages (tea.Msg types)
// ---------------------------------------------------------------------------

// TokenMsg carries a single streamed text token from the provider.
type TokenMsg struct {
	Content string
}

// DoneMsg signals that the provider has finished generating.
type DoneMsg struct{}

// ErrorMsg carries an error from the provider stream.
type ErrorMsg struct {
	Err error
}

// StatusMsg carries a provider status update (e.g. "thinking…").
type StatusMsg struct {
	Content string
}

// ---------------------------------------------------------------------------
// Commands (tea.Cmd factories)
// ---------------------------------------------------------------------------

// NextEvent returns a tea.Cmd that reads one event from the channel and
// converts it to the appropriate tea.Msg.
//
// The returned Cmd should be chained: after handling a TokenMsg, schedule
// another NextEvent() to read the next token. Stop chaining on DoneMsg or
// ErrorMsg.
//
// Usage in your Update():
//
//	case stream.TokenMsg:
//	    m.response += msg.Content
//	    return m, stream.NextEvent(m.eventChan)
//	case stream.DoneMsg:
//	    // done — don't chain
//	case stream.ErrorMsg:
//	    // handle error — don't chain
func NextEvent(ch <-chan provider.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			// Channel closed without an explicit Done — treat as done.
			return DoneMsg{}
		}
		switch evt.Type {
		case provider.EventToken:
			return TokenMsg{Content: evt.Content}
		case provider.EventDone:
			return DoneMsg{}
		case provider.EventError:
			return ErrorMsg{Err: evt.Error}
		case provider.EventStatus:
			return StatusMsg{Content: evt.Content}
		default:
			return DoneMsg{}
		}
	}
}

// DrainToString reads all events from a channel synchronously and returns
// the concatenated text. Used by one-shot CLI mode where no TUI is running.
// Returns the full response and the first error encountered (if any).
func DrainToString(ch <-chan provider.Event) (string, error) {
	var result string
	for evt := range ch {
		switch evt.Type {
		case provider.EventToken:
			result += evt.Content
		case provider.EventError:
			return result, evt.Error
		case provider.EventDone:
			return result, nil
		}
	}
	return result, nil
}
