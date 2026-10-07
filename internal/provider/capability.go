package provider

// Capabilities describes which optional features a Provider supports.
//
// The base Provider interface only requires Chat and Status. All other
// capabilities are optional. Code that uses a Provider should check the
// relevant Capabilities flag before calling methods that depend on it.
//
// Example:
//
//	caps := p.Capabilities()
//	if caps.Models {
//	    models, err := p.Models(ctx)
//	}
type Capabilities struct {
	// Chat indicates the provider supports basic conversational messages.
	// All providers must support this — it is included for completeness.
	Chat bool

	// Streaming indicates the provider can stream token-by-token responses.
	// If false, Chat() will emit a single EventToken followed by EventDone.
	Streaming bool

	// Models indicates the provider can list available models via Models().
	Models bool

	// ToolCalling indicates the provider supports structured tool/function calls
	// from the AI model (e.g. OpenAI function calling style).
	ToolCalling bool

	// Memory indicates the provider has server-side memory / long-term context.
	Memory bool

	// AgentExecution indicates the provider can autonomously execute multi-step
	// tasks (e.g. OpenClaw or Hermes with tool chains and planning).
	AgentExecution bool

	// FilesystemAccess indicates the provider's agent can read/write files on
	// the backend server.
	FilesystemAccess bool

	// CommandExecution indicates the provider's agent can run shell commands
	// on the backend server.
	CommandExecution bool
}

// OllamaCapabilities returns the capability set for Ollama providers.
// Ollama is a model provider: it supports chat, streaming, and model listing
// but does not have agent, tool, or memory capabilities.
func OllamaCapabilities() Capabilities {
	return Capabilities{
		Chat:      true,
		Streaming: true,
		Models:    true,
	}
}

// OpenClawCapabilities returns the capability set for OpenClaw providers.
// OpenClaw is a full agent provider.
func OpenClawCapabilities() Capabilities {
	return Capabilities{
		Chat:             true,
		Streaming:        true,
		Models:           false, // OpenClaw manages its own models
		ToolCalling:      true,
		Memory:           true,
		AgentExecution:   true,
		FilesystemAccess: true,
		CommandExecution: true,
	}
}

// HermesCapabilities returns the capability set for Hermes providers.
// Hermes is an agent provider similar to OpenClaw.
func HermesCapabilities() Capabilities {
	return Capabilities{
		Chat:           true,
		Streaming:      true,
		Models:         false,
		ToolCalling:    true,
		Memory:         true,
		AgentExecution: true,
	}
}

// CustomCapabilities returns the capability set for generic custom providers.
// Custom providers target any OpenAI-compatible HTTP API. We can safely assume
// chat, streaming, and model listing — but not agent, tool, or memory features
// because those depend on server-specific capabilities we cannot know statically.
func CustomCapabilities() Capabilities {
	return Capabilities{
		Chat:      true,
		Streaming: true,
		Models:    true,
	}
}
