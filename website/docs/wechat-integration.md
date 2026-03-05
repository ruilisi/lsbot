# 微信公众号一键接入指南

通过微信公众号「灵缇小秘」，无需服务器、无需公网 IP，即可将 AI Bot 接入微信。

## 隐私说明

> 云中继平台仅作为消息路由网关，**不保存任何用户数据**。所有对话历史、上下文记忆均存储在您的本地电脑上。详见 [隐私声明](为何更适合中国宝宝体质.md#隐私声明)。

## 接入流程

### 第一步：关注公众号获取 User ID

1. 微信搜索公众号：**灵缇小秘**
2. 关注后发送任意消息
3. 公众号将返回您的专属 `user-id`

<img src="https://lingti-1302055788.cos.ap-guangzhou.myqcloud.com/lingti-bot-wechat.png" alt="微信二维码" width="300">

### 第二步：安装 lingti-bot

**macOS / Linux / WSL:**
```bash
curl -fsSL https://files.lingti.com/install-bot.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot
```

### 第三步：配置并启动

使用获取到的 `user-id` 启动云中继模式：

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --model MiniMax-M2.1 \
  --api-key YOUR_API_KEY \
  --base-url "https://api.minimaxi.com/anthropic/v1"
```

**参数说明：**

| 参数 | 说明 | 示例 |
|------|------|------|
| `--user-id` | 从公众号获取的用户 ID | `wx_abc123` |
| `--platform` | 平台类型 | `wechat` |
| `--model` | AI 模型名称 | `MiniMax-M2.1`、`deepseek-chat`、`claude-3-5-sonnet` |
| `--api-key` | 模型 API 密钥 | 从模型提供商获取 |
| `--base-url` | API 端点 | 见下方示例 |
| `--wechat-app-id` | 公众号 AppID（可选，文件直传优化） | `wx1234567890` |
| `--wechat-app-secret` | 公众号 AppSecret（可选，文件直传优化） | 从公众平台获取 |

> **发送文件/图片**：默认即可发送图片、语音、视频，文件通过云中继服务器中转上传到微信。如果你拥有自己的公众号/服务号，可配置 `--wechat-app-id` 和 `--wechat-app-secret` 启用客户端直传模式（更快、文件不经过中转）。公众号仅支持图片/语音/视频，不支持任意文件附件（与企业微信不同），其他类型文件会以文本预览发送。详见 [文件发送指南](file-sending.md)。

### 第四步：开始对话

启动成功后，在微信公众号「灵缇小秘」中发送消息，即可与您本地的 AI Bot 对话。

## 模型配置示例

### MiniMax

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --model MiniMax-M2.1 \
  --api-key YOUR_MINIMAX_API_KEY \
  --base-url "https://api.minimaxi.com/anthropic/v1"
```

### DeepSeek

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --model deepseek-chat \
  --api-key YOUR_DEEPSEEK_API_KEY \
  --base-url "https://api.deepseek.com/v1"
```

### Kimi (Moonshot)

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --model moonshot-v1-8k \
  --api-key YOUR_KIMI_API_KEY \
  --base-url "https://api.moonshot.cn/v1"
```

### Claude (需代理)

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --model claude-3-5-sonnet-20241022 \
  --api-key YOUR_ANTHROPIC_API_KEY \
  --base-url "https://api.anthropic.com"
```

## 架构说明

```
┌──────────────┐     ┌─────────────────┐     ┌──────────────┐
│  微信公众号   │ ──▶ │   云中继网关     │ ──▶ │  本地 Bot    │
│  灵缇小秘     │     │  (仅转发消息)    │     │  (AI 处理)   │
└──────────────┘     └─────────────────┘     └──────────────┘
       ▲                                            │
       └────────────────────────────────────────────┘
                      返回 AI 响应
```

- **云中继网关**：WebSocket 长连接，实时转发消息，不存储任何数据
- **本地 Bot**：运行在您的电脑上，所有数据本地存储

## 常见问题

### Q: 需要保持电脑开机吗？

是的，lingti-bot 运行在您的本地电脑上，需要保持运行状态才能响应消息。

### Q: 支持哪些 AI 模型？

支持所有兼容 OpenAI API 格式的模型，包括 MiniMax、DeepSeek、Kimi、Claude、GPT 等。

### Q: 数据安全吗？

所有对话数据仅存储在您的本地电脑，云中继仅做消息转发，不保存任何内容。

### Q: 如何后台运行？

可以使用 `nohup` 或系统服务：

```bash
# 使用 nohup
nohup lingti-bot relay --user-id YOUR_USER_ID --platform wechat ... &

# 或安装为系统服务
lingti-bot service install
lingti-bot service start
```
