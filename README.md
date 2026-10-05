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

Hikari is an open-source, terminal-native AI platform written in Go. It provides a unified TUI, CLI, and Unix-style pipe interface for interacting with multiple AI backends—including **Ollama, OpenClaw, and Hermes**—running either locally or on remote servers/VPS. Cross-platform support for **Linux**, **Windows**, and **macOS**.

Hikari acts as the shell and orchestration layer, leaving AI processing to your chosen backend.

---

## Features

- 🖥️ **Full-Featured Interactive TUI**: Built with Bubble Tea and Lip Gloss featuring real-time streaming, auto-scrolling, status indicators, and modal pickers.
- ⚡ **CLI One-Shot Mode**: Fast single-query answers via `hikari "explain this error"`.
- 🚰 **Unix Pipeline Integration**: Stream stdin directly into prompts (`cat file | hikari "summarize"` or `git diff | hikari "review"`).
- 🔌 **Provider-Agnostic Abstraction**: Connect to local Ollama instances or remote VPS backends without rewriting scripts or changing workflows.
- 🎭 **Persona System**: Decouple personality and behavior from the platform using YAML personas (comes with `Hikari`, `Developer`, and `SysAdmin`).
- 🎨 **Theming & ASCII Art**: Fully customisable TOML themes and ASCII art banners (`Default Violet`, `Minimal`, `Tokyo Night`).
- 📜 **Session Persistence & History**: Automatically saves conversations with date grouping and resume support.
- 🔒 **Security-First**: Token overrides via environment variables (`HIKARI_<PROVIDER>_TOKEN`) and credential sanitisation in terminal logs.
- 🔄 **Built-in Auto-Updater**: Update to the latest release with a single command (`hikari update`).

---

## Installation

### One-Line Install

#### Linux & macOS
```bash
curl -fsSL https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.sh | sh
```

#### Windows (PowerShell)
Open PowerShell (either as Administrator or regular user) and run:
```powershell
irm https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.ps1 | iex
```
*(Alternatively: `iwr -useb https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.ps1 | iex`)*

This downloads the latest release binary for your architecture (`x86_64` / `arm64`), installs it to `%LOCALAPPDATA%\Programs\hikari`, and adds it to your user `PATH`.

---

