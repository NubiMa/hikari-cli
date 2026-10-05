# Contributing to Hikari

Thank you for your interest in contributing to Hikari!

Hikari is an open-source terminal-native AI platform designed around a core philosophy:
> **Hikari is the interface, not the intelligence.**

---

## Getting Started

### Prerequisites

- [Go](https://go.dev/) 1.22 or higher
- `make`
- (Optional) [golangci-lint](https://golangci-lint.run/)

### Building and Testing

```bash
# Clone the repository
git clone https://github.com/NubiMa/hikari-cli.git
cd hikari-cli

# Build binary
make build

# Run unit tests
make test

# Run linter
make lint

# Install locally
make install
```

---

## Adding New Features

### Adding a Persona

Personas are defined in YAML. You can add default personas to `assets/personas/<name>.yaml` or put them in `~/.config/hikari/personas/`:

```yaml
name: CustomPersona
description: Brief description of assistant role
greeting: "Hello, ready to assist."
system_prompt: |
  You are an expert software engineer...
behavior:
  tone: direct
  language: English
```

### Adding a Theme

Themes are defined in TOML files in `assets/themes/<name>.toml` or `~/.config/hikari/themes/`:

```toml
name = "my-theme"
description = "My custom color theme"

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
```

### Adding a Provider

1. Implement the `provider.Provider` interface in `internal/provider/<name>/<name>.go`:
   - `Connect(ctx context.Context) error`
   - `Close() error`
   - `Chat(ctx context.Context, history []provider.Message, prompt string) (<-chan provider.Event, error)`
   - `Status(ctx context.Context) (provider.ProviderStatus, error)`
   - `Models(ctx context.Context) ([]provider.Model, error)`
   - `Capabilities() provider.Capabilities`
2. Register the constructor in `internal/app/app.go`.
3. Add unit tests with mock HTTP handlers.

---

## Submitting a Pull Request

1. Fork the repository and create a feature branch (`git checkout -b feature/my-feature`).
2. Ensure `make test` and `make build` pass cleanly.
3. Commit with descriptive messages.
4. Push to your fork and submit a PR to the `main` branch.
