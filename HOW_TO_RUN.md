# How to Run & Connect Hikari

This guide provides step-by-step instructions for building, configuring, connecting, and running **Hikari** with **Ollama**, **Hermes**, and **OpenClaw**.

---

## ⚡ Quick Start (Under 2 Minutes)

### 1. Build from Source
```bash
git clone https://github.com/NubiMa/hikari-cli.git
cd hikari-cli
make build
# Or: go build -o hikari ./cmd/hikari
```

### 2. Initialize Configuration
```bash
./hikari --init
```
This generates the config file at `~/.config/hikari/config.toml` (or `%APPDATA%\hikari\config.toml` on Windows).

### 3. Configure Your Provider
Open `~/.config/hikari/config.toml` in your editor:
```bash
nano ~/.config/hikari/config.toml
```
Configure whichever provider you want to use (see below).

---

## 🔌 Connecting Providers

### 1. Ollama (Local Workstation or Remote Server)

[Ollama](https://ollama.com/) runs open-source LLMs locally with high speed, offline privacy, and zero API costs.

#### Step 1: Ensure Ollama is running and has a model
```bash
# Verify Ollama is running
curl -s http://127.0.0.1:11434/api/tags

# If not running, start it
ollama serve

# Pull a model (e.g., qwen3.5:9b or llama3.2)
ollama pull qwen3.5:9b
```

#### Step 2: Configure `config.toml`
Add or uncomment the Ollama block in `~/.config/hikari/config.toml`:
```toml
[default]
provider = "ollama-local"
persona  = "default"
model    = "qwen3.5:9b"

[providers.ollama-local]
type            = "ollama"
endpoint        = "http://127.0.0.1:11434"
model           = "qwen3.5:9b"
timeout_seconds = 120
```

> **For Remote Ollama (e.g. VPS or LAN):**
> Change `endpoint` to `https://ollama.yourdomain.com` or `http://192.168.1.100:11434`.

---

### 2. Hermes (Agent Provider)

Hermes is an agent backend capable of autonomous planning, memory, and task execution.

#### Step 1: Ensure Hermes server is running
```bash
# Verify Hermes endpoint is alive
curl -s http://127.0.0.1:8080/health
```

#### Step 2: Configure `config.toml`
```toml
[default]
provider = "hermes-local"
persona  = "developer"

[providers.hermes-local]
type            = "hermes"
endpoint        = "http://127.0.0.1:8080"
token           = ""  # Optional API key / bearer token
timeout_seconds = 180
```

> **Tip:** You can keep tokens out of plaintext config files using environment variables:
> ```bash
> export HIKARI_HERMES_LOCAL_TOKEN="your-secret-token"
> ```

---

### 3. OpenClaw (Autonomous Cloud / VPS Agent)

OpenClaw is a full autonomous agent platform with tool calling, shell command execution, filesystem access, and persistent memory.

#### Step 1: Prepare OpenClaw Endpoint and Token
You need your OpenClaw server URL and API bearer token.

#### Step 2: Configure `config.toml`
```toml
[default]
provider = "openclaw-vps"
persona  = "sysadmin"

[providers.openclaw-vps]
type            = "openclaw"
endpoint        = "https://agent.yourdomain.com"
token           = "your-secret-bearer-token"
timeout_seconds = 180
```

> **Tip:** Pass tokens securely via environment variables:
> ```bash
> export HIKARI_OPENCLAW_VPS_TOKEN="your-secret-bearer-token"
> ```

---

## 🚀 Running Hikari

Once your provider is configured, Hikari can be run in 3 distinct modes:

### Mode 1: Interactive TUI (Terminal User Interface)
Run without arguments to launch the Bubble Tea interactive dashboard:
```bash
./hikari
```

#### Interactive Controls:
- **Send message**: Type prompt and press `Enter`
- **Cancel in-flight generation**: `Ctrl+C`
- **Toggle Context Sidebar**: `Tab`
- **Clear Chat**: `Ctrl+L`
- **Scroll Chat History**: `PageUp` / `PageDown`
- **Close Modal Selectors**: `Esc`

#### In-TUI Slash Commands:
Type these directly into the message input field:
- `/provider` — Switch between configured providers (Ollama, Hermes, OpenClaw)
- `/model` — Browse and select models from the active provider
- `/persona` — Change active persona (`Nino`, `Developer`, `SysAdmin`)
- `/theme` — Switch UI theme (`default`, `minimal`, `tokyo-night`)
- `/session` or `/history` — Browse and resume past conversation sessions
- `/status` — View connection latency, active provider, and capabilities
- `/clear` — Clear the current chat view
- `/help` — List all available slash commands
- `/exit` — Quit Hikari

---

### Mode 2: CLI One-Shot Mode
Execute a prompt directly from the shell, stream the response to stdout, and exit:
```bash
./hikari "Say hello in 3 words"

./hikari "Explain the difference between a goroutine and an OS thread"
```

#### CLI Flags:
- `--provider <name>`: Override active provider for this query:
  ```bash
  ./hikari --provider ollama-local "Explain quicksort"
  ```
- `--persona <name>`: Override active persona:
  ```bash
  ./hikari --persona developer "Review this code snippet"
  ```
- `--model <name>`: Override active model:
  ```bash
  ./hikari --model qwen3.5:9b "Draft a git commit message"
  ```
- `--no-stream`: Wait for complete response before printing:
  ```bash
  ./hikari --no-stream "Give me a regex for email validation"
  ```

---

### Mode 3: Unix Pipeline / Stdin Mode
Pipe output from commands or files directly into Hikari:

```bash
# Review Git changes
git diff | ./hikari "Review these changes for bugs or performance issues"

# Troubleshoot system errors
journalctl -u nginx -n 50 | ./hikari "What is causing these errors?"

# Summarize a code or config file
cat go.mod | ./hikari "Explain the dependencies in this file"
```

---

## 🛠️ Common Errors & Troubleshooting

### 1. `Error: default provider "ollama-local" is not defined in [providers]`
- **Cause**: In `~/.config/hikari/config.toml`, `[default] provider = "ollama-local"` is set, but the `[providers.ollama-local]` section is commented out or missing.
- **Fix**: Open `~/.config/hikari/config.toml` and ensure `[providers.ollama-local]` is uncommented:
  ```toml
  [default]
  provider = "ollama-local"

  [providers.ollama-local]
  type     = "ollama"
  endpoint = "http://127.0.0.1:11434"
  model    = "qwen3.5:9b"
  ```

### 2. `Error: connecting to provider "ollama-local": ollama connect: cannot reach http://127.0.0.1:11434`
- **Cause**: Ollama is not running on your machine.
- **Fix**: Start Ollama in another terminal with:
  ```bash
  ollama serve
  ```
  Verify it is reachable with:
  ```bash
  curl http://127.0.0.1:11434/api/tags
  ```

### 3. `Error: no active provider; run hikari --provider <name> or set [default] provider in config.toml`
- **Cause**: Neither `[default] provider` is set, nor was `--provider` passed via CLI.
- **Fix**: Specify the provider name when running, or configure it in `config.toml`:
  ```bash
  ./hikari --provider ollama-local "Hello"
  ```

### 4. `Error: ollama: no model configured for provider "ollama-local"`
- **Cause**: The `model` key is empty in `[providers.ollama-local]`.
- **Fix**: Add `model = "your-model-name"` under `[providers.ollama-local]` in `config.toml`, or select it in the TUI using `/model`.

### 5. `go test -v` exits with code 1
- **Cause**: Running `go test -v` only tests the current directory (`.`), which has no Go files.
- **Fix**: Run across all subpackages using:
  ```bash
  go test -v ./...
  # or
  make test
  ```

---

## 🧪 Automated System Check

To run a full self-test of your environment, build, and configuration:
```bash
./scripts/check_system.sh
```
This runs all prerequisite checks, compiles the binary, executes the test suite, and outputs a detailed diagnostic report to `system_check.log`.
