# Slack Integration

This guide explains how to set up Slack integration for lingti-bot's message router.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    lingti-bot Router                     │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │    Router    │  │    Agent     │  │  MCP Tools   │   │
│  │  (messages)  │──│  (Claude)    │──│  (actions)   │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────┐
│     Slack       │
│  (Socket Mode)  │
└─────────────────┘
```

## Prerequisites

- A Slack workspace where you have permission to install apps
- An Anthropic API key for Claude
- lingti-bot binary built and available

## Step 1: Create a Slack App

1. Go to [Slack API Apps](https://api.slack.com/apps)
2. Click **"Create New App"**
3. Select **"From scratch"**
4. Enter App Name: `lingti-bot`
5. Select your workspace
6. Click **"Create App"**

## Step 2: Enable Socket Mode

Socket Mode allows the bot to receive events without exposing a public URL.

1. In your app settings, go to **"Socket Mode"** in the left sidebar
2. Toggle **"Enable Socket Mode"** to ON
3. You'll be prompted to create an App-Level Token:
   - Token Name: `socket-token`
   - Scopes: `connections:write`
4. Click **"Generate"**
5. **Save the App-Level Token** (`xapp-...`) - you'll need this later

## Step 3: Configure Bot Scopes

1. Go to **"OAuth & Permissions"** in the left sidebar
2. Scroll to **"Scopes"** section
3. Under **"Bot Token Scopes"**, add:

| Scope | Description |
|-------|-------------|
| `app_mentions:read` | View messages that mention the bot |
| `chat:write` | Send messages as the bot |
| `im:history` | View messages in DMs with the bot |
| `im:read` | View basic DM info |
| `im:write` | Start DMs with the bot |

## Step 4: Enable Event Subscriptions

1. Go to **"Event Subscriptions"** in the left sidebar
2. Toggle **"Enable Events"** to ON
3. Under **"Subscribe to bot events"**, add:

| Event | Description |
|-------|-------------|
| `app_mention` | Triggered when someone @mentions the bot |
| `message.im` | Triggered when someone DMs the bot |

4. Click **"Save Changes"**

## Step 5: Install the App

1. Go to **"Install App"** in the left sidebar
2. Click **"Install to Workspace"**
3. Review permissions and click **"Allow"**
4. **Copy the Bot User OAuth Token** (`xoxb-...`)

## Step 6: Run lingti-bot Router

### Using Environment Variables

```bash
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
export SLACK_APP_TOKEN="xapp-your-app-token"
export ANTHROPIC_API_KEY="sk-ant-your-api-key"
# Optional configuration
export ANTHROPIC_BASE_URL="https://your-proxy.com/v1"  # Custom API base URL
export ANTHROPIC_MODEL="claude-sonnet-4-20250514"       # Specify model

lingti-bot gateway
```

### Using Command-Line Flags

```bash
lingti-bot gateway \
  --slack-bot-token "xoxb-your-bot-token" \
  --slack-app-token "xapp-your-app-token" \
  --api-key "sk-ant-your-api-key" \
  --base-url "https://your-proxy.com/v1" \
  --model "claude-sonnet-4-20250514"
```

### Using a .env File

Create a `.env` file:

```bash
SLACK_BOT_TOKEN=xoxb-your-bot-token
SLACK_APP_TOKEN=xapp-your-app-token
ANTHROPIC_API_KEY=sk-ant-your-api-key
ANTHROPIC_BASE_URL=https://your-proxy.com/v1
ANTHROPIC_MODEL=claude-sonnet-4-20250514
```

Then run:

```bash
source .env && lingti-bot gateway
```

## Step 7: Test the Integration

1. Open Slack
2. Find your bot in the Apps section or DM it directly
3. Send a message like:
   - `@lingti-bot what's on my calendar today?`
   - `@lingti-bot list files in ~/Desktop`
   - `@lingti-bot what's my system info?`

## Available Commands

Once connected, the bot can:

| Category | Examples |
|----------|----------|
| **Calendar** | "What's on my calendar today?", "Schedule a meeting tomorrow at 2pm" |
| **Files** | "List files in ~/Downloads", "Find old files on my Desktop" |
| **System** | "What's my CPU usage?", "Show disk space" |
| **Shell** | "Run `ls -la`", "Check git status" |
| **Process** | "List running processes", "What's using the most memory?" |

## Troubleshooting

### Bot not responding

1. Check that all three tokens are set correctly
2. Verify the bot is running: `lingti-bot gateway`
3. Check logs for errors

### "not_authed" error

Your `SLACK_BOT_TOKEN` is invalid or expired. Reinstall the app and get a new token.

### "invalid_auth" for Socket Mode

Your `SLACK_APP_TOKEN` is invalid. Generate a new App-Level Token in Socket Mode settings.

### Bot responds in wrong channel

The bot only responds to:
- Direct messages (DMs)
- @mentions in channels

Make sure to @mention the bot in channels.

## Security Considerations

- Never commit tokens to version control
- Use environment variables or a secrets manager
- Restrict bot installation to trusted workspaces
- Review bot permissions regularly

## Running as a Service

To run the router as a background service, create a systemd unit or launchd plist:

### macOS (launchd)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.lingti.bot.router</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/lingti-bot</string>
        <string>router</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>SLACK_BOT_TOKEN</key>
        <string>xoxb-...</string>
        <key>SLACK_APP_TOKEN</key>
        <string>xapp-...</string>
        <key>ANTHROPIC_API_KEY</key>
        <string>sk-ant-...</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/lingti-bot-router.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/lingti-bot-router.log</string>
</dict>
</plist>
```

### Linux (systemd)

```ini
[Unit]
Description=Lingti Bot Router
After=network.target

[Service]
Type=simple
Environment=SLACK_BOT_TOKEN=xoxb-...
Environment=SLACK_APP_TOKEN=xapp-...
Environment=ANTHROPIC_API_KEY=sk-ant-...
ExecStart=/usr/local/bin/lingti-bot gateway
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## References

- [Slack API Documentation](https://api.slack.com/docs)
- [Socket Mode Guide](https://api.slack.com/apis/connections/socket)
- [Slack Events API](https://api.slack.com/events-api)
