# Hikari — Product Requirements Document (PRD)

**Product:** Hikari
**Type:** Open-source Terminal AI Platform
**Primary Language:** Go
**Target Platform:** Linux, macOS, Windows
**License:** Open Source

---

# 1. Product Overview

## 1.1 Vision

> **One terminal platform, unlimited AI personalities.**

Hikari adalah platform AI berbasis terminal yang menyediakan interface CLI/TUI untuk berinteraksi dengan berbagai AI agent dan model backend.

Hikari tidak bergantung pada satu AI backend. Pengguna dapat menghubungkan Hikari dengan backend seperti **OpenClaw, Hermes, dan Ollama**, baik yang berjalan secara lokal maupun pada server remote/VPS.

Hikari berfokus pada pengalaman terminal, konfigurasi, persona, session, dan provider management, sementara proses AI dapat dijalankan oleh backend yang dipilih pengguna.

---

# 2. Problem

Penggunaan AI assistant saat ini sering bergantung pada aplikasi GUI atau platform chat tertentu.

Untuk pengguna teknis, developer, Linux user, dan sysadmin, terminal merupakan environment utama untuk:

* coding;
* debugging;
* server management;
* Git;
* system administration;
* automation;
* log analysis;
* development workflow.

Namun setiap AI backend memiliki interface dan cara konfigurasi yang berbeda.

Hikari bertujuan menyediakan **satu terminal interface** untuk berbagai backend AI sehingga pengguna tidak perlu mengubah workflow ketika mengganti backend.

---

# 3. Goals

Hikari memiliki tujuan:

1. Menyediakan terminal-native AI interface.
2. Mendukung interactive TUI.
3. Mendukung CLI one-shot command.
4. Mendukung pipe/stdin workflow.
5. Mendukung berbagai AI provider.
6. Mendukung backend lokal dan remote/VPS.
7. Memisahkan platform Hikari dari persona AI.
8. Menyediakan sistem persona yang dapat dikustomisasi.
9. Menyediakan session dan conversation history.
10. Menyediakan provider switching tanpa mengubah aplikasi.
11. Mudah di-install dan digunakan.
12. Ramah untuk pengembangan open-source dan community contribution.

---

# 4. Non-Goals

Hikari tidak bertujuan untuk:

* membuat atau melatih LLM sendiri;
* menggantikan OpenClaw;
* menggantikan Hermes;
* menjadi model inference engine;
* menjadi aplikasi GUI desktop;
* mengimplementasikan seluruh logic agent di dalam Hikari;
* mengunci pengguna pada satu AI provider.

Hikari berfungsi sebagai **terminal interface dan orchestration layer** untuk berbagai AI backend.

---

# 5. Target Users

## 5.1 Developers

Pengguna yang membutuhkan AI untuk:

* coding;
* debugging;
* code review;
* Git;
* documentation;
* project management.

## 5.2 Linux / Server Users

Pengguna yang bekerja dengan:

* terminal;
* VPS;
* Docker;
* SSH;
* server monitoring;
* logs;
* system administration.

## 5.3 AI / Open Source Enthusiasts

Pengguna yang ingin:

* menjalankan AI secara lokal;
* menggunakan AI melalui VPS;
* mengganti model/backend;
* membuat persona sendiri;
* membuat extension atau provider baru.

---

# 6. Core Concept

Hikari memiliki tiga komponen utama:

```text
Hikari
 │
 ├── Terminal Interface
 │
 ├── Core / Router
 │
 └── Provider Layer
```

Provider Layer memungkinkan Hikari berkomunikasi dengan berbagai backend.

---

# 7. Provider Architecture

Salah satu prinsip utama Hikari adalah **provider-agnostic architecture**.

Hikari tidak boleh mengasumsikan bahwa hanya satu backend yang tersedia.

## 7.1 Provider Categories

Provider dibagi menjadi dua kategori utama.

### Agent Providers

Backend yang menyediakan kemampuan AI agent.

Contoh:

```text
OpenClaw
Hermes
```

Agent provider dapat memiliki kemampuan seperti:

* conversation;
* tool calling;
* filesystem access;
* command execution;
* memory;
* autonomous task execution.

### Model Providers

Backend yang menyediakan inference/model API.

Contoh:

```text
Ollama
```

Model provider berfokus pada komunikasi dengan model AI.

---

# 8. High-Level Architecture

```text
                         HIKARI
                           │
                  ┌────────┴────────┐
                  │                 │
                 TUI               CLI
                  │                 │
                  └────────┬────────┘
                           │
                     Hikari Core
                           │
                     Provider Router
                           │
              ┌────────────┴────────────┐
              │                         │
        Agent Providers           Model Providers
              │                         │
       ┌──────┴──────┐                  │
       │             │                  │
    OpenClaw       Hermes             Ollama
       │             │                  │
       └──────┬──────┘                  │
              │                         │
         Local / VPS               Local / VPS
```

