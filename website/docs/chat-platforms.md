# Supported Chat Platforms / 支持的聊天平台

lingti-bot 支持 **19 种聊天平台**，涵盖国内外主流 IM、社交和协作平台。所有平台均通过 `lingti-bot onboard` 交互式向导配置，也可通过命令行参数或环境变量指定。

lingti-bot supports **19 chat platforms** covering mainstream IM, social, and collaboration platforms globally. Configure via `lingti-bot onboard` interactive wizard, or specify via CLI flags and environment variables.

> **Tip**: Not sure which mode to use? See [Gateway vs Relay](gateway-vs-relay.md) for a detailed comparison.

## Platform List / 平台列表

| # | Platform | 名称 | Connection / 连接方式 | Setup / 接入方式 |
|---|----------|------|-----------------------|------------------|
| 1 | `wecom` | WeCom / 企业微信 | Callback API | Cloud Relay / Self-hosted 云中继或自建 |
| 2 | `wechat` | WeChat Official / 微信公众号 | Cloud Relay 云中继 | Relay only 仅云中继 |
| 3 | `dingtalk` | DingTalk / 钉钉 | Stream Mode | One-click 一键接入 |
| 4 | `feishu` | Feishu / Lark / 飞书 | WebSocket | One-click 一键接入 |
| 5 | `slack` | Slack | Socket Mode | One-click 一键接入 |
| 6 | `telegram` | Telegram | Bot API (long polling) | One-click 一键接入 |
| 7 | `discord` | Discord | Gateway (WebSocket) | One-click 一键接入 |
| 8 | `whatsapp` | WhatsApp Business | Webhook + Graph API | Self-hosted 自建 |
| 9 | `line` | LINE | Webhook + Push API | Self-hosted 自建 |
| 10 | `teams` | Microsoft Teams | Bot Framework + OAuth2 | Self-hosted 自建 |
| 11 | `matrix` | Matrix / Element | HTTP Sync Polling | Self-hosted 自建 |
| 12 | `googlechat` | Google Chat | Webhook + REST API | Self-hosted 自建 |
| 13 | `mattermost` | Mattermost | WebSocket + REST API | Self-hosted 自建 |
| 14 | `imessage` | iMessage (BlueBubbles) | HTTP Polling | Self-hosted 自建 |
| 15 | `signal` | Signal (signal-cli) | HTTP Polling | Self-hosted 自建 |
| 16 | `twitch` | Twitch | IRC | Self-hosted 自建 |
| 17 | `nostr` | NOSTR | WebSocket (Relays) | Self-hosted 自建 |
| 18 | `zalo` | Zalo | Webhook + REST API | Self-hosted 自建 |
| 19 | `nextcloud` | Nextcloud Talk | HTTP Polling + REST | Self-hosted 自建 |

## Configuration / 配置详情

### 1. WeCom / 企业微信

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Corp ID | `--wecom-corp-id` | `WECOM_CORP_ID` | Corporation ID / 企业 ID |
| Agent ID | `--wecom-agent-id` | `WECOM_AGENT_ID` | Agent ID / 应用 ID |
| Secret | `--wecom-secret` | `WECOM_SECRET` | Agent Secret / 应用密钥 |
| Token | `--wecom-token` | `WECOM_TOKEN` | Callback Token / 回调 Token |
| AES Key | `--wecom-aes-key` | `WECOM_AES_KEY` | EncodingAESKey / 消息加密密钥 |
| Port | `--wecom-port` | `WECOM_PORT` | Callback port (default: 8080) / 回调端口 |

> Guide / 教程: [WeCom Integration / 企业微信集成指南](wecom-integration.md)

### 2. WeChat Official / 微信公众号

WeChat Official Account uses Cloud Relay mode only. Configure via `lingti-bot onboard` and select `wechat`.

微信公众号仅支持云中继模式。通过 `lingti-bot onboard` 向导选择 `wechat` 即可。

> Guide / 教程: [WeChat Integration / 微信公众号接入指南](wechat-integration.md)

### 3. DingTalk / 钉钉

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Client ID | `--dingtalk-client-id` | `DINGTALK_CLIENT_ID` | AppKey from Developer Console / 开发者后台 AppKey |
| Client Secret | `--dingtalk-client-secret` | `DINGTALK_CLIENT_SECRET` | AppSecret / 应用密钥 |

### 4. Feishu / Lark / 飞书

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| App ID | `--feishu-app-id` | `FEISHU_APP_ID` | App ID (cli_...) |
| App Secret | `--feishu-app-secret` | `FEISHU_APP_SECRET` | App Secret / 应用密钥 |

> Guide / 教程: [Feishu Integration / 飞书集成指南](feishu-integration.md)

### 5. Slack

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Bot Token | `--slack-bot-token` | `SLACK_BOT_TOKEN` | Bot Token (xoxb-...) |
| App Token | `--slack-app-token` | `SLACK_APP_TOKEN` | App Token (xapp-...) |

> Guide / 教程: [Slack Integration / Slack 集成指南](slack-integration.md)

### 6. Telegram

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Bot Token | `--telegram-token` | `TELEGRAM_BOT_TOKEN` | Bot token from @BotFather |

### 7. Discord

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Bot Token | `--discord-token` | `DISCORD_BOT_TOKEN` | Bot token from Developer Portal |

### 8. WhatsApp Business

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Phone Number ID | `--whatsapp-phone-id` | `WHATSAPP_PHONE_NUMBER_ID` | WhatsApp Business Phone Number ID |
| Access Token | `--whatsapp-access-token` | `WHATSAPP_ACCESS_TOKEN` | Meta Graph API access token |
| Verify Token | `--whatsapp-verify-token` | `WHATSAPP_VERIFY_TOKEN` | Webhook verification token |

