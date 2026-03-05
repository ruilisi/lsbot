# 企业微信集成指南

本指南介绍如何将 lingti-bot 接入企业微信（WeCom/WeChat Work）。

## TL;DR - 3 分钟快速接入（云中继模式）

```bash
# 1. 安装 lingti-bot
# macOS / Linux / WSL:
curl -fsSL https://files.lingti.com/install-bot.sh | bash
# Windows (PowerShell):
irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot

# 2. 在企业微信后台创建自建应用，获取以下信息：
#    - 企业ID (CorpID): 我的企业 → 企业信息 → 企业ID
#    - AgentId: 应用管理 → 自建 → 创建应用 → AgentId
#    - Secret: 应用详情 → Secret
#    - Token & EncodingAESKey: 应用详情 → 接收消息 → 设置API接收

# 3. 配置企业可信IP（云中继服务器IP）
#    应用管理 → 找到你的应用 → 企业可信IP → 配置
#    添加: 106.52.166.51

# 4. 一条命令搞定！（同时处理回调验证和消息）
lingti-bot relay --platform wecom \
  --wecom-corp-id ww1234567890abcdef \
  --wecom-agent-id 1000002 \
  --wecom-secret "your-app-secret" \
  --wecom-token "your-callback-token" \
  --wecom-aes-key "your-43-char-encoding-aes-key" \
  --provider deepseek \
  --api-key "sk-your-deepseek-api-key"

# 5. 在企业微信后台「接收消息」中配置回调 URL：
#    https://bot.lingti.com/wecom
#    点击保存，验证会自动完成，消息立即可以处理

# 完成！现在可以在企业微信中与 AI 对话了
```

---

## 前置条件

1. 企业微信管理员账号
2. 公网可访问的服务器（用于接收回调）**或** 使用云中继模式（无需公网服务器）

## 第一步：创建企业微信应用

### 1.1 获取企业 ID (CorpID)

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin/frame)
2. 点击「我的企业」
3. 在页面底部找到「企业ID」

### 1.2 创建自建应用

1. 进入「应用管理」→「自建」→「创建应用」
2. 填写应用信息：
   - 应用名称：如 "灵小缇 AI 助手"
   - 应用 Logo：上传应用图标
   - 可见范围：选择可以使用此应用的部门/成员
3. 创建完成后，记录以下信息：
   - **AgentId**：应用的 AgentId
   - **Secret**：应用的 Secret（点击查看）

### 1.3 配置接收消息

1. 在应用详情页，找到「接收消息」→「设置API接收」
2. 填写回调配置：
   - **URL**：回调地址（见下方说明）
   - **Token**：自定义字符串（32位以内，字母数字）
   - **EncodingAESKey**：点击「随机获取」或自定义（43位）
3. 点击「保存」，企业微信会验证 URL 的有效性

**回调 URL 格式：**

假设你的服务器公网 IP 为 `203.0.113.100`，`--wecom-port` 设置为 `8080`，则回调 URL 为：

```
http://203.0.113.100:8080/wecom/callback
```

> **注意**：企业微信支持 HTTP 回调，无需 HTTPS。如需 HTTPS，可配置反向代理（见下文）。

### 1.4 配置企业可信IP（云中继模式必须）

如果使用云中继模式，需要将云中继服务器 IP 添加到企业可信IP列表：

1. 进入「应用管理」→ 找到你的应用
2. 在应用详情页找到「企业可信IP」→ 点击「配置」
3. 添加云中继服务器 IP：`106.52.166.51`
4. 点击「确定」保存

> **为什么需要这一步？** 云中继服务器代替你的本地客户端向企业微信发送消息。企业微信会验证 API 调用来源 IP，只有在可信IP列表中的 IP 才能调用发送消息接口。

## 部署方案对比

| 方案 | 需要公网服务器 | 回调 URL |
|------|---------------|---------|
| 自建服务器 | 是 | `http://YOUR_IP:8080/wecom/callback` |
| 云中继 | 否 | `https://bot.lingti.com/wecom` |

## 第二步：部署 lingti-bot

### 方式一：云中继模式（推荐，一键接入，无需公网服务器）

使用官方云中继服务，无需准备公网服务器。只需 3 步即可完成接入：

**一键接入流程：**

1. **配置企业可信IP**：应用管理 → 找到应用 → 企业可信IP → 添加 `106.52.166.51`
2. **启动 relay**：运行 `lingti-bot relay` 命令（同时处理回调验证和消息）
3. **配置回调 URL**：在「接收消息」设置中填写 `https://bot.lingti.com/wecom` 并保存

就这么简单！无需单独的 verify 步骤，relay 命令会自动处理验证请求。

**启动 relay 命令**

