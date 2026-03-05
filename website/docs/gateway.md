# Gateway

The **gateway** is the unified run command for lingti-bot. It starts all configured platform bots (Telegram, Slack, Discord, etc.) **and** a WebSocket API server in a single process.

```bash
lingti-bot gateway --api-key sk-ant-xxx
```

On startup the gateway:
1. Loads `~/.lingti.yaml` for platform credentials, agent definitions, and routing bindings
2. Registers all configured platforms as bot listeners
3. Starts the WebSocket server on `:18789` (use `--no-ws` to disable)
4. Routes each incoming message to the correct agent based on bindings
5. Writes `~/.lingti/gateway.pid` so `gateway restart` can reload config without a full restart

## Quick Start

```bash
# 1. Save credentials (once)
lingti-bot channels add --channel telegram --token 123456:ABC-xxx
lingti-bot channels add --channel slack --bot-token xoxb-... --app-token xapp-...

# 2. (Optional) Define agents and routing
lingti-bot agents add main --default
lingti-bot agents bind --bind telegram

# 3. Start
lingti-bot gateway --api-key sk-ant-xxx
```

## Managing Channels

Channels are platform credentials stored in `~/.lingti.yaml`. The `channels` command is the preferred way to manage them — it reads, updates, and writes the config file for you.

```bash
# Add a platform
lingti-bot channels add --channel telegram --token YOUR_TOKEN
lingti-bot channels add --channel slack --bot-token xoxb-... --app-token xapp-...
lingti-bot channels add --channel discord --token MTxxx
lingti-bot channels add --channel webapp --port 8080  # built-in web UI

# List current status
lingti-bot channels list

# Remove a platform
lingti-bot channels remove --channel discord
```

Once saved, credentials are loaded automatically every time you run `gateway` — no flags needed.

## Managing Agents

An **agent** is an isolated AI brain with its own workspace, session memory, model, instructions, and tool permissions. You can have multiple agents and route different platforms or channels to different agents.

### Adding agents

```bash
# Default agent — used when no binding matches
lingti-bot agents add main --default

# A focused work agent with a different model
lingti-bot agents add work \
  --model claude-opus-4-6 \
  --instructions "You are a focused work assistant. Be concise and precise."

# An agent that can only read (no write/edit/shell)
lingti-bot agents add readonly \
  --deny-tools "write,edit,shell"
```

Each agent gets its own workspace directory (`~/.lingti/agents/<id>` by default). Workspace, model, provider, API key, and instructions are all optional — unset fields inherit from the global `ai:` config.

### Routing bindings

Bindings control which agent handles messages from which source. Resolution order (most specific wins):

| Priority | Match | Example |
|----------|-------|---------|
| 1 (highest) | `platform` + `channel_id` | Only messages from `slack:C_WORK` |
| 2 | `platform` only | All messages from `telegram` |
| 3 (lowest) | Default agent | Anything unmatched |

```bash
# Route all Telegram messages to 'main'
lingti-bot agents bind --bind telegram

# Route a specific Slack channel to 'work'
lingti-bot agents bind --agent work --bind slack:C_WORK_CHANNEL

# See agents and their bindings
lingti-bot agents list --bindings

# Remove a binding
lingti-bot agents unbind --agent work --bind slack:C_WORK_CHANNEL

# Remove all bindings for an agent
lingti-bot agents unbind --agent work --all
```

### Agent config in ~/.lingti.yaml

```yaml
agents:
  - id: main
    default: true
    workspace: ~/.lingti/agents/main

  - id: work
    workspace: ~/my-work-workspace
    model: claude-opus-4-6
    instructions: "You are a focused work assistant."
    deny_tools: [write, edit, shell]

  - id: readonly
    allow_tools: [read, glob, grep]

bindings:
  - agent_id: main
    match:
      platform: telegram
  - agent_id: work
    match:
      platform: slack
      channel_id: C_WORK_CHANNEL
```

`allow_tools` non-empty = whitelist (only those tools available). `deny_tools` = blacklist (those tools removed from the full set).

## Gateway Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--addr` | `GATEWAY_ADDR` | `:18789` | WebSocket listen address |
| `--auth-token` | `GATEWAY_AUTH_TOKEN` | | Single auth token for WebSocket clients |
| `--auth-tokens` | `GATEWAY_AUTH_TOKENS` | | Comma-separated auth tokens |
| `--no-ws` | | `false` | Disable WebSocket server |
| `--provider` | `AI_PROVIDER` | `claude` | AI provider |
| `--api-key` | `AI_API_KEY` | | AI API key (required) |
| `--base-url` | `AI_BASE_URL` | | Custom AI API base URL |
| `--model` | `AI_MODEL` | | Model name |
| `--instructions` | | | Path to custom instructions file |
| `--call-timeout` | `AI_CALL_TIMEOUT` | `90` | AI API call timeout (seconds) |
| `--webapp-port` | `WEBAPP_PORT` | `0` | Web chat UI port (0 = disabled) |
| `--debug-dir` | `BROWSER_DEBUG_DIR` | | Browser debug screenshot directory |

