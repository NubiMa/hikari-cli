# How to Run & Test Hikari

This guide explains how to build, test, configure, and run Hikari.

> **One-line install** (after a release is published):
> ```bash
> curl -fsSL https://raw.githubusercontent.com/NubiMa/hikari-cli/main/scripts/install.sh | sh
> ```

---

## ⚠️ Important Note on Testing

If you run:
```bash
go test -v
```
Go will attempt to run tests **only in the current directory** (`.`), which contains no Go source files, exiting with code 1.

To run tests across all subpackages recursively, always use:
```bash
go test -v ./...
# or via Makefile:
make test
```

---

## 1. Prerequisites

- **Go 1.22+** installed (`go version`)
- **Make** (optional, but recommended)

---

## 2. Build

Compile the `hikari` binary:

```bash
# Using Make (injects version and build metadata)
make build

# Or using plain Go
go build -o hikari ./cmd/hikari
```

This creates an executable `./hikari` in the repository root.

---

## 3. Verify System & Generate Log

To run an automated verification check of all prerequisites, tests, builds, and configuration generation:

```bash
./scripts/check_system.sh
```

This will run all checks and save a complete report to `system_check.log`.

---

## 4. Initial Setup & Configuration

Bootstrap your default directories and configuration file:

```bash
./hikari --init
```

This creates:
- Configuration: `~/.config/hikari/config.toml` (or `%APPDATA%\hikari\config.toml` on Windows)
- Personas directory: `~/.config/hikari/personas/`
- Themes directory: `~/.config/hikari/themes/`
- ASCII art directory: `~/.config/hikari/ascii/`
- Sessions directory: `~/.config/hikari/sessions/`

### Example `~/.config/hikari/config.toml`

```toml
[default]
provider = "ollama-local"
persona  = "nino"
model    = "llama3.2"

[providers.ollama-local]
type     = "ollama"
endpoint = "http://127.0.0.1:11434"
model    = "llama3.2"

[providers.openclaw-vps]
type     = "openclaw"
endpoint = "https://agent.example.com"
token    = "" # Or export HIKARI_OPENCLAW_VPS_TOKEN

[ui]
theme = "default"
ascii = "default"
```

> **Tip:** Sensitive tokens can be exported in your environment instead of stored in plaintext:
> ```bash
> export HIKARI_OPENCLAW_VPS_TOKEN="your-secret-token"
> ```

---

## 5. Running Modes

Hikari supports three distinct operational modes:

### Mode A: Interactive TUI

Launch full-screen terminal interface:

```bash
./hikari
```

#### In-TUI Controls:
- **Send message**: `Enter`
- **Cancel streaming**: `Ctrl+C`
- **Clear chat**: `Ctrl+L`
- **Scroll history**: `PageUp` / `PageDown`
- **Close popup picker**: `Esc`

#### In-TUI Slash Commands:
Type these directly into the chat prompt:
- `/help` — List available commands
- `/provider` — Open interactive provider selector
- `/persona` — Open interactive persona selector (`Nino`, `Developer`, `SysAdmin`)
- `/theme` — Switch color themes (`default`, `minimal`, `tokyo-night`)
- `/model` — Browse and select models from the active provider
- `/session` or `/history` — Browse and resume past sessions
- `/status` — View connection status and provider capabilities
- `/clear` — Clear the current screen and start fresh
- `/exit` — Quit Hikari

---

### Mode B: CLI One-Shot Mode

Send a single prompt, receive the streamed response, and exit:

```bash
./hikari "Explain the difference between a mutex and a channel in Go"
```

Flags available:
```bash
./hikari --provider ollama-local --persona developer "Review this function"
./hikari --no-stream "Give me a quick regex for email validation"
```

---

### Mode C: Unix Pipeline / Stdin Mode

Pipe command output or files directly into Hikari:

```bash
# Review Git diffs
git diff | ./hikari "Review these changes for bugs"

# Troubleshoot system logs
journalctl -u nginx -n 50 | ./hikari "What is causing these errors?"

# Summarize a file
cat /etc/hosts | ./hikari "Summarize this configuration"
```

---

## 6. Testing

Run all unit tests:
```bash
make test
# or
go test -v ./...
```

Run race condition checks:
```bash
go test -race ./...
```
