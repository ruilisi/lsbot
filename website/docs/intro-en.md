English | [中文](./README.md)

---

# lingti-bot (Lingti) 🐕⚡

> 🐕⚡ **Minimal · Efficient · Compile Once Run Anywhere · Lightning Integration** AI Bot

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/Website-cli.lingti.com-blue?style=flat)](https://cli.lingti.com/bot)

**Lingti Bot** is an all-in-one AI Bot platform featuring \*\*MCP Server\*\*, \*\*multi-platform messaging gateway\*\*, \*\*rich toolset\*\*, and \*\*intelligent conversation\*\*.

**Core Advantages:**
- 🚀 **Zero Dependency** — Single 30MB binary, no Node.js/Python runtime needed, just `scp` and run
- ☁️ **Cloud Relay** — No public server, domain registration, or HTTPS certificate needed, 5-min setup for WeCom/WeChat
- 🤖 **Browser Automation** — Built-in CDP protocol control, snapshot-then-act pattern, no Puppeteer/Playwright installation
- 🛠️ **75+ MCP Tools** — Covers files, Shell, system, network, calendar, Git, GitHub, and more
- 🌏 **China Platform Native** — DingTalk, Feishu, WeCom, WeChat Official Account ready out-of-box
- 💬 **Built-in Web Chat UI** — Open in any browser, no client needed. Supports **multiple simultaneous sessions** each with isolated AI memory, persists across page refreshes
- 🔌 **Embedded Friendly** — Compile to ARM/MIPS, easy deployment to Raspberry Pi, routers, NAS
- 🧠 **Multi-AI Backend** — [16 AI providers](AI-PROVIDERS.md) including Claude, DeepSeek, Kimi, MiniMax, Gemini, OpenAI, with per-platform/channel model overrides
- 🔬 **Claude Extended Thinking** — Native Anthropic Thinking API support, real chain-of-thought reasoning via `/think high`
- 🐳 **Docker Support** — Multi-stage Dockerfile and docker-compose.yml for containerized deployment
- 🩺 **Health Diagnostics** — `lingti-bot doctor` checks config, connectivity, dependencies in one command

Supports WeCom, Feishu, DingTalk, Slack, Telegram, Discord, WhatsApp, LINE, Teams, and more — [19 chat platforms](docs/chat-platforms.md) in total — plus a built-in **browser Web Chat UI** with multiple parallel sessions. Either **5-minute cloud relay** or [OpenClaw](docs/openclaw-reference.md)-style **self-hosted deployment**. Check [Roadmap](docs/roadmap.md) for more features.

> 🐕⚡ **Why "Lingti"?** Lingti (灵缇) means Greyhound in Chinese - the fastest dog in the world, known for agility and loyalty. Lingti Bot is equally agile and efficient, your faithful AI assistant.

## Installation

### macOS / Linux / WSL

```bash
curl -fsSL https://files.lingti.com/install-bot.sh | bash
```

### Windows (PowerShell)

```powershell
irm https://cli.lingti.com/install.ps1 | iex
```

After installation, run the interactive setup wizard:

```bash
lingti-bot onboard
```

Once configured, start with no arguments needed:

```bash
lingti-bot relay
```

Or pass arguments directly to run multiple instances or override saved config:

```bash
lingti-bot relay --platform wecom --provider deepseek --api-key sk-xxx
```

## Examples

### Intelligent Chat, File Management, Information Retrieval
<table>
<tr>
<td width="33%"><img src="docs/images/demo-chat-1.png" alt="Smart Assistant" /></td>
<td width="33%"><img src="docs/images/demo-chat-2.png" alt="WeCom File Transfer" /></td>
<td width="33%"><img src="docs/images/demo-chat-3.png" alt="Information Search" /></td>
</tr>
<tr>
<td align="center"><sub>💬 Smart Chat</sub></td>
<td align="center"><sub>📁 WeCom File Transfer</sub></td>
<td align="center"><sub>🔍 Information Search</sub></td>
</tr>
</table>

<summary>📺 <b>Background Running Demo</b> — <code>make && dist/lingti-bot gateway</code></summary>
<br>
<img src="docs/images/demo-terminal.png" alt="Terminal Demo" />
<p><sub>Clone, compile and run directly, paired with DeepSeek model, processing DingTalk messages in real-time</sub></p>

## Why lingti-bot?

**Single Binary, Zero Dependencies, Local-First**

Unlike traditional Bot frameworks that require Docker, databases, or complex runtime environments, lingti-bot achieves the ultimate in simplicity:

1. **Zero Dependencies** — One 30MB binary file, no external dependencies, `scp` to any server and run
2. **Embedded Friendly** — Pure Go implementation, compilable to ARM/MIPS, deployable to Raspberry Pi, routers, NAS
3. **Plain Text Output** — No colored terminal output, avoiding extra rendering libraries or terminal compatibility issues
4. **Code Restraint** — Every line of code has a clear reason to exist, rejecting over-design
5. **Cloud Relay Boost** — No need for self-hosted web server, cloud relay completes WeChat Official Account and WeCom callback verification in seconds, Bot goes live immediately

```bash
# Clone, compile, run
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot && make
./dist/lingti-bot gateway --provider deepseek --api-key sk-xxx
```

### Single Binary

```bash
# Compile
make build

# Ready to use
./dist/lingti-bot serve
```

No Docker, no database, no cloud service required.

### Local-First

All functions run locally, data is not uploaded to the cloud. Your files, calendar, and process information are safely kept locally.

### Cross-Platform Support

Core functions support macOS, Linux, Windows. macOS users can enjoy native calendar, reminders, notes, music control and more.

**Supported Platforms:**

| Platform | Architecture | Build Command |
|----------|--------------|---------------|
| macOS | ARM64 (Apple Silicon) | `make darwin-arm64` |
| macOS | AMD64 (Intel) | `make darwin-amd64` |
| Linux | AMD64 | `make linux-amd64` |
| Linux | ARM64 | `make linux-arm64` |
| Linux | MIPS | `make linux-mips` |
| Windows | AMD64 | `make windows-amd64` |

## Architecture

### MCP Server — Claude Desktop Native Integration

lingti-bot is primarily an **MCP (Model Context Protocol) Server** that provides rich local tools for Claude Desktop and other MCP-compatible clients.

**Quick Start:**
1. Install lingti-bot:
   - macOS / Linux / WSL: `curl -fsSL https://files.lingti.com/install-bot.sh | bash`
   - Windows (PowerShell): `irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot`
2. Configure Claude Desktop MCP: `~/.config/Claude/claude_desktop_config.json`
   ```json
   {
     "mcpServers": {
       "lingti-bot": {
         "command": "lingti-bot",
         "args": ["serve"]
       }
     }
   }
   ```
3. Restart Claude Desktop, you can now use 75+ local tools

### Multi-Platform Gateway — Message Router

In addition to MCP mode, lingti-bot can also run as a **message router**, connecting multiple messaging platforms simultaneously.

**Supported Platforms:**

| Platform | Connection Method | Setup | File Sending | Status |
|----------|-------------------|-------|-------------|--------|
| **WeCom** | Callback API | Cloud Relay / Self-hosted | ✅ All formats | ✅ |
| **WeChat Official** | Cloud Relay | 10 seconds | ✅ Image/Voice/Video | ✅ |
| **DingTalk** | Stream Mode | One-click | 🔜 Planned | ✅ |
| **Feishu/Lark** | WebSocket | One-click | 🔜 Planned | ✅ |
| **Slack** | Socket Mode | One-click | 🔜 Planned | ✅ |
| **Telegram** | Bot API | One-click | 🔜 Planned | ✅ |
| **Discord** | Gateway | One-click | 🔜 Planned | ✅ |
| **WhatsApp** | Webhook + Graph API | Self-hosted | 🔜 Planned | ✅ |
| **LINE** | Webhook + Push API | Self-hosted | 🔜 Planned | ✅ |
| **Microsoft Teams** | Bot Framework | Self-hosted | 🔜 Planned | ✅ |
| **Matrix / Element** | HTTP Sync | Self-hosted | 🔜 Planned | ✅ |
| **Google Chat** | Webhook + REST | Self-hosted | 🔜 Planned | ✅ |
| **Mattermost** | WebSocket + REST | Self-hosted | 🔜 Planned | ✅ |
| **iMessage** | BlueBubbles | Self-hosted | 🔜 Planned | ✅ |
| **Signal** | signal-cli REST | Self-hosted | 🔜 Planned | ✅ |
| **Twitch** | IRC | Self-hosted | — | ✅ |
| **NOSTR** | WebSocket Relays | Self-hosted | 🔜 Planned | ✅ |
| **Zalo** | Webhook + REST | Self-hosted | 🔜 Planned | ✅ |
| **Nextcloud Talk** | HTTP Polling | Self-hosted | 🔜 Planned | ✅ |
| **Web Chat UI** | WebSocket (built-in) | `--webapp-port` | — | ✅ |

> File sending details (setup, supported types, limitations): [File Sending Guide](docs/file-sending.md)

> Full list with config details and env vars: [Chat Platforms](docs/chat-platforms.md)

**Cloud Relay Advantage:** No public server, no domain registration, no HTTPS certificate, no firewall configuration, 5 minutes to complete integration.

### MCP Toolset — 75+ Local System Tools

Covers all aspects of daily work, making AI your all-around assistant.

| Category | Tools | Features |
|----------|-------|----------|
| **File Operations** | 9 | Read/write, search, organize, batch delete, trash |
| **Shell Commands** | 2 | Command execution, path finding |
| **System Info** | 4 | CPU, memory, disk, environment variables |
| **Process Management** | 3 | List, details, terminate |
| **Network Tools** | 4 | Interfaces, connections, Ping, DNS |
| **Calendar** | 6 | View, create, search, delete events (macOS) |
| **Browser Automation** | 12 | Snapshot, click, input, screenshot, tab management |
| **Scheduled Tasks** | 5 | Create, list, delete, pause, resume cron jobs |

### Scheduled Tasks — Automate Your Workflow

Use standard Cron expressions to schedule periodic tasks for true unattended automation.

**Core Features:**
- 🕐 Support standard Cron expressions (minute, hour, day, month, weekday)
- 💾 Task persistence, auto-resume after restart
- 🔄 Pause/resume task execution
- 📊 Record execution status and error information
- 🛠️ Call any MCP tool

**Quick Examples:**

```bash
# Daily backup at 2 AM
cron_create(
  name="daily-backup",
  schedule="0 2 * * *",
  tool="shell_execute",
  arguments={"command": "tar -czf ~/backup-$(date +%Y%m%d).tar.gz ~/data"}
)

# Check disk space every 15 minutes
cron_create(
  name="disk-check",
  schedule="*/15 * * * *",
  tool="disk_usage",
  arguments={"path": "/"}
)

# Weekday morning standup reminder at 9 AM
cron_create(
  name="morning-standup",
  schedule="0 9 * * 1-5",
  tool="notification_send",
  arguments={"title": "Standup Reminder", "message": "Time for daily standup!"}
)
```

**Cron Expression Format:**

```
* * * * *
│ │ │ │ │
│ │ │ │ └─ Day of week (0-6, 0=Sunday)
│ │ │ └─── Month (1-12)
│ │ └───── Day of month (1-31)
│ └─────── Hour (0-23)
└───────── Minute (0-59)
```

**Common Examples:**
- `0 * * * *` - Every hour
- `*/15 * * * *` - Every 15 minutes
- `0 9 * * 1-5` - Weekdays at 9 AM
- `0 0 1 * *` - First day of every month
- `30 8-18 * * *` - Every hour from 8:30 to 18:30

Task configuration saved to `~/.lingti.db` (SQLite), auto-resume after MCP service restart.

### Built-in Web Chat UI — Open in Any Browser

No client apps needed. Start the web chat interface with a single flag:

```bash
lingti-bot gateway --provider deepseek --api-key sk-xxx --webapp-port 8080
# Open http://localhost:8080
```

<p align="center">
<img src="docs/images/webapp-demo.png" alt="Web Chat UI Demo" width="800" />
</p>

**Key features:**

| Feature | Details |
|---------|---------|
| **Multiple simultaneous sessions** | Each session in the sidebar is fully independent — talk to the bot on different tasks at the same time without interference |
| **Isolated memory per session** | Each session has its own `channelID` → separate AI conversation history. What you say in session A never bleeds into session B |
| **True parallel processing** | Start a long task in session A, immediately switch to session B and send a new message — both are processed concurrently |
| **Session persistence** | Session list and chat history are saved in browser `localStorage` — survive page refreshes and browser restarts |
| **Markdown rendering** | Bot replies render full Markdown: code blocks with syntax hints, tables, lists, bold/italic |
| **Auto port increment** | If the configured port is busy, automatically tries the next port until one is free |
| **Zero extra dependencies** | UI is a single embedded HTML file — no Node.js, no npm, no build step |

**Multiple sessions in action:**

```
Browser tab (http://localhost:8080)
├── Session A  ── "Analyze this CSV file..." (AI still working)
├── Session B  ── "Write a poem about cats" (AI responded instantly)
└── Session C  ── Yesterday's conversation (still accessible)
```

**Via config file:**

```yaml
# ~/.lingti.yaml
platforms:
  webapp:
    port: 8080
```

**Via environment variable:**

```bash
WEBAPP_PORT=8080 lingti-bot gateway --provider deepseek --api-key sk-xxx
```

### Skills — Modular Capability Packs

Skills are modular capability packs that teach lingti-bot how to use external tools. Each skill is a directory containing a `SKILL.md` file with YAML frontmatter for metadata and Markdown body for AI instructions.

```bash
# List all discovered skills
lingti-bot skills

# Check readiness status
lingti-bot skills check

# Get details on a specific skill
lingti-bot skills info github
```

Ships with 8 bundled skills: Discord, GitHub, Slack, Peekaboo (macOS UI automation), Tmux, Weather, 1Password, and Obsidian. Supports user-custom and project-specific skills.

See [Skills Guide](docs/skills.md) for full documentation.

### Multi-AI Backend

Supports **15 AI providers** covering mainstream LLM platforms globally:

| # | Provider | Name | Default Model |
|---|----------|------|---------------|
| 1 | `deepseek` | DeepSeek (recommended) | `deepseek-chat` |
| 2 | `qwen` | Qwen / 通义千问 | `qwen-plus` |
| 3 | `claude` | Claude (Anthropic) | `claude-sonnet-4-20250514` |
| 4 | `kimi` | Kimi / Moonshot | `moonshot-v1-8k` |
| 5 | `minimax` | MiniMax / 海螺 AI | `MiniMax-Text-01` |
| 6 | `doubao` | Doubao / 豆包 (ByteDance) | `doubao-pro-32k` |
| 7 | `zhipu` | Zhipu GLM / 智谱 | `glm-4-flash` |
| 8 | `openai` | OpenAI (GPT) | `gpt-4o` |
| 9 | `gemini` | Gemini (Google) | `gemini-2.0-flash` |
| 10 | `yi` | Yi / 零一万物 | `yi-large` |
| 11 | `stepfun` | StepFun / 阶跃星辰 | `step-2-16k` |
| 12 | `baichuan` | Baichuan / 百川智能 | `Baichuan4` |
| 13 | `spark` | Spark / 讯飞星火 (iFlytek) | `generalv3.5` |
| 14 | `siliconflow` | SiliconFlow / 硅基流动 (aggregator) | `Qwen/Qwen2.5-72B-Instruct` |
| 15 | `grok` | Grok (xAI) | `grok-2-latest` |

> Full list with API key links and aliases: [AI Providers](AI-PROVIDERS.md)

```bash
# Specify provider via command line
lingti-bot gateway --provider qwen --api-key "sk-xxx" --model "qwen-plus"

# Override default model
lingti-bot relay --provider openai --api-key "sk-xxx" --model "gpt-4o-mini"
```

## Documentation

- [AI Providers](AI-PROVIDERS.md) - 15 supported AI providers with API key links and aliases
- [Chat Platforms](docs/chat-platforms.md) - 19 supported chat platforms with config details and env vars
- [CLI Reference](docs/cli-reference.md) - Complete CLI documentation
- [Skills Guide](docs/skills.md) - Modular capability packs: create, discover, manage skills
- [Slack Integration Guide](docs/slack-integration.md) - Complete Slack app configuration tutorial
- [Feishu Integration Guide](docs/feishu-integration.md) - Feishu/Lark app configuration tutorial
- [WeCom Integration Guide](docs/wecom-integration.md) - WeCom app configuration tutorial
- [File Sending Guide](docs/file-sending.md) - Per-platform file transfer capabilities, setup, and limitations
- [Browser Automation Guide](docs/browser-automation.md) - Snapshot-then-act browser control
- [OpenClaw Feature Comparison](docs/openclaw-feature-comparison.md) - Detailed feature difference analysis

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contact

- Website: [cli.lingti.com](https://cli.lingti.com/bot)
- Email: `jiefeng@ruc.edu.cn` / `jiefeng.hopkins@gmail.com`
- GitHub: [github.com/ruilisi/lingti-bot](https://github.com/ruilisi/lingti-bot)