Arsitektur ini memungkinkan Hikari menggunakan backend yang berbeda tanpa mengubah interface utama.

---

# 9. Deployment Architecture

Hikari harus mendukung backend yang berjalan di mesin yang sama maupun remote server.

## 9.1 Full Local

```text
┌─────────────────────────────┐
│          Computer           │
│                             │
│  Hikari ────────► Ollama    │
│                      │      │
│                    Model    │
└─────────────────────────────┘
```

Contoh:

```text
Hikari
  ↓
Ollama localhost
  ↓
Local Model
```

---

## 9.2 Local Hikari → VPS OpenClaw

```text
Computer                  VPS
────────                  ───

Hikari ───── HTTPS ─────► OpenClaw
                            │
                            ▼
                           LLM
```

---

## 9.3 Local Hikari → VPS Hermes

```text
Computer                  VPS
────────                  ───

Hikari ───── HTTPS ─────► Hermes
                            │
                            ▼
                           LLM
```

---

## 9.4 Local Hikari → VPS Ollama

```text
Computer                  VPS
────────                  ───

Hikari ───── HTTPS ─────► Ollama
                            │
                            ▼
                          Model
```

---

# 10. Provider Interface

Hikari harus memiliki abstraction layer agar provider dapat diganti tanpa mengubah TUI atau CLI.

Konsep awal:

```go
type Provider interface {
    Connect() error
    Chat(session string, message string) (<-chan Event, error)
    Models() ([]Model, error)
    Status() ProviderStatus
}
```

Implementasi provider:

```text
OpenClawProvider
HermesProvider
OllamaProvider
```

Namun abstraction tidak boleh memaksakan capability yang tidak dimiliki semua provider.

Contohnya, Ollama tidak boleh diasumsikan memiliki kemampuan agent seperti filesystem access atau autonomous tools.

Karena itu Hikari harus memiliki **capability-aware provider system**.

Contoh:

```text
Provider
 ├── Chat
 ├── Streaming
 ├── Models
 ├── Tools
 ├── Memory
 ├── Agent Execution
 └── ...
```

Provider hanya mengaktifkan capability yang memang didukung.

---

# 11. Provider Configuration

Provider disimpan di configuration Hikari.

Contoh:

```toml
[default]
provider = "openclaw-vps"

[providers.openclaw-vps]
type = "openclaw"
endpoint = "https://agent.example.com"
token = "..."

[providers.hermes-local]
type = "hermes"
endpoint = "http://127.0.0.1:8080"

[providers.ollama-local]
type = "ollama"
endpoint = "http://127.0.0.1:11434"

[providers.ollama-vps]
type = "ollama"
endpoint = "https://ollama.example.com"
```

Credential sensitif sebaiknya tidak disimpan secara plaintext apabila tersedia mekanisme secure storage pada platform.

---

# 12. Provider Switching

Pengguna dapat menentukan provider melalui CLI.

Contoh:

```bash
hikari --provider ollama-local
```

atau:

```bash
hikari --provider openclaw-vps
```

Provider juga dapat diganti dari dalam TUI:

```text
/provider
```

Contoh:

```text
Available Providers

> openclaw-vps
  hermes-local
  ollama-local
  ollama-vps
```

Provider dapat diganti tanpa restart Hikari apabila capability dan session backend memungkinkan.

---

# 13. CLI Modes

Hikari memiliki tiga mode utama.

## 13.1 Interactive Mode

```bash
hikari
```

Membuka full-screen TUI.

Contoh interface:

```text
╭──────────────────────────────────────────────╮
│ HIKARI                         OpenClaw ●    │
├──────────────────────────────────────────────┤
│                                              │
│  Nino                                        │
│  ────                                        │
│  Hey, what are we working on?                │
│                                              │
│  You                                         │
│  Check why nginx is not running.             │
│                                              │
│  Nino                                        │
│  I'll check the service status first...      │
│                                              │
├──────────────────────────────────────────────┤
│ > Type your message...                       │
╰──────────────────────────────────────────────╯
```

---

# 14. One-Shot Mode

Pengguna dapat menjalankan prompt langsung.

```bash
hikari "cek kenapa nginx mati"
```

Output:

```text
Nino:
I'll check the nginx service status...
```

---

# 15. Pipe / Stdin Mode

Hikari harus dapat menerima input dari program lain.

Contoh:

```bash
cat error.log | hikari "analisa error ini"
```

atau:

```bash
git diff | hikari "review perubahan ini"
```

atau:

```bash
journalctl -u nginx | hikari "cari penyebab error"
```

