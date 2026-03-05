# 云中继（Cloud Relay）技术方案详解

> 以企业微信接入为例，详解云中继的架构原理、工作流程与部署实践

## 什么是云中继？

云中继是 lingti-bot 提供的一种**免公网服务器**接入方案。传统的聊天平台 Bot 接入需要一台拥有公网 IP 的服务器来接收回调消息，而云中继通过在云端部署一个中转服务器（`bot.lingti.com`），让用户在本地（笔记本、家庭网络、公司内网）即可完成 Bot 的全部功能。

**一句话总结：你不需要买服务器，只需在本地运行一条命令，就能拥有一个企业微信 AI 助手。**

## 为什么需要云中继？

### 传统方案的痛点

要将 AI Bot 接入企业微信，传统方案需要：

1. **购买云服务器**（如腾讯云、阿里云 ECS）
2. **配置公网 IP 和域名**
3. **申请 HTTPS 证书**（部分平台要求）
4. **配置防火墙、安全组**
5. **部署和运维 Bot 服务**

对于个人用户或小团队来说，这些步骤繁琐且有成本。

### 云中继的解决方案

云中继将上述 5 步简化为 **3 步**：

1. 在企业微信后台添加一个 IP 到白名单
2. 在本地运行 `lingti-bot relay` 命令
3. 在企业微信后台填写回调 URL

**无需公网服务器、无需域名、无需 HTTPS 证书、无需防火墙配置。**

## 架构原理

### 整体架构

```
                    互联网（公网）                          内网/本地
┌──────────┐    ┌────────────────────────┐    ┌─────────────────────────┐
│          │    │   云中继服务器           │    │   用户本地环境           │
│ 企业微信  │───▶│   bot.lingti.com       │◀───│   lingti-bot relay      │
│ 服务器    │◀───│                        │───▶│                         │
│          │    │  ┌──────────────────┐   │    │  ┌───────────────────┐  │
│          │    │  │ HTTPS 回调端点   │   │    │  │ WebSocket 客户端  │  │
│          │    │  │ /wecom           │   │    │  │ wss://bot.lingti  │  │
│          │    │  └──────────────────┘   │    │  │ .com/ws           │  │
│          │    │  ┌──────────────────┐   │    │  └───────────────────┘  │
│          │    │  │ WebSocket 服务端 │   │    │  ┌───────────────────┐  │
│          │    │  │ /ws              │   │    │  │ AI 处理引擎       │  │
│          │    │  └──────────────────┘   │    │  │ (DeepSeek/Claude) │  │
│          │    │  ┌──────────────────┐   │    │  └───────────────────┘  │
│          │    │  │ 消息发送代理     │   │    │                         │
│          │    │  │ (企微 API 转发)  │   │    │                         │
│          │    │  └──────────────────┘   │    │                         │
└──────────┘    └────────────────────────┘    └─────────────────────────┘
```

### 三个核心角色

| 角色 | 说明 | 位置 |
|------|------|------|
| **企业微信服务器** | 腾讯运营的企业微信平台，负责收发用户消息 | 腾讯云 |
| **云中继服务器** | `bot.lingti.com`，负责接收回调、转发消息、代理 API 调用 | 公网（IP: `106.52.166.51`） |
| **本地客户端** | `lingti-bot relay`，负责 AI 处理和生成回复 | 用户本地（任意网络环境） |

### 关键设计决策

1. **WebSocket 长连接**：本地客户端主动连接云中继，穿透 NAT/防火墙，无需公网 IP
2. **凭据动态同步**：企业微信凭据通过 WebSocket 发送到云中继，用于回调验证和消息转发
3. **AI 处理在本地**：消息经云中继转发到本地，AI 推理在本地完成，API Key 不经过云中继

## 完整消息流程

### 第一阶段：建立连接

```
本地客户端                        云中继服务器
    │                                │
    │──── WebSocket 连接 ───────────▶│
    │                                │
    │──── 认证消息 (auth) ──────────▶│  包含:
    │     {                          │  - user_id
    │       type: "auth",            │  - platform: "wecom"
    │       user_id: "wecom_ww123",  │  - wecom_corp_id
    │       platform: "wecom",       │  - wecom_agent_id
    │       client_version: "1.5.0", │  - wecom_secret
    │       wecom_corp_id: "ww123",  │  - wecom_token
    │       wecom_token: "...",      │  - wecom_aes_key
    │       wecom_aes_key: "...",    │
    │       ...                      │
    │     }                          │
    │                                │
    │◀─── 认证结果 (auth_result) ────│  { success: true, session_id: "xxx" }
    │                                │
    │◀──── ping ─────────────────────│  每 3 秒心跳保活
    │───── pong ────────────────────▶│
```

### 第二阶段：回调验证（首次配置时）