```bash
# 一条命令搞定验证和消息处理
lingti-bot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

运行后会显示连接成功信息。

**配置企业微信回调 URL：**
1. 进入应用详情 → 接收消息 → 设置API接收
2. URL 填写：`https://bot.lingti.com/wecom`
3. Token 和 EncodingAESKey 填写之前记录的值
4. 点击「保存」

企业微信会向 `https://bot.lingti.com/wecom` 发送验证请求，云中继服务器使用你发送的凭据完成验证。如果看到保存成功，说明验证通过。

验证成功后，用户发送的消息会立即被 AI 处理并响应。

> **说明**：
> - `--user-id`：可选参数，WeCom 平台会自动从 corp_id 生成
> - `--wecom-corp-id`：企业 ID
> - `--wecom-agent-id`：应用的 AgentId
> - `--wecom-secret`：应用的 Secret
> - `--wecom-token`：回调配置中的 Token
> - `--wecom-aes-key`：回调配置中的 EncodingAESKey
>
> **工作原理**：当你运行 `lingti-bot relay` 时，客户端会通过 WebSocket 将 WeCom 凭据发送到云中继服务器。当企业微信发送回调验证请求时，云中继服务器使用这些凭据完成验证。用户消息也会通过同一连接转发到本地进行 AI 处理。

也可以使用环境变量：

```bash
export WECOM_CORP_ID="your-corp-id"
export WECOM_AGENT_ID="your-agent-id"
export WECOM_SECRET="your-secret"
export WECOM_TOKEN="your-callback-token"
export WECOM_AES_KEY="your-encoding-aes-key"
export AI_PROVIDER="deepseek"
export AI_API_KEY="your-api-key"

lingti-bot relay --platform wecom
```

### 方式二：自建服务器部署

在公网服务器上直接运行 lingti-bot：

```bash
# 1. 安装 lingti-bot
# macOS / Linux / WSL:
curl -fsSL https://files.lingti.com/install-bot.sh | bash
# Windows (PowerShell):
irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot

# 2. 启动服务
lingti-bot gateway \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --wecom-port 8080 \
  --provider deepseek \
  --model deepseek-chat \
  --api-key YOUR_API_KEY \
  --base-url "https://api.deepseek.com/v1"
```

然后在企业微信后台配置回调 URL：

```
http://YOUR_SERVER_IP:8080/wecom/callback
```

### 方式二：使用环境变量

```bash
export WECOM_CORP_ID="your-corp-id"
export WECOM_AGENT_ID="your-agent-id"
export WECOM_SECRET="your-secret"
export WECOM_TOKEN="your-callback-token"
export WECOM_AES_KEY="your-encoding-aes-key"
export WECOM_PORT="8080"
export AI_PROVIDER="deepseek"
export AI_API_KEY="your-api-key"
export AI_BASE_URL="https://api.deepseek.com/v1"
export AI_MODEL="deepseek-chat"

lingti-bot gateway
```

### 方式三：使用 HTTPS（可选）

