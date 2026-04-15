<p align="center">
  <img src="https://img.shields.io/badge/cmux-claude_multiplexer-00d26a?style=for-the-badge&logo=gnometerminal&logoColor=white" alt="cmux" />
</p>

<h1 align="center">🖥️ cmux</h1>

<p align="center">
  <strong>A web-based terminal multiplexer for <a href="https://docs.anthropic.com/en/docs/claude-code">Claude Code</a> sessions</strong>
</p>

<p align="center">
  <a href="https://github.com/Corwind/cmux/actions"><img src="https://img.shields.io/github/actions/workflow/status/Corwind/cmux/ci.yml?style=flat-square&logo=github&label=CI" alt="CI" /></a>
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React" />
  <img src="https://img.shields.io/badge/TypeScript-5.7-3178C6?style=flat-square&logo=typescript&logoColor=white" alt="TypeScript" />
  <img src="https://img.shields.io/badge/Tailwind-4-06B6D4?style=flat-square&logo=tailwindcss&logoColor=white" alt="Tailwind" />
  <img src="https://img.shields.io/github/license/Corwind/cmux?style=flat-square&color=yellow" alt="License" />
</p>

<p align="center">
  <em>Create, organize, and interact with multiple Claude Code CLI sessions — right from your browser. 🚀</em>
</p>

---

<!-- Uncomment when you have a screenshot:
<p align="center">
  <img src="docs/screenshot.png" alt="cmux screenshot" width="900" />
</p>

--- -->

## ✨ Features

🧠 **Manage Claude Code sessions** — Create, list, and delete sessions from a sleek sidebar

🖥️ **Real terminal in the browser** — Full xterm.js terminal with 256-color support, cursor control, and TUI rendering

📂 **Built-in file browser** — Pick working directories visually when creating sessions

🔄 **Persistent sessions** — Sessions survive page reloads; reconnect to running Claude instances seamlessly

📐 **Resizable layout** — Drag to resize the sidebar, terminal auto-fits to available space

⚡ **One command to start** — `make dev` and you're up and running

## 🏗️ Architecture

```
┌───────────────────────────────────────────────────────┐
│                     Browser (React)                   │
│  ┌───────────┐  ┌────────────┐  ┌──────────────────┐  │
│  │  Sessions │  │    File    │  │     xterm.js     │  │
│  │  Sidebar  │  │   Browser  │  │     Terminal     │  │
│  └─────┬─────┘  └──────┬─────┘  └────────┬─────────┘  │
│        │ REST          │ REST            │ WebSocket  │
└────────┼───────────────┼─────────────────┼────────────┘
         │               │                 │
┌────────┼───────────────┼─────────────────┼────────────┐
│        ▼               ▼                 ▼    Go API  │
│  ┌────────────────────────────────────────────────┐   │
│  │              HTTP Router (chi)                 │   │
│  └──────────────────────┬─────────────────────────┘   │
│                         ▼                             │
│  ┌────────────────────────────────────────────────┐   │
│  │          Session Service (app layer)           │   │
│  └───────┬──────────────┬─────────────────┬───────┘   │
│          ▼              ▼                 ▼           │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   SQLite   │  │ PTY Manager  │  │  Filesystem  │   │
│  │   (store)  │  │ (creack/pty) │  │   Browser    │   │
│  └────────────┘  └──────┬───────┘  └──────────────┘   │
│                         │                             │
│                         ▼                             │
│                  ┌──────────────┐                     │
│                  │  sandbox-exec│                     │
│                  │   (macOS)    │                     │
│                  └──────┬───────┘                     │
│                         ▼                             │
│                  ┌──────────────┐                     │
│                  │  claude CLI  │                     │
│                  │    (PTY)     │                     │
│                  └──────────────┘                     │
└───────────────────────────────────────────────────────┘
```

The backend follows **hexagonal architecture** with clear boundaries:

| Layer | Purpose |
|-------|---------|
| 🎯 **Domain** | Session model, validation, business rules |
| 🔌 **Ports** | Interfaces for repository, process manager, filesystem |
| ⚙️ **App** | Session service orchestrating domain + ports |
| 🔧 **Adapters** | HTTP handlers, SQLite, PTY manager, filesystem |

## 🚀 Quick Start

### Prerequisites