```
企业微信                   云中继服务器                    本地客户端
   │                          │                              │
   │── GET /wecom ───────────▶│                              │
   │   ?msg_signature=xxx     │                              │
   │   &timestamp=xxx         │  使用客户端上报的             │
   │   &nonce=xxx             │  Token + AESKey              │
   │   &echostr=xxx           │  解密 echostr                │
   │                          │                              │
   │◀── 返回解密后的明文 ──────│                              │
   │                          │                              │
   │   ✅ 验证通过！           │                              │
```

> **重点**：回调验证完全由云中继服务器处理，使用本地客户端通过 WebSocket 上报的凭据。用户无需任何操作。

### 第三阶段：消息收发

```
用户                企业微信              云中继服务器           本地客户端          AI 服务
 │                    │                     │                    │                  │
 │── 发送消息 ───────▶│                     │                    │                  │
 │   "帮我写周报"      │                     │                    │                  │
 │                    │── POST /wecom ─────▶│                    │                  │
 │                    │   (加密 XML)         │                    │                  │
 │                    │                     │── WebSocket ──────▶│                  │
 │                    │                     │   wecom_raw 消息    │                  │
 │                    │                     │   (原始加密体)       │                  │
 │                    │                     │                    │── 本地解密 ───────│
 │                    │                     │                    │   AES-128-CBC     │
 │                    │                     │                    │                  │
 │                    │                     │                    │── API 调用 ──────▶│
 │                    │                     │                    │   "帮我写周报"     │
 │                    │                     │                    │                  │
 │                    │                     │                    │◀── AI 回复 ───────│
 │                    │                     │                    │   "以下是周报..."  │
 │                    │                     │                    │                  │
 │                    │                     │◀── Webhook ────────│                  │
 │                    │                     │   响应消息           │                  │
 │                    │◀── 企微发送 API ─────│                    │                  │
 │                    │   (代理调用)          │                    │                  │
 │◀── 收到回复 ───────│                     │                    │                  │
 │   "以下是周报..."   │                     │                    │                  │
```

### 消息处理细节

**WeCom 平台采用「原始转发」模式**：云中继将企业微信发来的加密 XML 原样转发给本地客户端，由本地客户端使用 Token 和 AESKey 在本地解密。这意味着：

- 云中继服务器**不解密消息内容**（WeCom 平台下）
- 消息的加解密完全在本地完成
- 凭据仅用于回调验证和 API 调用代理

## 协议规范

### WebSocket 消息类型

| 类型 | 方向 | 说明 |
|------|------|------|
| `auth` | 客户端 → 服务端 | 认证，携带平台凭据 |
| `auth_result` | 服务端 → 客户端 | 认证结果，返回 session_id |
| `message` | 服务端 → 客户端 | 标准消息（飞书/Slack 等平台） |
| `wecom_raw` | 服务端 → 客户端 | 企微原始加密消息 |
| `response` | 客户端 → 服务端 | 通过 Webhook HTTP POST 发送 |
| `ping` / `pong` | 双向 | 心跳保活（3 秒间隔） |
| `error` | 服务端 → 客户端 | 错误通知 |

### 连接保活与重连

| 参数 | 值 | 说明 |
|------|------|------|
| 心跳间隔 | 3 秒 | 客户端发 pong 响应服务端 ping |
| 写超时 | 10 秒 | WebSocket 写操作超时 |
| 读超时 | 60 秒 | WebSocket 读操作超时 |
| 初始重连延迟 | 5 秒 | 断线后首次重连等待时间 |
| 最大重连延迟 | 5 分钟 | 指数退避上限 |
| 重连策略 | 指数退避 | 5s → 10s → 20s → ... → 5min |

## 以企业微信为例：完整接入流程

### 前提条件

- 一台能运行 lingti-bot 的电脑（macOS / Linux / Windows）
- 企业微信管理员权限
- 一个 AI 服务的 API Key（如 DeepSeek）

### 步骤一：安装 lingti-bot

**macOS / Linux / WSL:**
```bash
curl -fsSL https://files.lingti.com/install-bot.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot
```

### 步骤二：在企业微信后台创建应用

