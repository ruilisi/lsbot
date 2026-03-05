# Command Line Reference

Complete command line reference for lingti-bot.

## Table of Contents

- [Global Options](#global-options)
- [Commands](#commands)
  - [channels](#channels) — Manage platform credentials
  - [agents](#agents) — Manage agents and routing bindings
  - [gateway](#gateway) — Start everything (unified run command)
  - [serve](#serve) — Start MCP server
  - [relay](#relay) — Cloud relay connection
  - [doctor](#doctor) — Check system health
  - [skills](#skills) — Manage modular skills
  - [version](#version) — Show version
- [router (deprecated)](#router-deprecated)
- [Environment Variables](#environment-variables)
- [AI Providers](#ai-providers)
- [Configuration File](#configuration-file)

---

## Global Options

These options are available for all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `-h, --help` | | Show help for any command |
| `--log` | `info` | Log level: trace, debug, info, warn, error, fatal, panic |
| `-y, --yes` | `false` | Auto-approve all operations (skip confirmation prompts) |
| `--no-files` | `false` | Disable all file operation tools |

---

## Commands

### channels

Manage platform channel credentials stored in `~/.lingti.yaml`. Credentials saved here are automatically loaded by `gateway` — no flags needed at startup.

```
lingti-bot channels <subcommand>
```

#### channels add

Add or update credentials for a platform channel.

```bash
lingti-bot channels add --channel <name> [credential flags]
```

**Required flag:** `--channel <name>` — the platform to configure.

**Platform credential flags:**

| Channel | Flags |
|---------|-------|
| `telegram` | `--token` |
| `slack` | `--bot-token`, `--app-token` |
| `discord` | `--token` |
| `feishu` | `--app-id`, `--app-secret` |
| `dingtalk` | `--client-id`, `--client-secret` |
| `wecom` | `--corp-id`, `--agent-id`, `--secret`, `--token`, `--aes-key`, `--port` |
| `whatsapp` | `--phone-id`, `--access-token`, `--verify-token` |
| `line` | `--channel-secret`, `--channel-token` |
| `teams` | `--app-id`, `--app-password`, `--tenant-id` |
| `matrix` | `--homeserver-url`, `--user-id`, `--access-token` |
| `mattermost` | `--server-url`, `--token`, `--team-name` |
| `signal` | `--api-url`, `--phone-number` |
| `imessage` | `--bluebubbles-url`, `--bluebubbles-password` |
| `twitch` | `--token`, `--channel-name`, `--bot-name` |
| `nostr` | `--private-key`, `--relays` |
| `zalo` | `--app-id`, `--secret-key`, `--access-token` |
| `nextcloud` | `--server-url`, `--username`, `--password`, `--room-token` |
| `googlechat` | `--project-id`, `--credentials-file` |
| `webapp` | `--port`, `--auth-token` |

**Examples:**

```bash
# Add Telegram
lingti-bot channels add --channel telegram --token 123456:ABC-xxx

# Add Slack
lingti-bot channels add --channel slack \
  --bot-token xoxb-xxx \
  --app-token xapp-xxx

# Add Discord
lingti-bot channels add --channel discord --token MTxxxxxxxx

# Add WeCom
lingti-bot channels add --channel wecom \
  --corp-id CORP_ID \
  --agent-id AGENT_ID \
  --secret SECRET \
  --token TOKEN \
  --aes-key AES_KEY \
  --port 8080

# Enable built-in web chat UI on port 8080
lingti-bot channels add --channel webapp --port 8080
```

#### channels list

Show a table of all platforms with their configuration status.

```bash
lingti-bot channels list
```

**Output:**

```
CHANNEL       STATUS  DETAIL
-------       ------  ------
telegram      ✓       123456:ABC-xxx
slack         ✓       xoxb-xxx...
discord       ✗
feishu        ✗
...
```

#### channels remove

Remove all credentials for a platform from `~/.lingti.yaml`.

```bash
lingti-bot channels remove --channel <name>
```

**Example:**

```bash
lingti-bot channels remove --channel telegram
```

---

### agents

Manage named agents and their routing bindings. Each agent has its own isolated workspace, AI model, and instructions. Routing bindings control which agent handles messages from which platform or channel.

```
lingti-bot agents <subcommand>
```

#### agents add

Add a new agent to `~/.lingti.yaml`. Creates the workspace directory if it doesn't exist.

```bash
lingti-bot agents add <id> [flags]
```

**Arguments:** `<id>` — unique agent identifier (required, positional).

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--workspace <dir>` | `~/.lingti/agents/<id>` | Working directory for this agent |
| `--model <id>` | _(inherited)_ | Model override (e.g. `claude-opus-4-6`) |
| `--provider <name>` | _(inherited)_ | AI provider override |
| `--api-key <key>` | _(inherited)_ | API key override |
| `--instructions <text-or-path>` | | Inline instructions text, or path to a file |
| `--default` | `false` | Mark as the default agent |
| `--allow-tools <list>` | | Comma-separated tool whitelist (empty = allow all) |
| `--deny-tools <list>` | | Comma-separated tool blacklist |

**Examples:**

```bash
# Add a default agent using inherited AI settings
lingti-bot agents add main --default

# Add a work agent with a specific model and custom instructions
lingti-bot agents add work \
  --model claude-opus-4-6 \
  --instructions "You are a focused work assistant. Be concise."

# Add an agent that reads instructions from a file
lingti-bot agents add support \
  --instructions ~/.lingti/support-instructions.txt \
  --workspace ~/my-support-workspace

# Add a restricted agent that can only use read-only tools
lingti-bot agents add readonly \
  --allow-tools "read,glob,grep,shell_read"

# Add an agent that cannot write or edit files
lingti-bot agents add safe \
  --deny-tools "write,edit,shell"
```

#### agents list

List all configured agents. Add `--bindings` to also show routing bindings.

```bash
lingti-bot agents list [--bindings]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `-b, --bindings` | Also show routing bindings table |

**Example output:**

```
ID            DEFAULT   MODEL                 PROVIDER              WORKSPACE
--            -------   -----                 --------              ---------
main          ✓         (inherited)           (inherited)           /Users/alex/.lingti/agents/main
work                    claude-opus-4-6       (inherited)           /Users/alex/.lingti/agents/work

AGENT         PLATFORM      CHANNEL_ID
-----         --------      ----------
main          telegram
work          slack         C_WORK_CHANNEL
```

#### agents bind

Add routing bindings that direct messages from a platform (or specific channel) to a named agent.

```bash
lingti-bot agents bind [--agent <id>] --bind <spec> [--bind <spec> ...]
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent <id>` | Agent ID (defaults to the config default agent) |
| `--bind <spec>` | Binding spec: `<platform>` or `<platform>:<channelID>` (repeatable) |

**Binding spec format:**

- `telegram` — all messages from Telegram
- `slack:C_WORK` — only messages from the Slack channel with ID `C_WORK`

**Examples:**

```bash
# Bind all Telegram traffic to the default agent
lingti-bot agents bind --bind telegram

# Bind a specific Slack channel to the 'work' agent
lingti-bot agents bind --agent work --bind slack:C_WORK_CHANNEL

# Bind multiple platforms to one agent at once
lingti-bot agents bind --agent work \
  --bind slack:C_WORK \
  --bind discord:1234567890
```

#### agents unbind

Remove routing bindings for an agent.

```bash
lingti-bot agents unbind [--agent <id>] (--bind <spec> [...] | --all)
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--agent <id>` | Agent ID (defaults to the config default agent) |
| `--bind <spec>` | Specific binding to remove (repeatable) |
| `--all` | Remove all bindings for this agent |

**Examples:**

```bash
# Remove a specific binding
lingti-bot agents unbind --agent work --bind slack:C_WORK

# Remove all bindings for an agent
lingti-bot agents unbind --agent work --all
```

---

### gateway

The unified run command. Starts all configured platform bots and the WebSocket server in a single process.

**What gateway does:**
1. Reads `~/.lingti.yaml` for platform credentials and agent definitions
2. Registers all configured platforms (Telegram, Slack, Discord, etc.)
3. Starts the WebSocket server on `:18789` (unless `--no-ws`)
4. Routes incoming messages to the correct agent based on bindings
5. Writes `~/.lingti/gateway.pid` for use by `gateway restart`

```bash
lingti-bot gateway [flags]
lingti-bot gateway restart
```

**Flags:**

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--addr` | `GATEWAY_ADDR` | `:18789` | WebSocket server address |
| `--auth-token` | `GATEWAY_AUTH_TOKEN` | | Single WebSocket auth token |
| `--auth-tokens` | `GATEWAY_AUTH_TOKENS` | | Comma-separated WebSocket auth tokens |
| `--no-ws` | | `false` | Disable WebSocket server (platform bots only) |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | | Model name |
| `--instructions` | | | Path to custom instructions file |
| `--call-timeout` | `AI_CALL_TIMEOUT` | `90` | AI API call timeout in seconds |
| `--debug-dir` | `BROWSER_DEBUG_DIR` | | Directory for browser debug screenshots |
| `--webapp-port` | `WEBAPP_PORT` | `0` | Web chat UI port (0 = disabled) |

All platform credential flags are also available (`--telegram-token`, `--slack-bot-token`, etc.) as overrides — these take precedence over `~/.lingti.yaml`. See the [Platform Flags](#platform-flags) section.

#### Typical workflow

```bash
# Step 1: Save credentials once
lingti-bot channels add --channel telegram --token 123456:ABC-xxx
lingti-bot channels add --channel slack --bot-token xoxb-... --app-token xapp-...

# Step 2: (Optional) Configure agents and routing
lingti-bot agents add main --default
lingti-bot agents add work --model claude-opus-4-6
lingti-bot agents bind --bind telegram               # → main (default)
lingti-bot agents bind --agent work --bind slack     # → work agent

# Step 3: Start everything
lingti-bot gateway --api-key sk-ant-xxx
```

#### Using flags directly (Docker / CI)

Flags override `~/.lingti.yaml`, so you can run without a config file:

```bash
lingti-bot gateway \
  --api-key sk-ant-xxx \
  --telegram-token 123456:ABC-xxx \
  --slack-bot-token xoxb-xxx \
  --slack-app-token xapp-xxx
```

Or via environment variables:

```bash
export AI_API_KEY=sk-ant-xxx
export TELEGRAM_BOT_TOKEN=123456:ABC-xxx
lingti-bot gateway
```

#### Disable WebSocket server

```bash
# Platform bots only — no WebSocket API
lingti-bot gateway --no-ws --api-key sk-ant-xxx

# Or via env var (empty string disables it)
GATEWAY_ADDR="" lingti-bot gateway --api-key sk-ant-xxx
```

#### Web chat UI

```bash
# Serve web chat UI on port 8080 in addition to everything else
lingti-bot gateway --webapp-port 8080 --api-key sk-ant-xxx
```

#### Reload config without restart

After saving changes to `~/.lingti.yaml` (e.g. adding a channel), signal the running gateway to reload:

```bash
lingti-bot gateway restart
```

This sends `SIGHUP` to the running gateway process via `~/.lingti/gateway.pid`.

#### gateway restart

```bash
lingti-bot gateway restart
```

Reads `~/.lingti/gateway.pid`, sends `SIGHUP` to the process. The gateway logs `[Gateway] Received SIGHUP, reloading config...` and re-reads its configuration.

---

### Platform Flags

All platform credentials can be passed as flags to `gateway` for one-off overrides or Docker deployments. They follow the same 3-tier priority: **CLI flag > environment variable > `~/.lingti.yaml`**.

| Platform | Flag | Env Var |
|----------|------|---------|
| Slack | `--slack-bot-token` | `SLACK_BOT_TOKEN` |
| Slack | `--slack-app-token` | `SLACK_APP_TOKEN` |
| Telegram | `--telegram-token` | `TELEGRAM_BOT_TOKEN` |
| Discord | `--discord-token` | `DISCORD_BOT_TOKEN` |
| Feishu | `--feishu-app-id` | `FEISHU_APP_ID` |
| Feishu | `--feishu-app-secret` | `FEISHU_APP_SECRET` |
| DingTalk | `--dingtalk-client-id` | `DINGTALK_CLIENT_ID` |
| DingTalk | `--dingtalk-client-secret` | `DINGTALK_CLIENT_SECRET` |
| WeCom | `--wecom-corp-id` | `WECOM_CORP_ID` |
| WeCom | `--wecom-agent-id` | `WECOM_AGENT_ID` |
| WeCom | `--wecom-secret` | `WECOM_SECRET` |
| WeCom | `--wecom-token` | `WECOM_TOKEN` |
| WeCom | `--wecom-aes-key` | `WECOM_AES_KEY` |
| WeCom | `--wecom-port` | `WECOM_PORT` |
| WhatsApp | `--whatsapp-phone-id` | `WHATSAPP_PHONE_NUMBER_ID` |
| WhatsApp | `--whatsapp-access-token` | `WHATSAPP_ACCESS_TOKEN` |
| WhatsApp | `--whatsapp-verify-token` | `WHATSAPP_VERIFY_TOKEN` |
| LINE | `--line-channel-secret` | `LINE_CHANNEL_SECRET` |
| LINE | `--line-channel-token` | `LINE_CHANNEL_TOKEN` |
| Teams | `--teams-app-id` | `TEAMS_APP_ID` |
| Teams | `--teams-app-password` | `TEAMS_APP_PASSWORD` |
| Teams | `--teams-tenant-id` | `TEAMS_TENANT_ID` |
| Matrix | `--matrix-homeserver-url` | `MATRIX_HOMESERVER_URL` |
| Matrix | `--matrix-user-id` | `MATRIX_USER_ID` |
| Matrix | `--matrix-access-token` | `MATRIX_ACCESS_TOKEN` |
| Mattermost | `--mattermost-server-url` | `MATTERMOST_SERVER_URL` |
| Mattermost | `--mattermost-token` | `MATTERMOST_TOKEN` |
| Mattermost | `--mattermost-team-name` | `MATTERMOST_TEAM_NAME` |
| Signal | `--signal-api-url` | `SIGNAL_API_URL` |
| Signal | `--signal-phone-number` | `SIGNAL_PHONE_NUMBER` |
| iMessage | `--bluebubbles-url` | `BLUEBUBBLES_URL` |
| iMessage | `--bluebubbles-password` | `BLUEBUBBLES_PASSWORD` |
| Twitch | `--twitch-token` | `TWITCH_TOKEN` |
| Twitch | `--twitch-channel` | `TWITCH_CHANNEL` |
| Twitch | `--twitch-bot-name` | `TWITCH_BOT_NAME` |
| NOSTR | `--nostr-private-key` | `NOSTR_PRIVATE_KEY` |
| NOSTR | `--nostr-relays` | `NOSTR_RELAYS` |
| Zalo | `--zalo-app-id` | `ZALO_APP_ID` |
| Zalo | `--zalo-secret-key` | `ZALO_SECRET_KEY` |
| Zalo | `--zalo-access-token` | `ZALO_ACCESS_TOKEN` |
| Nextcloud | `--nextcloud-server-url` | `NEXTCLOUD_SERVER_URL` |
| Nextcloud | `--nextcloud-username` | `NEXTCLOUD_USERNAME` |
| Nextcloud | `--nextcloud-password` | `NEXTCLOUD_PASSWORD` |
| Nextcloud | `--nextcloud-room-token` | `NEXTCLOUD_ROOM_TOKEN` |
| Google Chat | `--googlechat-project-id` | `GOOGLE_CHAT_PROJECT_ID` |
| Google Chat | `--googlechat-credentials-file` | `GOOGLE_CHAT_CREDENTIALS_FILE` |
| Webapp | `--webapp-port` | `WEBAPP_PORT` |

---

### serve

Start the MCP (Model Context Protocol) server for integration with Claude Desktop, Cursor, and other MCP clients.

```bash
lingti-bot serve
```

**Configuration for Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "lingti-bot": {
      "command": "/path/to/lingti-bot",
      "args": ["serve"]
    }
  }
}
```

---

### relay

Connect to the lingti-bot cloud relay service — no public server needed. See [Gateway vs Relay](gateway-vs-relay.md).

```bash
lingti-bot relay [flags]
```

**Supported platforms:** feishu, slack, wechat, wecom

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--platform` | `RELAY_PLATFORM` | | Platform: feishu, slack, wechat, wecom (required) |
| `--user-id` | `RELAY_USER_ID` | | Your user ID from `/whoami` |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--server` | `RELAY_SERVER_URL` | `wss://bot.lingti.com/ws` | Relay server URL |

---

### doctor

Run diagnostic checks on configuration, credentials, connectivity, and required tools.

```bash
lingti-bot doctor
```

**Checks:** config file, AI API key, AI connectivity, platform credentials, required binaries, browser CDP, MCP servers, temp directory.

---

### skills

Manage modular skill extensions.

```bash
lingti-bot skills list
lingti-bot skills install <name>
lingti-bot skills remove <name>
```

---

### version

Show version information.

```bash
lingti-bot version
```

---

## router (deprecated)

> ⚠️ **`router` is deprecated.** The command is still present but prints a migration guide and exits. Use `gateway` instead — it includes all router functionality plus the WebSocket server.

Running `lingti-bot gateway` (with or without flags) prints:

```
WARNING: 'router' is deprecated and will be removed in a future release.
Use the new 'channels', 'agents', and 'gateway' commands instead:

  # 1. Save your platform credentials to ~/.lingti.yaml
  lingti-bot channels add --channel telegram --token YOUR_BOT_TOKEN
  lingti-bot channels add --channel slack --bot-token xoxb-... --app-token xapp-...

  # 2. Add an agent (optional — skip if using env vars / flags only)
  lingti-bot agents add main --default

  # 3. Start everything
  lingti-bot gateway --api-key YOUR_API_KEY

  # Or keep passing flags/env vars directly (all router flags work on gateway):
  lingti-bot gateway --telegram-token YOUR_BOT_TOKEN --api-key YOUR_API_KEY
```

**Migration is straightforward:** every flag that `router` accepted (`--telegram-token`, `--api-key`, `--provider`, `--webapp-port`, etc.) is available on `gateway` with the same name.

---

## Environment Variables

### AI Provider

| Variable | Description |
|----------|-------------|
| `AI_PROVIDER` | Provider name: `claude`, `deepseek`, `kimi`, `qwen`, etc. |
| `AI_API_KEY` | API key |
| `AI_BASE_URL` | Custom base URL |
| `AI_MODEL` | Model name override |
| `AI_CALL_TIMEOUT` | API call timeout in seconds (default: 90) |

**Legacy aliases (also accepted):**

| Variable | Maps to |
|----------|---------|
| `ANTHROPIC_API_KEY` | `AI_API_KEY` |
| `ANTHROPIC_OAUTH_TOKEN` | `AI_API_KEY` |
| `ANTHROPIC_BASE_URL` | `AI_BASE_URL` |
| `ANTHROPIC_MODEL` | `AI_MODEL` |

### Gateway (WebSocket Server)

| Variable | Default | Description |
|----------|---------|-------------|
| `GATEWAY_ADDR` | `:18789` | Listen address |
| `GATEWAY_AUTH_TOKEN` | | Single auth token |
| `GATEWAY_AUTH_TOKENS` | | Comma-separated auth tokens |

### Browser Debug

| Variable | Description |
|----------|-------------|
| `BROWSER_DEBUG` | Set to `1` or `true` to enable debug screenshots |
| `BROWSER_DEBUG_DIR` | Directory for debug screenshots |

### Webapp

| Variable | Description |
|----------|-------------|
| `WEBAPP_PORT` | Web chat UI port (0 = disabled) |

---

## AI Providers

### Claude (Anthropic)

```bash
export AI_PROVIDER=claude
export AI_API_KEY=sk-ant-api03-xxx
export AI_MODEL=claude-sonnet-4-6  # optional
```

### DeepSeek

```bash
export AI_PROVIDER=deepseek
export AI_API_KEY=sk-xxx
```

### Kimi (Moonshot)

```bash
export AI_PROVIDER=kimi
export AI_API_KEY=sk-xxx
export AI_MODEL=moonshot-v1-8k  # optional
```

See [AI-PROVIDERS.md](../AI-PROVIDERS.md) for the full list of 16+ supported providers.

---

## Configuration File

All settings can be stored in `~/.lingti.yaml`. The 3-tier resolution order is:

**CLI flag > environment variable > `~/.lingti.yaml`**

Example configuration with agents and bindings:

```yaml
ai:
  provider: claude
  api_key: sk-ant-xxx
  model: claude-sonnet-4-6

platforms:
  telegram:
    token: "123456:ABC-xxx"
  slack:
    bot_token: xoxb-xxx
    app_token: xapp-xxx

agents:
  - id: main
    default: true
    workspace: ~/.lingti/agents/main

  - id: work
    workspace: ~/.lingti/agents/work
    model: claude-opus-4-6
    instructions: "You are a focused work assistant. Keep answers brief."
    deny_tools: [write, edit, shell]

bindings:
  - agent_id: main
    match:
      platform: telegram
  - agent_id: work
    match:
      platform: slack
      channel_id: C_WORK_CHANNEL
```

See [CONFIGURATION.md](../CONFIGURATION.md) for the complete config file reference.

---

## See Also

- [gateway.md](gateway.md) — WebSocket protocol reference
- [CONFIGURATION.md](../CONFIGURATION.md) — Full config file reference
- [chat-platforms.md](chat-platforms.md) — Platform-specific setup guides
- [AI-PROVIDERS.md](../AI-PROVIDERS.md) — All supported AI providers
