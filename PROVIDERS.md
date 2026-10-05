# Hikari Provider Connection Guide

This guide provides step-by-step instructions for connecting Hikari to supported AI backends: **Ollama**, **OpenClaw**, and **Hermes**, whether running locally on your workstation or remotely on a VPS/cloud server.

---

## Table of Contents

1. [Architecture & Provider Categories](#1-architecture--provider-categories)
2. [Quick Reference Configuration](#2-quick-reference-configuration)
3. [Connecting to Ollama (Model Provider)](#3-connecting-to-ollama-model-provider)
   - [Local Workstation Setup](#31-local-workstation-setup)
   - [Remote VPS / Network Setup](#32-remote-vps--network-setup)
   - [Configuration Options](#33-configuration-options)
   - [Verification](#34-verification)
4. [Connecting to OpenClaw (Agent Provider)](#4-connecting-to-openclaw-agent-provider)
   - [Overview & Capabilities](#41-overview--capabilities)
   - [Configuration Options](#42-configuration-options)
   - [Security & Token Management](#43-security--token-management)
   - [Verification](#44-verification)
5. [Connecting to Hermes (Agent Provider)](#5-connecting-to-hermes-agent-provider)
   - [Overview & Capabilities](#51-overview--capabilities)
   - [Configuration Options](#52-configuration-options)
   - [Verification](#53-verification)
6. [Switching Providers in Hikari](#6-switching-providers-in-hikari)
7. [Provider Capabilities Matrix](#7-provider-capabilities-matrix)
8. [Troubleshooting & Common Issues](#8-troubleshooting--common-issues)

---

## 1. Architecture & Provider Categories

Hikari decouples the terminal user interface from AI inference and execution engines. Backends fall into two distinct categories:

| Category | Role | Backends | Key Capabilities |
|---|---|---|---|
| **Model Providers** | Pure LLM inference & chat completion | **Ollama** | Streaming, model listing, low latency, offline privacy |
| **Agent Providers** | Autonomous agents with execution capabilities | **OpenClaw**, **Hermes** | Streaming, tool calling, filesystem access, command execution, persistent memory |

---

## 2. Quick Reference Configuration

Configuration is stored at:
- **Linux/macOS**: `~/.config/hikari/config.toml`
- **Windows**: `%APPDATA%\hikari\config.toml`

Run `hikari --init` to generate the default directory structure and skeleton config.

```toml
[default]
provider = "ollama-local"
persona  = "hikari"
model    = "llama3.2"

# ── 1. Local Ollama ──────────────────────────────────────────────────────────
[providers.ollama-local]
type            = "ollama"
endpoint        = "http://127.0.0.1:11434"
model           = "llama3.2"
timeout_seconds = 120

# ── 2. Remote Ollama on VPS / Home Server ───────────────────────────────────
[providers.ollama-vps]
type            = "ollama"
endpoint        = "https://ollama.yourdomain.com"
model           = "qwen2.5-coder:7b"
timeout_seconds = 180
tls_skip_verify = false

# ── 3. OpenClaw Autonomous Agent (Remote VPS) ──────────────────────────────
[providers.openclaw-vps]
type            = "openclaw"
endpoint        = "https://agent.yourdomain.com"
token           = ""  # Export HIKARI_OPENCLAW_VPS_TOKEN="secret-token"
timeout_seconds = 180

# ── 4. Hermes Agent Backend ────────────────────────────────────────────────
[providers.hermes-local]
type            = "hermes"
endpoint        = "http://127.0.0.1:8080"
token           = ""  # Export HIKARI_HERMES_LOCAL_TOKEN="secret-token"
timeout_seconds = 120

[ui]
theme = "default"
ascii = "default"
```

---

## 3. Connecting to Ollama (Model Provider)

[Ollama](https://ollama.com/) allows running open-source LLMs locally (e.g., Llama 3.2, Qwen 2.5, DeepSeek-R1, Mistral, CodeLlama).

### 3.1 Local Workstation Setup

1. **Install Ollama**:
   ```bash
   # Linux / macOS
   curl -fsSL https://ollama.com/install.sh | sh
   ```
2. **Start the Ollama daemon**:
   ```bash
   ollama serve
   ```
3. **Pull your preferred model**:
   ```bash
   ollama pull llama3.2
   # or coding-specific models:
   ollama pull qwen2.5-coder:7b
   ```
4. **Configure in `config.toml`**:
   ```toml
   [providers.ollama-local]
   type     = "ollama"
   endpoint = "http://127.0.0.1:11434"
   model    = "llama3.2"
   ```

### 3.2 Remote VPS / Network Setup

If Ollama runs on a separate machine, home lab server, or cloud VPS:

1. **Configure Ollama to listen on all interfaces**:
   On the server running Ollama, set `OLLAMA_HOST`:
   ```bash
   # systemd service edit:
   sudo systemctl edit ollama.service
   ```
   Add:
   ```ini
   [Service]
   Environment="OLLAMA_HOST=0.0.0.0:11434"
   ```
   Restart the service:
   ```bash
   sudo systemctl daemon-reload && sudo systemctl restart ollama
   ```
2. **Secure with Reverse Proxy (Recommended)**:
   Place Nginx or Caddy with HTTPS and basic/bearer auth in front of port 11434.
3. **Configure in `config.toml`**:
   ```toml
   [providers.ollama-vps]
   type            = "ollama"
   endpoint        = "https://ollama.yourserver.com"
   model           = "llama3.2"
   timeout_seconds = 180
   tls_skip_verify = false  # Set to true only if using self-signed certs
   ```

### 3.3 Configuration Options

| Option | Type | Required | Default | Description |
|---|---|---|---|---|
| `type` | string | **Yes** | — | Must be `"ollama"` |
| `endpoint` | string | **Yes** | — | Base URL (e.g., `http://127.0.0.1:11434`) |
| `model` | string | Optional | `""` | Default model tag. Can be changed inside TUI via `/model` |
| `timeout_seconds` | int | Optional | `120` | HTTP request timeout in seconds |
| `tls_skip_verify` | bool | Optional | `false` | Disable TLS certificate checks (dev/testing only) |

### 3.4 Verification

Test raw Ollama connectivity before launching Hikari:
```bash
# Check available models on Ollama:
curl http://127.0.0.1:11434/api/tags

# Test with Hikari one-shot:
hikari --provider ollama-local "Reply with 'Ollama is online' in 5 words."
```

---

## 4. Connecting to OpenClaw (Agent Provider)

[OpenClaw](https://github.com/) is an autonomous AI agent gateway that supports tool calling, filesystem interactions, shell commands, and persistent memory.

### 4.1 Overview & Capabilities

When Hikari connects to OpenClaw:
- Queries execute within an active agent session.
- Hikari handles prompt streaming and rendering in the terminal.
- OpenClaw coordinates tool execution and server-side memory.

### 4.2 Configuration Options

In `~/.config/hikari/config.toml`:

```toml
[providers.openclaw-vps]
type            = "openclaw"
endpoint        = "https://agent.example.com"
token           = ""   # Leave blank if using environment variables
timeout_seconds = 180
```

| Option | Type | Required | Description |
|---|---|---|---|
| `type` | string | **Yes** | Must be `"openclaw"` |
| `endpoint` | string | **Yes** | Base URL of OpenClaw server (e.g. `https://agent.example.com`) |
| `token` | string | **Yes\*** | API Bearer token. Recommended to inject via env var. |
| `timeout_seconds` | int | Optional | Timeout for agent execution (default: `120`s) |

### 4.3 Security & Token Management

> [!IMPORTANT]
> Never commit plaintext API keys or tokens into `config.toml` if sharing configurations or backing up dotfiles to public Git repositories.

Hikari provides automatic environment variable overrides for any provider token. The format is:
```text
HIKARI_<PROVIDER_NAME_UPPERCASE>_TOKEN
```
Hyphens (`-`) in the provider name are replaced with underscores (`_`).

**Examples**:
- Provider name `openclaw-vps` → `HIKARI_OPENCLAW_VPS_TOKEN`
- Provider name `openclaw-prod` → `HIKARI_OPENCLAW_PROD_TOKEN`

Add to your `~/.bashrc`, `~/.zshrc`, or pass at runtime:
```bash
export HIKARI_OPENCLAW_VPS_TOKEN="sk-claw-xxxxxxxxxxxxxxxxxxxx"
```

Hikari automatically sanitizes tokens from debug output, session history files, and logs.

### 4.4 Verification

```bash
# Test connection with Hikari:
hikari --provider openclaw-vps "Hello! Report your active capabilities."
```

Inside the TUI, type:
```text
/status
```
Hikari will display connection latency and the capability checklist (Streaming, Tools, Exec, Memory).

---

## 5. Connecting to Hermes (Agent Provider)

[Hermes](https://github.com/) is an agent execution backend designed for task management, function calling, and structured conversation pipelines.

### 5.1 Overview & Capabilities

Hermes integrates as an agent provider, providing:
- Structured function and tool calls
- Persistent multi-turn conversation context
- Dynamic workflow execution

### 5.2 Configuration Options

In `~/.config/hikari/config.toml`:

```toml
# Local Hermes service
[providers.hermes-local]
type            = "hermes"
endpoint        = "http://127.0.0.1:8080"
timeout_seconds = 120

# Remote VPS Hermes service
[providers.hermes-vps]
type            = "hermes"
endpoint        = "https://hermes.example.com"
token           = ""  # Export HIKARI_HERMES_VPS_TOKEN
timeout_seconds = 180
```

| Option | Type | Required | Description |
|---|---|---|---|
| `type` | string | **Yes** | Must be `"hermes"` |
| `endpoint` | string | **Yes** | Base URL (e.g. `http://127.0.0.1:8080`) |
| `token` | string | Optional | API token for authenticated instances |
| `timeout_seconds` | int | Optional | Request timeout in seconds (default: `120`) |

### 5.3 Verification

```bash
# Test one-shot with Hermes:
hikari --provider hermes-local "Check in and status report"
```

---

## 6. Switching Providers in Hikari

Hikari allows effortless switching between providers across all interaction modes.

### 6.1 Set Default Provider
Set your preferred daily driver in `~/.config/hikari/config.toml`:
```toml
[default]
provider = "ollama-local"
```

### 6.2 CLI Override (One-Shot & Pipeline)
Use the `--provider` flag to override the default on a per-command basis:
```bash
# Route to local Ollama
hikari --provider ollama-local "Explain raft consensus"

# Route to remote OpenClaw agent
hikari --provider openclaw-vps "Audit this Dockerfile" < Dockerfile

# Route to Hermes
hikari --provider hermes-local "Generate database schema"
```

### 6.3 Interactive Switching in TUI
While running inside the interactive TUI (`hikari`):

1. **Switch Provider**:
   Type `/provider` in the input bar. A popup picker appears listing all configured providers and their types (`[model]` or `[agent]`). Use `↑` / `↓` and press `Enter` to switch instantly without restarting.

2. **Switch Model** (Ollama):
   Type `/model` in the input bar. Hikari queries the active Ollama instance for pulled models and lets you select one interactively.

3. **Check Connection Status**:
   Type `/status` to view ping latency, endpoint address, and capability flags.

---

## 7. Provider Capabilities Matrix

Hikari queries each provider's capabilities and dynamically enables or disables features in the UI:

| Feature | Ollama | OpenClaw | Hermes |
|---|:---:|:---:|:---:|
| **Provider Type** | Model | Agent | Agent |
| **Real-time Streaming** | ✅ Yes | ✅ Yes | ✅ Yes |
| **Model Listing (`/model`)** | ✅ Yes (`/api/tags`) | ❌ No | ❌ No |
| **Tool Calling** | ❌ No | ✅ Yes | ✅ Yes |
| **Filesystem Access** | ❌ No | ✅ Yes | ✅ Yes |
| **Command Execution** | ❌ No | ✅ Yes | ✅ Yes |
| **Agent Memory** | ❌ No | ✅ Yes | ✅ Yes |
| **Offline Execution** | ✅ Yes (Local) | ❌ No (Requires server) | ❌ No (Requires server) |

---

## 8. Troubleshooting & Common Issues

### Issue 1: `connection refused` connecting to Ollama
**Symptom**: `ollama connect: dial tcp 127.0.0.1:11434: connect: connection refused`  
**Cause**: The Ollama service is not running.  
**Solution**:
1. Check if Ollama is running:
   ```bash
   pgrep ollama || ollama serve
   ```
2. Verify port listening:
   ```bash
   curl -s http://127.0.0.1:11434/api/tags
   ```

---

### Issue 2: Remote Ollama on VPS times out or rejects connection
**Symptom**: `dial tcp <ip>:11434: i/o timeout` or `connection refused`  
**Cause**: By default, Ollama binds only to `127.0.0.1`. It does not listen on external network interfaces.  
**Solution**:
On the VPS host, set `OLLAMA_HOST=0.0.0.0:11434`:
```bash
export OLLAMA_HOST="0.0.0.0:11434"
ollama serve
```
If managed by `systemd`, configure `Environment="OLLAMA_HOST=0.0.0.0:11434"` in `/etc/systemd/system/ollama.service.d/override.conf`.

---

### Issue 3: Self-Signed Certificate Error on Private VPS
**Symptom**: `x509: certificate signed by unknown authority`  
**Solution**:
If using internal self-signed TLS certificates for development, set `tls_skip_verify = true` in the provider block:
```toml
[providers.ollama-vps]
type            = "ollama"
endpoint        = "https://192.168.1.100:11434"
tls_skip_verify = true
```

---

### Issue 4: `openclaw provider: token is required`
**Symptom**: Hikari exits with error indicating missing token for OpenClaw.  
**Cause**: OpenClaw is an agent provider that mandates authentication.  
**Solution**:
Export the token in your shell environment:
```bash
export HIKARI_OPENCLAW_VPS_TOKEN="your-token-here"
```
Or define it in `config.toml`:
```toml
[providers.openclaw-vps]
token = "your-token-here"
```

---

### Issue 5: Response generation times out on large models
**Symptom**: Request aborts after 120 seconds with context deadline exceeded.  
**Cause**: Large local models (e.g. 70B parameter models on CPU or limited VRAM) may take longer to begin inference.  
**Solution**:
Increase `timeout_seconds` in `config.toml`:
```toml
[providers.ollama-local]
timeout_seconds = 300
```