All platform credential flags are also accepted (e.g. `--telegram-token`, `--slack-bot-token`) as one-time overrides. See the [CLI Reference](cli-reference.md#platform-flags) for the full table.

## Disabling the WebSocket Server

The WebSocket server starts by default. Disable it if you only want platform bots:

```bash
lingti-bot gateway --no-ws --api-key sk-ant-xxx
```

## Reloading Config

After changing `~/.lingti.yaml` (e.g. adding a channel or agent), reload the running gateway without restarting it:

```bash
lingti-bot gateway restart
```

This sends `SIGHUP` to the running process via `~/.lingti/gateway.pid`.

## Docker / CI (Flags Only)

For deployments without a config file, all credentials can be supplied via flags or environment variables:

```bash
# Flags
lingti-bot gateway \
  --api-key sk-ant-xxx \
  --telegram-token 123456:ABC-xxx

# Environment variables
export AI_API_KEY=sk-ant-xxx
export TELEGRAM_BOT_TOKEN=123456:ABC-xxx
lingti-bot gateway

# docker-compose.yml
services:
  lingti-bot:
    image: lingti-bot
    environment:
      AI_API_KEY: ${AI_API_KEY}
      TELEGRAM_BOT_TOKEN: ${TELEGRAM_BOT_TOKEN}
    command: gateway
```

---

## WebSocket API

The gateway also exposes a WebSocket API on `:18789` for custom clients (web UIs, scripts, mobile apps). This is independent of platform bots — both run concurrently.

### Authentication

By default all WebSocket connections are accepted. To require a token:

```bash
lingti-bot gateway --auth-token my-secret --api-key sk-ant-xxx

# Multiple tokens (each person gets their own)
lingti-bot gateway --auth-tokens "alice-token,bob-token" --api-key sk-ant-xxx
```

When auth is enabled, clients must send an `auth` message before chatting.

### HTTP endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Returns `{"status":"ok"}` |
| `GET` | `/status` | Running status, client count, auth state |
| `GET` | `/ws` | WebSocket upgrade endpoint |

```bash
curl http://localhost:18789/health
curl http://localhost:18789/status
```

### Message protocol

All WebSocket messages are JSON with this envelope:

```json
{
  "id":        "unique-message-id",
  "type":      "message-type",
  "payload":   { ... },
  "timestamp": 1700000000000
}
```

#### Client → Server

| Type | Description |
|------|-------------|
| `ping` | Keep-alive; server replies with `pong` |
| `auth` | Authenticate: `{"payload": {"token": "..."}}` |
| `chat` | Send message: `{"payload": {"text": "...", "session_id": "optional"}}` |
| `command` | Built-in command: `{"payload": {"command": "status"}}` or `"clear"` |

#### Server → Client

| Type | Description |
|------|-------------|
| `pong` | Reply to `ping` |
| `auth_result` | Auth outcome: `{"payload": {"success": true}}` |
| `response` | AI reply: `{"payload": {"text": "...", "session_id": "...", "done": true}}` |
| `event` | Command result |
| `error` | Error: `{"payload": {"code": "unauthorized", "message": "..."}}` |

**Error codes:** `unauthorized`, `invalid_message`, `invalid_payload`, `handler_error`, `no_handler`, `unknown_type`, `unknown_command`

### Example clients

**JavaScript:**

```js
const ws = new WebSocket("ws://localhost:18789/ws");

ws.onopen = () => {
  // Skip if no auth configured
  ws.send(JSON.stringify({ type: "auth", payload: { token: "my-secret" } }));
};

ws.onmessage = ({ data }) => {
  const msg = JSON.parse(data);
  if (msg.type === "auth_result" && msg.payload.success) {
    ws.send(JSON.stringify({
      id: "req-1",
      type: "chat",
      payload: { text: "Hello, what can you do?" }
    }));
  }
  if (msg.type === "response" && msg.payload.done) {
    console.log("AI:", msg.payload.text);
  }
};
```

**Python:**

```python
import json, websocket

ws = websocket.create_connection("ws://localhost:18789/ws")
ws.send(json.dumps({"type": "auth", "payload": {"token": "my-secret"}}))
json.loads(ws.recv())  # auth_result

ws.send(json.dumps({"id": "1", "type": "chat", "payload": {"text": "Hello"}}))
print(json.loads(ws.recv())["payload"]["text"])
ws.close()
```

### Sessions

Each WebSocket connection has one active session. The session ID is established on the first `chat` message:
- Provide `session_id` in the payload to use a specific session
- Omit it to use the connection's client ID as the session

Send `{"type": "command", "payload": {"command": "clear"}}` to reset the session and start a fresh conversation.

### Connection lifecycle

```
Client                          Gateway
  |--- WebSocket upgrade -------->|
  |<-- connection accepted -------|
  |--- auth (if required) ------->|
  |<-- auth_result ---------------|
  |--- chat {"text": "Hi"} ------>|
  |<-- response {"done": true} ---|
  |--- command "clear" ---------->|
  |<-- event "cleared" ----------|
  |--- [disconnect] ------------->|
```

The server sends WebSocket-level Ping frames every 30 seconds. Connections that don't respond with Pong within 60 seconds are closed.