### Prebuilt Binaries
Download archives directly for your OS and architecture from the [GitHub Releases](https://github.com/NubiMa/hikari-cli/releases) page:
- **Linux**: `hikari_<version>_linux_amd64.tar.gz` / `arm64`
- **Windows**: `hikari_<version>_windows_amd64.zip` / `arm64`
- **macOS**: `hikari_<version>_darwin_amd64.tar.gz` / `arm64` (Apple Silicon)

---

### Build From Source
Requires [Go](https://go.dev/) 1.24+:

```bash
git clone https://github.com/NubiMa/hikari-cli.git
cd hikari-cli
make build
make install
```

---

### Keeping Hikari Updated
Hikari includes a built-in self-updater. Check for updates and automatically upgrade in place:
```bash
hikari update
```

---

## ⚡ Quick Start

### 1. First-Run Setup Wizard
The first time you launch `hikari` on a fresh system, it automatically starts the **interactive setup wizard**:
```bash
hikari
```
The wizard guides you through:
1. **Selecting an AI backend** (`Ollama`, `OpenClaw`, or `Hermes`)
2. **Configuring endpoint, model, and authentication**
3. **Live connection test** (pings your backend to ensure it's online)
4. **Selecting a default persona** (`Hikari`, `Developer`, `SysAdmin`)
5. **Selecting a UI theme** (`Default Violet`, `Minimal`, `Tokyo Night`)
6. **Writing `config.toml` automatically**

### 2. The `hikari config` CLI & Manager
Hikari features a dedicated configuration management system. You can open the full-screen interactive manager or configure providers with one-line subcommands:

```bash
# Open interactive configuration manager:
hikari config
```

```
┌───────────────────────────────────────────────┐
│              ⚙ HIKARI CONFIG                  │
│                                               │
│  › Providers            1 configured          │
│    Test Connections     ping all backends     │
│    Default Persona      hikari                │
│    Default Theme        default               │
│    Edit config.toml     open in $EDITOR       │
│    Show config path     ~/.config/...         │
│    Re-run setup wizard  guided first-run      │
│    Exit                                       │
└───────────────────────────────────────────────┘
```

#### Direct Provider Setup Subcommands (auto-updates `config.toml`):
Configure any provider interactively or via flags without touching TOML files by hand:

```bash
# Configure Ollama (local or remote):
hikari config ollama
hikari config ollama --endpoint http://127.0.0.1:11434 --model llama3.2 --default

# Configure OpenClaw (autonomous agent):
hikari config openclaw
hikari config openclaw --endpoint https://agent.yourdomain.com --token "sk-claw-..." --default

# Configure Hermes (agent pipeline):
hikari config hermes
hikari config hermes --endpoint http://127.0.0.1:8080 --default
```

#### Useful Config Commands:
| Command | Description |
|---|---|
| `hikari config` | Open interactive configuration menu |
| `hikari config provider` | Interactive provider manager (test, set default, delete, add) |
| `hikari config provider list` | List all configured providers and the active default |
| `hikari config test` | Ping all configured providers and verify connection latencies |
| `hikari config test <name>` | Ping a specific provider |
| `hikari config setup` | Re-run the interactive first-run setup wizard |
| `hikari config edit` | Open `config.toml` in your default `$EDITOR` |
| `hikari config path` | Print the exact path to `config.toml` on your system |
| `hikari --init` | Generate default directory structure and skeleton config |

Configuration file locations:
- **Linux & macOS**: `~/.config/hikari/config.toml`
- **Windows**: `%APPDATA%\hikari\config.toml` (e.g. `C:\Users\<User>\AppData\Roaming\hikari\config.toml`)

---

## 🔌 How to Connect Your AI Backend

Hikari connects to both **Model Providers** (inference/LLMs) and **Agent Providers** (autonomous workflows with tool/command execution).

For full details, see the [Provider Connection Guide (PROVIDERS.md)](PROVIDERS.md).

### 1. Connecting to Ollama (Local Workstation)

[Ollama](https://ollama.com/) runs open-source models (like Llama 3, Qwen 2.5, DeepSeek, Mistral) locally with zero API cost and complete privacy.

#### Step 1: Start Ollama and verify it is running
```bash
# Verify Ollama service is reachable
curl -s http://127.0.0.1:11434/api/tags

# If not running, start Ollama:
ollama serve

# Pull your preferred model (e.g. llama3.2, qwen2.5:7b, qwen2.5-coder:7b):
ollama pull llama3.2
```

#### Step 2: Configure Hikari
Run the setup command (auto-updates `config.toml`):
```bash
hikari config ollama
```

Or configure manually in `config.toml`:
```toml
[default]
provider = "ollama-local"
persona  = "hikari"
model    = "llama3.2"

[providers.ollama-local]
type            = "ollama"
endpoint        = "http://127.0.0.1:11434"
model           = "llama3.2"
timeout_seconds = 120
```

---

### 2. Connecting to Ollama (Remote VPS / LAN)

If your GPU server or workstation runs Ollama remotely:

1. **Configure Ollama to accept remote connections**:
   Set `OLLAMA_HOST=0.0.0.0:11434` when launching Ollama on the server.
2. **Configure in Hikari**:
   ```bash
   # Quick CLI setup:
   hikari config ollama --name ollama-vps --endpoint https://ollama.yourdomain.com --model qwen2.5-coder:7b --default
   ```
   Or configure manually in `config.toml`:
   ```toml
   [default]
   provider = "ollama-vps"

   [providers.ollama-vps]
   type            = "ollama"
   endpoint        = "https://ollama.yourdomain.com" # or http://192.168.1.100:11434
   model           = "qwen2.5-coder:7b"
   timeout_seconds = 180
   tls_skip_verify = false # Set true only if using self-signed certs
   ```

---

### 3. Connecting to OpenClaw (Autonomous Agent Provider)

[OpenClaw](https://github.com/) is an autonomous agent backend supporting tool execution, shell commands, filesystem access, and persistent memory.

#### Quick Setup via CLI (auto-updates `config.toml`):
```bash
# Interactive setup:
hikari config openclaw

# Or one-liner with flags:
hikari config openclaw --endpoint https://agent.yourdomain.com --token "sk-claw-..." --default
```

Or configure manually in `config.toml`:
```toml
[default]
provider = "openclaw-vps"
persona  = "sysadmin"

[providers.openclaw-vps]
type            = "openclaw"
endpoint        = "https://agent.yourdomain.com"
token           = ""  # Optional in file; recommended via environment variable
timeout_seconds = 180
```

#### Secure Token Injection
To keep secrets out of plaintext config files, export the environment variable:
```bash
# Linux / macOS:
export HIKARI_OPENCLAW_VPS_TOKEN="sk-claw-your-token-here"

# Windows (PowerShell):
$env:HIKARI_OPENCLAW_VPS_TOKEN="sk-claw-your-token-here"
```
Hikari automatically maps `HIKARI_<PROVIDER_NAME>_TOKEN` to the matching provider and redacts credentials from logs.

---

### 4. Connecting to Hermes (Agent Provider)

[Hermes](https://github.com/) powers multi-turn agent pipelines, tool calling, and planning.

#### Quick Setup via CLI (auto-updates `config.toml`):
```bash
# Interactive setup:
hikari config hermes

# Or one-liner with flags:
hikari config hermes --endpoint http://127.0.0.1:8080 --default
```

Or configure manually:
1. **Verify Hermes is reachable**:
   ```bash
   curl -s http://127.0.0.1:8080/health
   ```
2. **Configure in `config.toml`**:
   ```toml
   [default]
   provider = "hermes-local"
   persona  = "developer"

   [providers.hermes-local]
   type            = "hermes"
   endpoint        = "http://127.0.0.1:8080"
   token           = ""
   timeout_seconds = 120
   ```
   Or export `HIKARI_HERMES_LOCAL_TOKEN="your-token"`.

---

## 🚀 Running Hikari

Hikari offers 3 distinct execution modes:

### Mode 1: Interactive TUI Dashboard
Launch the Bubble Tea interactive dashboard:
```bash
hikari
```

- Type your message and hit `Enter` to send.
- Use `Ctrl+C` to cancel an in-flight response.
- Use `Tab` to toggle sidebar panels.
- Scroll history with `PageUp` / `PageDown`.
- Type `/help` to see all slash commands.

### Mode 2: CLI One-Shot Command
Execute a prompt directly from your shell, stream the response to stdout, and exit:
```bash
# Basic query
hikari "Explain the difference between a goroutine and an OS thread"

# Override provider or model on the fly
hikari --provider ollama-local --model qwen2.5-coder:7b "Write a Go HTTP handler"

# Override active persona
hikari --persona sysadmin "How do I check open ports on Linux?"

# Disable streaming (wait for complete response)
hikari --no-stream "Generate a regex for RFC 5322 email validation"
```

### Mode 3: Unix Pipeline Integration
Pipe outputs from commands, logs, or files directly into Hikari:
```bash
# Code review
git diff | hikari "Review these changes for potential bugs or security issues"

# Log troubleshooting
journalctl -u nginx -n 50 | hikari "Diagnose the root cause of these errors"

# File summarization
cat go.mod | hikari "Explain the purpose of the primary dependencies here"
```

---

## ⌨️ TUI Keyboard Shortcuts & Slash Commands

### In-Chat Slash Commands
Type commands directly into the prompt bar starting with `/`:

| Command | Action |
|---|---|
| `/help` | Display list of internal commands |
| `/provider` | Open interactive provider selector modal |
| `/persona` | Open interactive persona selector modal (`hikari`, `developer`, `sysadmin`) |
| `/model` | List and select available models from the active provider |
| `/theme` | Switch color theme (`default`, `minimal`, `tokyo-night`) |
| `/session` / `/history` | Browse past sessions grouped by date and resume conversation |
| `/status` | View provider connection latency, endpoint, and capabilities |
| `/clear` | Clear chat screen and restart conversation |
| `/exit` | Exit Hikari |

### Keybindings
- `Enter`: Send message
- `Ctrl+C`: Cancel streaming response or exit Hikari
- `Ctrl+L`: Clear conversation view
- `PageUp` / `PageDown`: Scroll through message history
- `Esc` / `Ctrl+C`: Close open modal selector

---

## 🛠️ Common Errors & Troubleshooting

### 1. `Error: default provider "ollama-local" is not defined in [providers]`
- **Cause**: In `config.toml`, `[default] provider = "ollama-local"` is set, but the `[providers.ollama-local]` block is missing or commented out.
- **Fix**: Open `config.toml` and ensure `[providers.ollama-local]` is defined with `type = "ollama"` and `endpoint = "http://127.0.0.1:11434"`.

### 2. `Error: connecting to provider: ollama connect: cannot reach http://127.0.0.1:11434`
- **Cause**: Ollama is not running on your machine.
- **Fix**: Start Ollama with `ollama serve`. Verify it is responding with `curl http://127.0.0.1:11434/api/tags`.

### 3. `Error: ollama: no model configured for provider "ollama-local"`
- **Cause**: The `model` key is empty under `[providers.ollama-local]`.
- **Fix**: Add `model = "llama3.2"` (or any installed model) to the provider section, or select one inside the TUI with `/model`.

### 4. `Error: no active provider; run hikari --provider <name> or set [default] provider in config.toml`
- **Cause**: Neither `[default] provider` is set in config, nor was `--provider` passed on the CLI.
- **Fix**: Set `provider = "ollama-local"` under `[default]` in `config.toml`, or specify `hikari --provider <name>`.

---

## 🎨 Customization

### Personas (`~/.config/hikari/personas/`)
Create custom YAML files in your personas directory:

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
Create custom TOML files in your themes directory:

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
        (OpenClaw, Hermes)            (Ollama)
               │                         │
          Local / VPS               Local / VPS
```

---

## License

Hikari is released under the [MIT License](LICENSE).