- 🐹 [Go](https://go.dev/) 1.24+
- 📦 [Node.js](https://nodejs.org/) 20+
- 🤖 [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and authenticated

### Install & run

```bash
git clone https://github.com/Corwind/cmux.git
cd cmux
make install           # install Go + npm dependencies
make install-service   # build, codesign, and start the launchd service
```

Create a config file at `~/.cmux/config.toml` so spawned sessions inherit your shell environment:

```toml
[server]
port = "2689"
db_path = "~/.cmux/cmux.db"

[sandbox]
template_dir = "~/.cmux/sandbox-templates"
templates = []

[shell]
path = "/bin/zsh"
init_files = ["~/.zshrc", "~/.zprofile"]

[env]
```

Then open **http://localhost:2689** 🎉

The service starts automatically on login and restarts if it crashes.

### Other commands

```bash
make restart-service   # rebuild and restart the service
make uninstall-service # stop and remove the service
make service-logs      # tail -f ~/.cmux/cmux.log
make dev               # run in foreground (backend :2689 + frontend :5173)
make test              # run all tests (Go + Vitest)
make lint              # lint everything (golangci-lint + ESLint)
make clean             # remove build artifacts
```

## 📡 API

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/api/sessions` | 🆕 Create a session (spawns Claude PTY) |
| `GET` | `/api/sessions` | 📋 List all sessions |
| `GET` | `/api/sessions/:id` | 🔍 Get session details |
| `DELETE` | `/api/sessions/:id` | 🗑️ Kill process & delete session |
| `GET` | `/api/fs?path=` | 📂 Browse directories |
| `WS` | `/ws/sessions/:id` | 🔌 Terminal WebSocket (binary: PTY I/O, text: resize/status) |

## 🔒 Sandbox

Every Claude Code session runs inside a macOS `sandbox-exec` jail with **deny-by-default** rules. The sandbox restricts filesystem writes to the working directory, Claude config files, and temp directories — preventing Claude from modifying files outside its scope.

**What's allowed:**

| Access | Paths |
|--------|-------|
| **Read** | System paths (`/usr`, `/System`, `/Library`, `/bin`, `/sbin`, `/opt`, `/private`, `/dev`), working directory, Claude config (`~/.claude`, `~/.config`, `~/.local`), shell dotfiles, `~/Library/Keychains` |
| **Write** | Working directory, `~/.claude`, `~/.config`, `~/.claude.json`, temp dirs (`/tmp`, `/private/tmp`, `/private/var/folders`), `/dev` |
| **Blocked** | `~/.ssh`, `~/.aws`, `~/.gnupg`, `~/Documents`, `~/Desktop`, other projects, any path not explicitly listed |

The sandbox profile can be extended via templates stored in the `sandbox-profiles/` directory, or by passing custom SBPL rules through the session creation API.

## ⚙️ Configuration

cmux can be configured via a TOML file at `~/.cmux/config.toml`. This is especially useful when running as a launchd service, where spawned Claude processes would otherwise inherit a minimal environment.

```toml
[server]
port = "2689"
db_path = "~/.cmux/cmux.db"

[sandbox]
template_dir = "sandbox-profiles"
templates = ["standard"]

# Shell environment for spawned Claude processes
[shell]
path = "/bin/zsh"
init_files = ["~/.zshrc", "~/.zprofile"]

# Explicit env var overrides (applied on top of shell env)
[env]
PATH = "/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
GOPATH = "/Users/me/go"
```

| Section | Purpose |
|---------|---------|
| **`[server]`** | Port and database path |
| **`[sandbox]`** | Sandbox template directory and active templates |
| **`[shell]`** | Shell path and init files to source — captures the full environment for spawned processes |
| **`[env]`** | Explicit env var overrides applied on top of the captured shell environment |

**Precedence:** config file > environment variable (`CMUX_PORT`, `CMUX_DB_PATH`, etc.) > default

The config file is optional — if it doesn't exist, cmux falls back to environment variables and built-in defaults.

## 🛠️ Tech Stack

<table>
<tr><td>🔙 <strong>Backend</strong></td><td>Go, chi, SQLite (modernc.org), creack/pty, coder/websocket</td></tr>
<tr><td>🖼️ <strong>Frontend</strong></td><td>React 19, Vite 6, TypeScript, Tailwind CSS 4</td></tr>
<tr><td>📊 <strong>State</strong></td><td>TanStack Query 5 (server), Zustand 5 (client)</td></tr>
<tr><td>🖥️ <strong>Terminal</strong></td><td>xterm.js 6 with FitAddon + WebLinksAddon</td></tr>
<tr><td>🧪 <strong>Testing</strong></td><td>Go testing, Vitest, React Testing Library, Playwright</td></tr>
<tr><td>🐳 <strong>Containers</strong></td><td>Docker + docker-compose (frontend)</td></tr>
</table>

## 📁 Project Structure

```
cmux/
├── 📄 Makefile                    # Top-level commands
├── 🐳 docker-compose.yml         # Frontend container
│
├── 🔙 backend/
│   ├── cmd/cmux/main.go          # Entry point
│   └── internal/
│       ├── domain/                # 🎯 Session model + validation
│       ├── ports/                 # 🔌 Interfaces
│       ├── app/                   # ⚙️ Session service
│       └── adapters/
│           ├── http/              # 🌐 REST + WebSocket handlers
│           ├── sqlite/            # 💾 Persistence
│           ├── pty/               # 🖥️ PTY process management
│           │   └── sandbox/      # 🔒 sandbox-exec profile builder
│           └── filesystem/        # 📂 Directory browser
│
└── 🖼️ frontend/
    └── src/
        ├── components/layout/     # 📐 App shell (sidebar + terminal)
        ├── features/
        │   ├── sessions/          # 🧠 Session CRUD
        │   ├── terminal/          # 🖥️ xterm.js component
        │   └── file-browser/      # 📂 Directory picker
        └── pages/                 # 🏠 Home page
```

## 🤝 Contributing

Contributions welcome! Feel free to open issues and pull requests.

## 📝 License

[MIT](LICENSE)

---

<p align="center">
  Made with 💚 and <a href="https://docs.anthropic.com/en/docs/claude-code">Claude Code</a>
</p>