### 9. LINE

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Channel Secret | `--line-channel-secret` | `LINE_CHANNEL_SECRET` | LINE Channel Secret |
| Channel Token | `--line-channel-token` | `LINE_CHANNEL_TOKEN` | LINE Channel Access Token |

### 10. Microsoft Teams

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| App ID | `--teams-app-id` | `TEAMS_APP_ID` | Teams App ID |
| App Password | `--teams-app-password` | `TEAMS_APP_PASSWORD` | Teams App Password |
| Tenant ID | `--teams-tenant-id` | `TEAMS_TENANT_ID` | Azure Tenant ID |

### 11. Matrix / Element

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Homeserver URL | `--matrix-homeserver-url` | `MATRIX_HOMESERVER_URL` | e.g. https://matrix.org |
| User ID | `--matrix-user-id` | `MATRIX_USER_ID` | e.g. @bot:matrix.org |
| Access Token | `--matrix-access-token` | `MATRIX_ACCESS_TOKEN` | Matrix access token |

### 12. Google Chat

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Project ID | `--googlechat-project-id` | `GOOGLE_CHAT_PROJECT_ID` | Google Cloud project ID |
| Credentials File | `--googlechat-credentials-file` | `GOOGLE_CHAT_CREDENTIALS_FILE` | Service account JSON path / 服务账号 JSON 路径 |

### 13. Mattermost

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Server URL | `--mattermost-server-url` | `MATTERMOST_SERVER_URL` | Mattermost server URL |
| Token | `--mattermost-token` | `MATTERMOST_TOKEN` | Personal access token |
| Team Name | `--mattermost-team-name` | `MATTERMOST_TEAM_NAME` | Team name / 团队名称 |

### 14. iMessage (BlueBubbles)

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Server URL | `--bluebubbles-url` | `BLUEBUBBLES_URL` | BlueBubbles server URL |
| Password | `--bluebubbles-password` | `BLUEBUBBLES_PASSWORD` | BlueBubbles server password |

> Requires [BlueBubbles](https://bluebubbles.app/) server running on macOS.
> 需要在 macOS 上运行 [BlueBubbles](https://bluebubbles.app/) 服务器。

### 15. Signal

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| API URL | `--signal-api-url` | `SIGNAL_API_URL` | signal-cli REST API URL |
| Phone Number | `--signal-phone-number` | `SIGNAL_PHONE_NUMBER` | Registered phone number / 注册手机号 |

> Requires [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) running.
> 需要运行 [signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) 服务。

### 16. Twitch

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| OAuth Token | `--twitch-token` | `TWITCH_TOKEN` | Twitch OAuth token (oauth:xxx) |
| Channel | `--twitch-channel` | `TWITCH_CHANNEL` | Channel name / 频道名 |
| Bot Name | `--twitch-bot-name` | `TWITCH_BOT_NAME` | Bot username / 机器人用户名 |

### 17. NOSTR

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Private Key | `--nostr-private-key` | `NOSTR_PRIVATE_KEY` | Private key (hex or nsec) / 私钥 |
| Relays | `--nostr-relays` | `NOSTR_RELAYS` | Comma-separated relay URLs / 中继地址（逗号分隔） |

### 18. Zalo

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| App ID | `--zalo-app-id` | `ZALO_APP_ID` | Zalo App ID |
| Secret Key | `--zalo-secret-key` | `ZALO_SECRET_KEY` | Zalo Secret Key |
| Access Token | `--zalo-access-token` | `ZALO_ACCESS_TOKEN` | Zalo Access Token |

### 19. Nextcloud Talk

| Field / 字段 | Flag | Env / 环境变量 | Description / 说明 |
|---------------|------|----------------|---------------------|
| Server URL | `--nextcloud-server-url` | `NEXTCLOUD_SERVER_URL` | Nextcloud server URL |
| Username | `--nextcloud-username` | `NEXTCLOUD_USERNAME` | Bot username / 机器人用户名 |
| Password | `--nextcloud-password` | `NEXTCLOUD_PASSWORD` | Password or app password / 密码或应用密码 |
| Room Token | `--nextcloud-room-token` | `NEXTCLOUD_ROOM_TOKEN` | Talk room token / 房间 Token |

## Usage / 用法

```bash
# Interactive wizard / 交互式向导
lingti-bot onboard

# Command line examples / 命令行示例
lingti-bot gateway --provider deepseek --api-key sk-xxx \
  --slack-bot-token xoxb-... --slack-app-token xapp-...

lingti-bot gateway --provider deepseek --api-key sk-xxx \
  --telegram-token 123456:ABC-DEF

lingti-bot relay --platform wecom --provider deepseek --api-key sk-xxx

# Environment variables / 环境变量
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF"
lingti-bot gateway --provider deepseek --api-key sk-xxx
```

## Notes / 说明

- Multiple platforms can run simultaneously via `lingti-bot gateway`. Each platform with valid credentials will be registered automatically.
- 多个平台可通过 `lingti-bot gateway` 同时运行。提供了有效凭证的平台会自动注册。
- Cloud Relay (`lingti-bot relay`) is the easiest way to connect WeCom and WeChat Official Account — no public server needed.
- 云中继（`lingti-bot relay`）是接入企业微信和微信公众号最简单的方式 — 无需公网服务器。
- All platform credentials can be saved via `lingti-bot onboard` and stored in `~/Library/Preferences/Lingti/bot.yaml` (macOS) or `~/.config/lingti/bot.yaml` (Linux).
- 所有平台凭证可通过 `lingti-bot onboard` 保存到 `~/Library/Preferences/Lingti/bot.yaml`（macOS）或 `~/.config/lingti/bot.yaml`（Linux）。
