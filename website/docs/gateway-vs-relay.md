# Gateway vs Relay

lingti-bot provides two modes for connecting to chat platforms: **gateway** and **relay**.

## Quick Comparison

| Feature | `gateway` (Self-hosted) | `relay` (Cloud) |
|---------|-------------------------|-----------------|
| Architecture | Direct connection to platform APIs | Connect to lingti-bot cloud (bot.lingti.com) |
| Server needed | Requires port forwarding / tunnel | No server needed |
| Platforms | All 19 platforms | feishu, slack, wechat, wecom |
| Data flow | User → Platform → Your PC → AI | User → Platform → Cloud → Your PC → AI |

## When to use `gateway`

Use `gateway` when:
- You want full control over the connection
- The platform is not supported by relay
- You need to run multiple platforms simultaneously
- You have a server with public IP or use a tunnel (frp, ngrok, Cloudflare Tunnel)

```bash
# Run gateway with any supported platform
lingti-bot gateway --provider deepseek --api-key sk-xxx \
  --telegram-token 123456:ABC-DEF

lingti-bot gateway --provider claude --api-key sk-ant-xxx \
  --dingtalk-client-id xxx --dingtalk-client-secret xxx
```

All 19 platforms are supported in gateway mode.

## When to use `relay`

Use `relay` when:
- You don't have a server with public IP
- You want the easiest setup (no firewall, no tunnel)
- The platform is supported: feishu, slack, wechat, wecom

```bash
# Connect to cloud relay - no public server needed
lingti-bot relay --platform wecom --user-id your-id \
  --provider deepseek --api-key sk-xxx
```

The cloud relay handles the platform connection for you. Your AI runs locally.

## Platform Support by Mode

| Platform | gateway | relay |
|----------|---------|-------|
| DingTalk (钉钉) | ✅ Stream Mode | ❌ Not needed* |
| Feishu (飞书) | ✅ WebSocket | ✅ |
| Slack | ✅ Socket Mode | ✅ |
| WeCom (企业微信) | ✅ Callback API | ✅ |
| WeChat (微信公众号) | ❌ | ✅ Cloud only |
| Telegram | ✅ | ❌ |
| Discord | ✅ | ❌ |
| WhatsApp | ✅ | ❌ |
| LINE | ✅ | ❌ |
| Teams | ✅ | ❌ |
| Others... | ✅ | ❌ |

\* *DingTalk uses Stream Mode (WebSocket) which is inherently "serverless" — same benefit as cloud relay, so relay is not needed.*

## Why DingTalk has no relay

DingTalk already supports **Stream Mode**, a native WebSocket connection that works without a public server:

```
DingTalk Server ←WebSocket→ lingti-bot (running locally)
```

This achieves the same goal as cloud relay — no need to buy a server or configure a tunnel. Therefore, DingTalk doesn't need relay support.

## How each mode works

### Gateway (Self-hosted)

```
User → Platform API → lingti-bot gateway (your machine) → AI
                    ↑                              ↓
              Direct connection              Response back
```

You run `lingti-bot gateway` locally. The bot connects directly to the platform's API.

### Relay (Cloud)

```
User → Platform → lingti-bot Cloud (bot.lingti.com) ←WebSocket→ lingti-bot relay (your machine) → AI
```

You run `lingti-bot relay` locally. The cloud server handles the platform connection and forwards messages to your local instance via WebSocket.

## Summary

- **gateway**: Full control, all platforms, may need a server/tunnel
- **relay**: Easiest setup for supported platforms, no server needed
- **DingTalk**: Uses Stream Mode = no server needed = gateway only
