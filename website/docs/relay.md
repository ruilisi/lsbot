# 云中继（`lingti-bot relay`）

云中继是运行 lingti-bot 的**首选方式**。在本地运行一条命令，AI bot 就能接入所有支持的聊天平台——无需服务器、无需公网 IP、无需配置防火墙。

```
聊天平台 → bot.lingti.com（云中继服务器）←WebSocket→ lingti-bot relay（你的电脑）→ AI
```

你的电脑主动向 `bot.lingti.com` 发起出站 WebSocket 连接，由云端服务器承接平台的回调请求。所有 AI 推理均在本地完成，API Key 不会离开你的机器。

---

## 支持云中继的平台

| 平台 | 云中继支持 | 备注 |
|------|-----------|------|
| 微信公众号 | ✅ | 仅支持云中继，不支持自建服务器 |
| 企业微信（WeCom） | ✅ | 注意 [IP 策略限制](#企业微信-ip-策略) |
| 飞书（Feishu / Lark） | ✅ | |
| Slack | ✅ | |
| Bot Page（`bot.lingti.com/bots/<id>`） | ✅ | 浏览器直接访问，无需平台账号 |

其他平台（Telegram、Discord、钉钉、WhatsApp 等）通过 [`gateway`](docs/gateway-vs-relay.md) 接入。

> **钉钉说明：** 钉钉支持 Stream 模式（原生 WebSocket 长连接），天然不需要公网服务器，效果与云中继相同。直接用 `gateway` 即可。

---

## 快速开始

### Bot Page——无需任何平台账号，三步搞定

这是上手 lingti-bot 最简单的方式。无需注册任何聊天平台，打开浏览器就能用。

**第一步：配置 AI**

```bash
lingti-bot agents add mybot \
  --provider minimax \
  --api-key your-minimax-api-key \
  --default
```

**第二步：启动云中继**

```bash
lingti-bot relay
```

启动后终端会打印你的专属链接：

```
[Relay] Your bot page: https://bot.lingti.com/bots/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

**第三步：打开网页开始聊天**

在浏览器中打开上面打印的链接，即可直接与你的 AI bot 对话。无需登录，无需安装任何 App。

> 把链接分享给任何人，他们都能打开多标签对话界面与你的 bot 聊天。UUID 既是标识符也是访问凭证，请像对待密码一样保管。

---

### 微信公众号

微信搜索公众号 **灵缇小秘**，关注后发送任意消息获取你的 `user-id`，然后：

```bash
lingti-bot relay \
  --platform wechat \
  --user-id <你的-user-id> \
  --provider deepseek \
  --api-key sk-xxx
```

完整教程：[docs/wechat-integration.md](docs/wechat-integration.md)

### 企业微信

```bash
lingti-bot relay --platform wecom \
  --wecom-corp-id ww1234567890abcdef \
  --wecom-agent-id 1000002 \
  --wecom-secret "your-secret" \
  --wecom-token "your-token" \
  --wecom-aes-key "your-43-char-aes-key" \
  --provider deepseek \
  --api-key sk-xxx
```

在企业微信管理后台将回调 URL 设置为：`https://bot.lingti.com/wecom`

完整教程：[docs/wecom-integration.md](docs/wecom-integration.md)

### 飞书

```bash
lingti-bot relay --platform feishu \
  --feishu-app-id cli_xxx \
  --feishu-app-secret xxx \
  --user-id your-id \
  --provider claude \
  --api-key sk-ant-xxx
```

完整教程：[docs/feishu-integration.md](docs/feishu-integration.md)

### Slack

```bash
lingti-bot relay --platform slack \
  --slack-bot-token xoxb-xxx \
  --slack-app-token xapp-xxx \
  --user-id your-id \
  --provider claude \
  --api-key sk-ant-xxx
```

完整教程：[docs/slack-integration.md](docs/slack-integration.md)

### Bot Page（无需任何平台账号）

每个 relay 实例都有一个持久 UUID，保存在 `~/.lingti.yaml`。启动时会打印：

```
[Relay] Your bot page: https://bot.lingti.com/bots/xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

任何人拿到这个链接，就能在浏览器里打开多标签对话界面，直接与你的 bot 聊天。无需登录、无需安装任何应用。UUID 既是标识符也是访问凭证——把链接分享给谁，谁就能访问。

```bash
# 不需要 --platform，bot page 可以独立运行
lingti-bot relay \
  --user-id your-id \
  --provider deepseek \
  --api-key sk-xxx
```

轮换 UUID（使现有分享链接失效）：

```bash
lingti-bot relay --refresh-bot-id
```

加上 `--refresh-bot-id` 后，会立即生成新的 UUID 并保存到 `~/.lingti.yaml`，打印新链接后退出，不启动 relay 服务。旧链接立即失效。

---

## 所有参数

```
--user-id        relay 用户 ID（必填，或 RELAY_USER_ID 环境变量）
--platform       feishu | slack | wechat | wecom（仅用 bot page 时可省略）
--server         WebSocket 地址（默认：wss://bot.lingti.com/ws）
--webhook        Webhook 地址（默认：https://bot.lingti.com/webhook）

--provider       AI 提供商：claude、deepseek、kimi、qwen、minimax、ollama 等
--api-key        AI API Key（或 AI_API_KEY 环境变量）
--model          模型名称（或 AI_MODEL 环境变量）
--base-url       自定义 API 地址（或 AI_BASE_URL 环境变量）
--instructions   自定义系统提示文件路径
--max-rounds     每条消息最大工具调用轮数（默认 100）
--call-timeout   AI 调用超时秒数（默认 90）

--refresh-bot-id 生成新的 bot UUID（使现有 bot page 链接失效）

# 企业微信
--wecom-corp-id  --wecom-agent-id  --wecom-secret  --wecom-token  --wecom-aes-key

# 微信公众号
--wechat-app-id  --wechat-app-secret

# 飞书
--feishu-app-id  --feishu-app-secret

# Slack
--slack-bot-token  --slack-app-token
```

所有参数均支持环境变量，详见 [CONFIGURATION.md](CONFIGURATION.md)。

---

## 配置文件

将凭据写入 `~/.lingti.yaml`，之后只需 `lingti-bot relay` 即可启动：

```yaml
relay:
  platform: wecom
  user_id: your-id

ai:
  provider: deepseek
  api_key: sk-xxx
  model: deepseek-chat

platforms:
  wecom:
    corp_id: ww1234567890abcdef
    agent_id: 1000002
    secret: your-secret
    token: your-token
    aes_key: your-43-char-aes-key
```

之后：

```bash
lingti-bot relay
```

---

## 工作原理

```
1. lingti-bot relay 向 wss://bot.lingti.com/ws 建立 WebSocket 连接
2. 携带 user_id、platform 和平台凭据进行认证
3. 云中继服务器为你的平台注册公网端点
   （例如企业微信回调：https://bot.lingti.com/wecom）
4. 平台消息送达云中继 → 通过 WebSocket 转发到你的本地机器
5. 本地 AI 处理完成 → POST 响应到 https://bot.lingti.com/webhook
6. 云中继服务器调用平台 API 将回复发送给用户
```

AI API Key 只在第 4–5 步本地使用，不经过云中继服务器。

### 连接稳定性

| 参数 | 数值 |
|------|------|
| 心跳间隔 | 3 秒 |
| 读超时 | 60 秒 |
| 首次重连延迟 | 5 秒 |
| 最大重连延迟 | 40 秒 |
| 重连策略 | 指数退避 |

网络中断后自动重连，无需人工干预。

---

## 企业微信 IP 策略

> **注意：** 企业微信云中继偶尔会出现 API 调用失败，原因是 `bot.lingti.com` 的 IP（`106.52.166.51`）由所有用户共用。当流量较大或请求模式异常时，腾讯的 IP 信誉系统可能将该 IP 临时标记为可疑。

**表现症状：**
- 回调验证成功，但发送回复失败
- 日志中出现企业微信 API 返回权限错误或 IP 相关错误
- 时好时坏、不稳定

**根本原因：** 企业微信要求调用发消息 API 的 IP 必须在"企业可信 IP"名单中。`bot.lingti.com` 是多用户共享的服务器，腾讯可能对其 IP 施加更严格的审查。

**解决方案：**

1. **自建中继服务器** — 在自己的 VPS 上运行 `lingti-bot-server`，把自己的 IP 加入企业可信 IP。参考 [lingti-bot-server](https://github.com/ruilisi/lingti-bot-server)。

2. **改用 `gateway` 模式** — 如果你有公网服务器，直接运行 `lingti-bot gateway`。企业微信回调直接打到你的服务器，不经过共享 IP。

3. **换用其他平台** — 微信公众号（`--platform wechat`）没有这个 IP 问题，飞书和 Slack 的云中继也不存在此问题。

4. **重新添加 IP 后重试** — 进入企业微信管理后台：应用管理 → 企业可信 IP → 添加 `106.52.166.51`。有时重新添加或等待一段时间后问题会自动消失。

---

## 后台运行

### macOS / Linux（nohup）

```bash
nohup lingti-bot relay > ~/.lingti/relay.log 2>&1 &
echo $! > ~/.lingti/relay.pid
```

停止：
```bash
kill $(cat ~/.lingti/relay.pid)
```

### Linux（systemd）

创建 `/etc/systemd/system/lingti-bot-relay.service`：

```ini
[Unit]
Description=lingti-bot Cloud Relay
After=network.target

[Service]
Type=simple
User=ubuntu
EnvironmentFile=/etc/lingti-bot/env
ExecStart=/usr/local/bin/lingti-bot relay
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

环境变量文件 `/etc/lingti-bot/env`：
```bash
RELAY_USER_ID=your-id
AI_PROVIDER=deepseek
AI_API_KEY=sk-xxx
WECOM_CORP_ID=ww1234567890abcdef
WECOM_AGENT_ID=1000002
WECOM_SECRET=your-secret
WECOM_TOKEN=your-token
WECOM_AES_KEY=your-aes-key
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now lingti-bot-relay
sudo journalctl -u lingti-bot-relay -f
```

---

## relay vs gateway 对比

| | `relay`（云中继） | `gateway`（自建） |
|---|---|---|
| 需要公网服务器 | 否 | 是（或隧道工具） |
| 支持平台 | 微信、企业微信、飞书、Slack、bot page | 全部 19 个平台 |
| 配置步骤 | 3 步 | 因平台而异 |
| AI 运行位置 | 本地 | 本地（或服务器） |
| API Key 安全 | 仅本地 | 仅本地 |
| 企业微信 IP 问题 | 可能出现（共享 IP） | 不存在（用自己的 IP） |
| 延迟 | 多一跳 WebSocket（约 50ms） | 直连 |

除非需要 relay 不支持的平台，或需要规避企业微信共享 IP 问题，否则优先使用 `relay`。

详细对比见 [docs/gateway-vs-relay.md](docs/gateway-vs-relay.md)。

---

## 安全性

- **传输层：** WebSocket over TLS（`wss://`）
- **平台凭据：** 仅在建立连接时发送一次，仅用于回调验证和 API 代理调用
- **企业微信消息内容：** 加密 XML 原样转发到本地，云中继服务器不解密企业微信消息
- **AI API Key：** 仅在本地使用，不经过云中继服务器
- **Bot page 访问控制：** URL 中的 UUID 是唯一的访问凭证，请像对待密码一样保管

---

## 常见问题排查

**启动时提示"platform is required"**
→ 添加 `--platform wecom`（或 feishu/slack/wechat），或在 `~/.lingti.yaml` 中设置 `relay.platform`。如果只用 bot page 模式，确保配置文件中存在 `bot_id`。

**连接成功但收不到消息**
→ 确认平台管理后台的回调 URL 已正确配置并指向 `bot.lingti.com`。

**企业微信回调验证失败**
→ 确保在企业微信后台点击「保存」*之前*，`lingti-bot relay` 已经在运行。云中继服务器需要活跃的 WebSocket 连接才能处理验证请求。

**企业微信验证通过后发送回复失败**
→ 参见上方 [企业微信 IP 策略](#企业微信-ip-策略)。

**relay 频繁断线**
→ 检查网络状况。客户端会自动重连。如果运行在会休眠的设备上，建议改用服务器或 VPS。

**Bot page 显示"bot offline"**
→ relay 客户端未连接，或 `bot_id` 未注册。重启 `lingti-bot relay`，确认 URL 中的 UUID 与启动时打印的一致。

---

## 相关文档

- [CONFIGURATION.md](CONFIGURATION.md) — 完整配置参考
- [AI-PROVIDERS.md](AI-PROVIDERS.md) — 支持的 AI 提供商列表
- [docs/gateway-vs-relay.md](docs/gateway-vs-relay.md) — relay 与 gateway 详细对比
- [docs/wecom-integration.md](docs/wecom-integration.md) — 企业微信接入教程
- [docs/wechat-integration.md](docs/wechat-integration.md) — 微信公众号接入教程
- [docs/feishu-integration.md](docs/feishu-integration.md) — 飞书接入教程
- [docs/slack-integration.md](docs/slack-integration.md) — Slack 接入教程
- [docs/chat-platforms.md](docs/chat-platforms.md) — 全部 19 个支持平台