Hal ini memungkinkan Hikari menjadi bagian dari Unix-style workflow.

---

# 16. TUI

TUI dibangun menggunakan:

* Go;
* Bubble Tea;
* Lip Gloss.

Komponen utama:

```text
Header
Status Bar
Chat View
Input View
Sidebar / Context
Command Interface
```

Fitur TUI:

* keyboard navigation;
* scrolling;
* streaming response;
* command completion;
* provider indicator;
* model indicator;
* persona indicator;
* session indicator.

---

# 17. Internal Commands

Hikari menyediakan command yang tidak dikirim ke AI.

Contoh:

```text
/help
/clear
/exit
/status
/model
/provider
/persona
/session
/history
/settings
```

Contoh:

```text
/provider
```

digunakan untuk memilih backend.

Sedangkan:

```text
/persona
```

digunakan untuk memilih karakter AI.

---

# 18. Persona System

Persona merupakan identitas dan behavior AI yang terpisah dari platform Hikari.

Hikari bukan nama karakter.

Contoh:

```text
Platform:
Hikari

Personas:
Nino
Hikari
Developer
SysAdmin
Custom
```

Persona harus disimpan sebagai configuration/data, bukan hardcoded ke source code.

---

# 19. Persona Configuration

Contoh:

```yaml
name: Nino
description: Friendly coding assistant
greeting: "Hey, what are we working on?"
system_prompt: |
  You are Nino.
  You are a helpful coding assistant.
behavior:
  tone: casual
  language: Indonesian
```

Persona dapat memiliki:

* name;
* description;
* greeting;
* system prompt;
* tone;
* behavior;
* language;
* personality configuration.

---

# 20. Persona Switching

Persona dapat diganti tanpa restart.

```text
/persona
```

Contoh:

```text
Available Personas

> Nino
  Developer
  SysAdmin
  Hikari
  Custom
```

---

# 21. Custom Persona

Pengguna dapat membuat persona sendiri.

Directory:

```text
~/.config/hikari/personas/
```

Contoh:

```text
personas/
├── nino.yaml
├── developer.yaml
├── sysadmin.yaml
└── my-character.yaml
```

Community dapat membagikan persona tanpa perlu memodifikasi source code Hikari.

---

# 22. Theme System

Hikari harus memiliki theme system yang dapat dikustomisasi.

Contoh:

```text
~/.config/hikari/themes/
```

Theme dapat mengatur:

* primary color;
* secondary color;
* background;
* text;
* accent;
* border;
* status;
* error;
* code block.

---

# 23. ASCII System

ASCII art juga harus dapat dikustomisasi.

Directory:

```text
~/.config/hikari/ascii/
```

Contoh:

```text
ascii/
├── default.txt
├── minimal.txt
└── custom.txt
```

Pengguna dapat mengganti ASCII tanpa mengubah source code.

---

# 24. Session System

Hikari harus mendukung session.

Contoh:

```text
/session
```

Session dapat menyimpan:

* provider;
* persona;
* conversation;
* timestamp;
* metadata;
* model information.

Directory:

```text
~/.config/hikari/sessions/
```

Session memungkinkan pengguna melanjutkan conversation sebelumnya.

---

# 25. History

Hikari menyediakan conversation history.

```text
/history
```

Contoh:

```text
Today
 ├── nginx debugging
 ├── Git review
 └── Linux troubleshooting

Yesterday
 └── Docker configuration
```

---

# 26. Configuration

Default configuration directory:

```text
~/.config/hikari/
```

Struktur:

```text
hikari/
├── config.toml
├── personas/
├── themes/
├── ascii/
└── sessions/
```

---

# 27. Security

Karena Hikari dapat terhubung ke remote backend, security menjadi bagian penting.

Hikari harus mendukung:

* API token;
* HTTPS;
* authentication configuration;
* secure credential storage;
* configurable timeout;
* connection validation;
* provider health check.

Hikari tidak boleh mengekspos credential melalui:

* terminal output;
* logs;
* debug output;
* error message.

---

# 28. Provider Health Check

Hikari harus dapat mengetahui status provider.

Contoh:

```text
Provider Status

OpenClaw VPS     ● Connected
Hermes Local     ● Connected
Ollama Local     ● Connected
Ollama VPS       ○ Offline
```

Status juga ditampilkan pada TUI.

---

# 29. Technology Stack

## Core

```text
Go
```

## TUI

```text
Bubble Tea
Lip Gloss
```

## CLI

```text
Cobra
```

## Configuration

```text
TOML
YAML
```

## Build / Release

```text
GitHub Actions
GoReleaser
```

---

# 30. Repository Structure