如需 HTTPS，使用 Nginx 配置反向代理：

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /wecom/callback {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

回调 URL 配置为：`https://your-domain.com/wecom/callback`

## 第三步：验证配置

### 3.1 URL 验证

保存回调配置时，企业微信会发送 GET 请求验证 URL：

```
GET /wecom/callback?msg_signature=xxx&timestamp=xxx&nonce=xxx&echostr=xxx
```

lingti-bot 会自动处理验证请求，解密 echostr 并返回明文。

### 3.2 消息测试

1. 在企业微信 App 中找到你创建的应用
2. 发送消息给应用
3. 检查 lingti-bot 日志确认消息接收

```bash
# 查看详细日志
lingti-bot gateway --log verbose ...
```

## 架构说明

### 云中继模式（推荐）

```
┌──────────────┐     ┌─────────────────────────────┐     ┌──────────────────┐
│  企业微信     │     │    云中继服务器              │     │   本地客户端      │
│  用户消息     │ ──▶ │  bot.lingti.com/wecom       │ ──▶ │  lingti-bot relay │
│              │     │  (验证回调、转发消息)         │ WS  │  (AI 处理)        │
└──────────────┘     └─────────────────────────────┘     └──────────────────┘
       ▲                          │                              │
       │                          │ ◀─────────────────────────────┘
       │                          │        Webhook 响应
       └──────────────────────────┘
              发送消息 API
```

**优点**：
- 无需公网服务器，本地运行客户端即可
- 凭据动态同步，一键完成企业微信回调验证
- 消息经云端中转，但 AI 处理在本地完成

### 自建服务器模式

```
┌──────────────┐     ┌─────────────────────────────┐     ┌──────────────┐
│  企业微信     │     │    公网服务器                │     │   AI 模型    │
│  用户消息     │ ──▶ │  lingti-bot gateway          │ ──▶ │  处理响应    │
│              │     │  http://IP:8080/wecom/callback │    │              │
└──────────────┘     └─────────────────────────────┘     └──────────────┘
       ▲                          │
       └──────────────────────────┘
              发送消息 API
```

### 消息流程

1. 用户在企业微信中发送消息
2. 企业微信服务器 POST 加密消息到回调 URL
3. lingti-bot 解密消息，调用 AI 处理
4. lingti-bot 通过 API 发送响应消息
5. 用户在企业微信中收到回复

## 配置参数说明

| 参数 | 环境变量 | 说明 |
|------|---------|------|
| `--wecom-corp-id` | `WECOM_CORP_ID` | 企业 ID |
| `--wecom-agent-id` | `WECOM_AGENT_ID` | 应用 AgentId |
| `--wecom-secret` | `WECOM_SECRET` | 应用 Secret |
| `--wecom-token` | `WECOM_TOKEN` | 回调 Token |
| `--wecom-aes-key` | `WECOM_AES_KEY` | 回调 EncodingAESKey |
| `--wecom-port` | `WECOM_PORT` | 回调服务端口 (默认 8080) |
| `--provider` | `AI_PROVIDER` | AI 提供商: claude, deepseek, kimi |
| `--model` | `AI_MODEL` | 模型名称 |
| `--api-key` | `AI_API_KEY` | API 密钥 |
| `--base-url` | `AI_BASE_URL` | API 端点 |

## 后台运行

### 使用 nohup

```bash
nohup lingti-bot gateway \
  --wecom-corp-id ... \
  --wecom-port 8080 \
  ... > /var/log/lingti-bot.log 2>&1 &
```

### 使用 systemd

创建 `/etc/systemd/system/lingti-bot.service`：

```ini
[Unit]
Description=Lingti Bot WeCom Service
After=network.target

[Service]
Type=simple
User=root
Environment="WECOM_CORP_ID=your-corp-id"
Environment="WECOM_AGENT_ID=your-agent-id"
Environment="WECOM_SECRET=your-secret"
Environment="WECOM_TOKEN=your-token"
Environment="WECOM_AES_KEY=your-aes-key"
Environment="WECOM_PORT=8080"
Environment="AI_PROVIDER=deepseek"
Environment="AI_API_KEY=your-api-key"
Environment="AI_BASE_URL=https://api.deepseek.com/v1"
Environment="AI_MODEL=deepseek-chat"
ExecStart=/usr/local/bin/lingti-bot gateway
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable lingti-bot
sudo systemctl start lingti-bot
sudo systemctl status lingti-bot
```

## 常见问题

### Q: 云中继模式 URL 验证失败？

1. **确保 relay 命令正在运行**：在企业微信后台保存配置之前，必须先运行 `lingti-bot relay` 命令
2. **检查凭据是否正确**：Token 和 EncodingAESKey 必须与企业微信后台填写的完全一致
3. **注意复制时不要有空格**：特别是 EncodingAESKey（43位）前后不能有空格
4. **检查网络连接**：确保能连接到 `wss://bot.lingti.com/ws`

### Q: 自建服务器 URL 验证失败？

1. 确保服务器公网可访问，防火墙开放端口
2. 检查 Token 和 EncodingAESKey 配置是否正确（注意复制时不要有空格）
3. 确保 lingti-bot 已启动并监听正确端口
4. 查看日志：`lingti-bot gateway --log verbose`

### Q: 收不到消息？

1. 检查应用的可见范围是否包含测试用户
2. 确保回调 URL 配置正确
3. 检查防火墙是否开放端口：`sudo ufw allow 8080`

### Q: 发送消息失败？

1. 检查 access_token 是否有效（查看日志）
2. 确认 Secret 配置正确
3. 确保 AgentId 正确

### Q: 如何获取用户真实姓名？

默认回调只返回 UserID，需要调用通讯录 API 获取用户信息。需要在「应用管理」中配置「通讯录同步」权限。

## 安全建议

1. **Token 保密**：不要将 Token 和 EncodingAESKey 提交到代码仓库
2. **IP 白名单**：在企业微信后台配置可信 IP
3. **防火墙**：仅开放必要端口
4. **日志脱敏**：生产环境不要记录完整消息内容

## 相关文档

- [企业微信开发者中心](https://developer.work.weixin.qq.com/)
- [回调配置文档](https://developer.work.weixin.qq.com/document/path/90930)
- [获取 access_token](https://developer.work.weixin.qq.com/document/path/91039)
- [发送应用消息](https://developer.work.weixin.qq.com/document/path/90236)
