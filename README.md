# Hikari

```text
    ██╗  ██╗██╗██╗  ██╗ █████╗ ██████╗ ██╗
    ██║  ██║██║██║ ██╔╝██╔══██╗██╔══██╗██║
    ███████║██║█████═╝ ███████║██████╔╝██║
    ██╔══██║██║██╔═██╗ ██╔══██║██╔══██╗██║
    ██║  ██║██║██║  ██╗██║  ██║██║  ██║██║
    ╚═╝  ╚═╝╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝
```

> **One terminal platform, unlimited AI personalities.**

Hikari is an open-source, terminal-native AI platform written in Go. It provides a unified TUI, CLI, and Unix-style pipe interface for interacting with multiple AI backends—including **Ollama, OpenClaw, and Hermes**—running either locally or on remote servers/VPS.

Hikari acts as the shell and orchestration layer, leaving AI processing to your chosen backend.

---

## Features

- 🖥️ **Full-Featured Interactive TUI**: Built with Bubble Tea and Lip Gloss featuring real-time streaming, auto-scrolling, status indicators, and modal pickers.
- ⚡ **CLI One-Shot Mode**: Fast single-query answers via `hikari "explain this error"`.
- 🚰 **Unix Pipeline Integration**: Stream stdin directly into prompts (`cat file | hikari "summarize"` or `git diff | hikari "review"`).
- 🔌 **Provider-Agnostic Abstraction**: Connect to local Ollama instances or remote VPS backends without rewriting scripts or changing workflows.
- 🎭 **Persona System**: Decouple personality and behavior from the platform using YAML personas (comes with `Nino`, `Developer`, and `SysAdmin`).
- 🎨 **Theming & ASCII Art**: Fully customisable TOML themes and ASCII art banners (`Default Violet`, `Minimal`, `Tokyo Night`).
- 📜 **Session Persistence & History**: Automatically saves conversations with date grouping and resume support.
- 🔒 **Security-First**: Token overrides via environment variables (`HIKARI_<PROVIDER>_TOKEN`) and credential sanitisation in terminal logs.

---

## Installation

### One-Line Install (Linux / macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/nubiv/hikari/main/scripts/install.sh | sh
```

### From Source

Requires [Go](https://go.dev/) 1.22+:

```bash
git clone https://github.com/nubiv/hikari.git
cd hikari
make build
make install
```

---

## Quick Start

### 1. Initialize Configuration

Generate default directories and a starter configuration:

```bash
hikari --init
```

This creates `~/.config/hikari/config.toml` (or `%APPDATA%\hikari\config.toml` on Windows).

### 2. Configure Providers

Edit `~/.config/hikari/config.toml`:

```toml
[default]
provider = "ollama-local"
persona  = "nino"

[providers.ollama-local]
type     = "ollama"
endpoint = "http://127.0.0.1:11434"
model    = "llama3.2"

[providers.openclaw-vps]
type     = "openclaw"
endpoint = "https://agent.example.com"
token    = "" # Or export HIKARI_OPENCLAW_VPS_TOKEN

[ui]
theme    = "default"
ascii    = "default"
```

### 3. Launch Hikari

#### Interactive TUI
```bash
hikari
```

#### One-Shot Command
```bash
hikari "Explain the difference between a goroutine and an OS thread"
```

#### Unix Pipeline
```bash
git diff | hikari "Review these changes for bugs and performance issues"
cat /var/log/nginx/error.log | hikari "Diagnose the root cause of these 502 errors"
```

---

## TUI Keyboard Shortcuts & Commands

### In-Chat Commands

Type commands directly in the prompt starting with `/`:

| Command | Action |
|---|---|
| `/help` | Display list of internal commands |
| `/provider` | Open interactive provider selector modal |
| `/persona` | Open interactive persona selector modal |
| `/model` | List and select available models for active provider |
| `/theme` | Switch color theme (`default`, `minimal`, `tokyo-night`) |
| `/session` / `/history` | Browse past sessions grouped by date and resume |
| `/status` | View provider connection latency and capability flags |
| `/clear` | Clear chat screen and restart conversation |
| `/exit` | Exit Hikari |

### Keybindings

- `Enter`: Send message
- `Ctrl+C`: Cancel streaming response or exit Hikari
- `Ctrl+L`: Clear conversation view
- `PageUp` / `PageDown`: Scroll through message history
- `Esc` / `Ctrl+C`: Close modal selector

---

## Customization

### Personas (`~/.config/hikari/personas/`)

Create custom YAML files in `~/.config/hikari/personas/`:

```yaml
name: DevOps
description: Kubernetes and Cloud Infrastructure specialist
greeting: "DevOps assistant online. What are we deploying?"
system_prompt: |
  You are an expert DevOps engineer specializing in Kubernetes, Terraform, and CI/CD pipelines.
behavior:
  tone: concise
  language: English
```

### Themes (`~/.config/hikari/themes/`)

Create custom TOML files in `~/.config/hikari/themes/`:

```toml
name = "cyberpunk"
description = "Neon Cyberpunk theme"

[colors]
primary     = "#FF007F"
accent      = "#00F0FF"
dim         = "#555555"
subtle      = "#222222"
text        = "#FFFFFF"
text_muted  = "#888888"
user        = "#00FF66"
assistant   = "#00F0FF"
system      = "#FFE600"
success     = "#00FF66"
error       = "#FF0033"
warning     = "#FFB800"
border      = "#FF007F"
background  = "#0A0A10"
```

---

## Architecture

```text
                         HIKARI
                            │
                   ┌────────┴────────┐
                   │                 │
                  TUI               CLI
             (Bubble Tea)         (Cobra)
                   │                 │
                   └────────┬────────┘
                            │
                       Hikari Core
                   (App Orchestration)
                            │
                      Provider Router
                            │
               ┌────────────┴────────────┐
               │                         │
         Agent Providers           Model Providers
         (OpenClaw, Hermes)           (Ollama)
               │                         │
          Local / VPS               Local / VPS
```

---

## License

Hikari is released under the [MIT License](LICENSE).