```text
hikari/
│
├── cmd/
│   └── hikari/
│       └── main.go
│
├── internal/
│   ├── app/
│   ├── tui/
│   ├── cli/
│   ├── provider/
│   │   ├── openclaw/
│   │   ├── hermes/
│   │   └── ollama/
│   │
│   ├── persona/
│   ├── config/
│   ├── session/
│   └── history/
│
├── assets/
│   ├── ascii/
│   └── themes/
│
├── scripts/
│   └── install.sh
│
├── .github/
│   └── workflows/
│
├── go.mod
├── go.sum
├── LICENSE
├── README.md
└── CONTRIBUTING.md
```

---

# 31. Installation

Target installation experience:

```bash
curl -fsSL https://.../install.sh | sh
```

Setelah installation:

```bash
hikari
```

langsung membuka TUI.

Future distribution:

```text
Homebrew
AUR
Binary Releases
Package Managers
Docker
```

---

# 32. Open Source Strategy

Hikari akan dikembangkan sebagai open-source project.

Community dapat berkontribusi pada:

* provider;
* persona;
* theme;
* ASCII;
* TUI;
* integrations;
* documentation.

Future provider ecosystem:

```text
hikari-provider-openclaw
hikari-provider-hermes
hikari-provider-ollama
hikari-provider-xxx
```

Tujuannya adalah memungkinkan provider baru ditambahkan tanpa mengubah core Hikari secara besar-besaran.

---

# 33. MVP

MVP harus memiliki:

### Core

* [ ] Go application
* [ ] `hikari` executable
* [ ] configuration system
* [ ] provider abstraction

### TUI

* [ ] Interactive terminal interface
* [ ] keyboard input
* [ ] scrolling
* [ ] streaming response
* [ ] status indicator

### Providers

* [ ] OpenClaw provider
* [ ] Hermes provider
* [ ] Ollama provider
* [ ] local endpoint support
* [ ] remote endpoint support

### Persona

* [ ] basic persona system
* [ ] custom persona
* [ ] persona switching

### Session

* [ ] conversation session
* [ ] session persistence
* [ ] history

### CLI

* [ ] interactive mode
* [ ] one-shot mode
* [ ] stdin/pipe mode
* [ ] provider selection

### Commands

* [ ] `/help`
* [ ] `/clear`
* [ ] `/exit`
* [ ] `/status`
* [ ] `/model`
* [ ] `/provider`
* [ ] `/persona`
* [ ] `/session`
* [ ] `/history`

---

# 34. Post-MVP

Fitur setelah MVP:

* advanced themes;
* advanced persona configuration;
* provider marketplace;
* plugin system;
* community persona registry;
* community theme registry;
* model management;
* tool capability management;
* advanced authentication;
* encrypted credential storage;
* remote session synchronization;
* multiple concurrent sessions;
* provider fallback;
* automatic provider routing.

---

# 35. Long-Term Vision

Hikari berkembang menjadi sebuah **universal terminal AI platform**.

Pengguna tidak perlu peduli AI backend apa yang digunakan.

Mereka cukup menggunakan:

```bash
hikari
```

Kemudian memilih:

```text
Provider
 ├── OpenClaw
 ├── Hermes
 ├── Ollama
 └── Future Providers
```

dan:

```text
Persona
 ├── Nino
 ├── Developer
 ├── SysAdmin
 └── Custom
```

Dengan demikian:

```text
Hikari = Terminal Platform

Provider = AI Backend

Persona = AI Character / Behavior

Model = AI Intelligence

Session = Conversation Context
```

Keempat komponen tersebut dibuat terpisah agar pengguna dapat mengombinasikannya sesuai kebutuhan.

---

# 36. Success Criteria

Hikari dianggap berhasil mencapai MVP apabila pengguna dapat:

1. Install Hikari dari satu command.
2. Menjalankan `hikari`.
3. Masuk ke interactive TUI.
4. Memilih provider.
5. Terhubung ke OpenClaw, Hermes, atau Ollama.
6. Menggunakan backend lokal maupun remote.
7. Mengirim prompt dan menerima streaming response.
8. Mengganti persona.
9. Membuat persona sendiri.
10. Menyimpan dan melanjutkan session.
11. Menggunakan Hikari melalui pipe/stdin.
12. Menggunakan Hikari tanpa perlu mengetahui implementasi internal masing-masing provider.

---

# 37. Core Product Principle

> **Hikari should be the interface, not the intelligence.**

Hikari bertanggung jawab terhadap:

```text
UX
TUI
CLI
Configuration
Personas
Sessions
Provider Routing
```

Sedangkan backend bertanggung jawab terhadap:

```text
AI
Models
Tools
Agents
Memory
Inference
```

Dengan pemisahan tersebut, Hikari dapat berkembang tanpa terkunci pada satu AI ecosystem.