1. 登录 [企业微信管理后台](https://work.weixin.qq.com/wework_admin/frame)
2. 「我的企业」→ 页面底部获取 **企业 ID (CorpID)**
3. 「应用管理」→「自建」→「创建应用」→ 记录 **AgentId** 和 **Secret**
4. 应用详情 →「接收消息」→「设置API接收」→ 记录 **Token** 和 **EncodingAESKey**（先不要保存，等步骤四再保存）

### 步骤三：添加企业可信 IP

在应用详情页 →「企业可信IP」→ 添加云中继服务器 IP：

```
106.52.166.51
```

> 云中继服务器代替本地客户端调用企业微信发送消息 API，企业微信会验证调用来源 IP 是否在白名单中。

### 步骤四：启动本地客户端

```bash
lingti-bot relay --platform wecom \
  --wecom-corp-id ww1234567890abcdef \
  --wecom-agent-id 1000002 \
  --wecom-secret "your-app-secret" \
  --wecom-token "your-callback-token" \
  --wecom-aes-key "your-43-char-encoding-aes-key" \
  --provider deepseek \
  --api-key "sk-your-deepseek-api-key"
```

看到类似以下输出表示连接成功：

```
[INFO] Connected to relay server (session: abc123)
[INFO] Platform: wecom | Provider: deepseek
[INFO] Waiting for messages...
```

### 步骤五：配置企业微信回调 URL

回到步骤二中「设置API接收」页面：

- **URL** 填写：`https://bot.lingti.com/wecom`
- **Token** 和 **EncodingAESKey** 填写之前记录的值
- 点击「保存」

保存成功 = 验证通过。现在可以在企业微信中与 AI 对话了。

### 使用配置文件（推荐长期使用）

运行交互式向导自动生成配置文件：

```bash
lingti-bot onboard
```

或手动创建 `~/.lingti.yaml`：

```yaml
mode: relay

relay:
  platform: wecom

ai:
  provider: deepseek
  api_key: sk-your-deepseek-api-key

platforms:
  wecom:
    corp_id: ww1234567890abcdef
    agent_id: 1000002
    secret: your-app-secret
    token: your-callback-token
    aes_key: your-43-char-encoding-aes-key
```

之后只需一条命令即可启动：

```bash
lingti-bot relay
```

## 云中继 vs 自建服务器

| 对比项 | 云中继 (`relay`) | 自建服务器 (`router`) |
|--------|-----------------|---------------------|
| **需要公网服务器** | 否 | 是 |
| **需要域名/证书** | 否 | 可选 |
| **回调 URL** | `https://bot.lingti.com/wecom` | `http://YOUR_IP:PORT/wecom/callback` |
| **部署位置** | 笔记本、家庭网络、内网均可 | 必须公网可访问 |
| **配置步骤** | 3 步 | 需要服务器运维 |
| **消息延迟** | 多一跳 WebSocket 中转（通常 < 100ms） | 直连，延迟最低 |
| **AI 处理位置** | 本地 | 服务器本地 |
| **API Key 安全** | 仅存储在本地 | 存储在服务器 |
| **可用性** | 依赖本地客户端在线 | 依赖服务器在线 |
| **适用场景** | 个人/小团队、快速验证 | 生产环境、高可用需求 |

## 安全性说明

### 数据流向

```
API Key:      仅存储在本地，直接调用 AI 服务 API，不经过云中继
消息内容:     企微加密 XML → 云中继原样转发 → 本地解密（云中继不解密 WeCom 消息）
平台凭据:     通过 WebSocket（WSS 加密）传输到云中继，用于回调验证和 API 调用代理
```

### 安全设计

1. **传输加密**：WebSocket 使用 WSS（TLS）协议
2. **消息加密**：企业微信消息使用 AES-128-CBC 加密，在本地解密
3. **API Key 隔离**：AI 服务的 API Key 仅存储在本地，不经过云中继
4. **企微凭据用途有限**：上报到云中继的凭据仅用于回调验证和发送消息 API 代理

### 建议

- 不要将凭据提交到代码仓库，使用环境变量或配置文件
- 定期轮换企业微信应用 Secret
- 生产环境建议使用自建服务器模式以获得完全控制

## 后台运行

### macOS / Linux（使用 nohup）

```bash
nohup lingti-bot relay > /var/log/lingti-bot.log 2>&1 &
```

### Linux（使用 systemd）

创建 `/etc/systemd/system/lingti-bot.service`：

```ini
[Unit]
Description=Lingti Bot Cloud Relay
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
WECOM_CORP_ID=ww1234567890abcdef
WECOM_AGENT_ID=1000002
WECOM_SECRET=your-secret
WECOM_TOKEN=your-token
WECOM_AES_KEY=your-aes-key
AI_PROVIDER=deepseek
AI_API_KEY=sk-your-api-key
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now lingti-bot
```

## 常见问题

### 回调验证失败

1. **确保 relay 命令正在运行** — 必须先启动 relay，再在企业微信后台保存回调配置
2. **检查 Token 和 AESKey** — 必须与企业微信后台完全一致，注意复制时不要带空格
3. **检查网络** — 确保能连接到 `wss://bot.lingti.com/ws`

### 消息能收到但回复失败

1. **检查企业可信 IP** — 确认已添加 `106.52.166.51`
2. **检查 Secret 和 AgentId** — 用于调用发送消息 API

### 断线后会怎样？

客户端会自动重连，采用指数退避策略（5 秒 → 10 秒 → ... → 5 分钟上限）。重连成功后自动恢复消息处理。断线期间用户发送的消息会丢失。

### 能同时运行多个 relay 实例吗？

可以，每个实例使用不同的 `--user-id`（或不同的企微应用凭据）即可。适用于一台机器接入多个企微应用的场景。

## 相关文档

- [企业微信集成指南](wecom-integration.md) — 完整的企业微信接入教程
- [配置说明](../CONFIGURATION.md) — 配置文件和环境变量参考
- [AI 服务商](../AI-PROVIDERS.md) — 支持的 AI 后端列表
- [聊天平台](chat-platforms.md) — 支持的 19 种聊天平台
