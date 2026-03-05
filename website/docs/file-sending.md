# 文件发送指南

lingti-bot 支持通过自然语言在聊天平台中传输本地文件。用户只需对 AI 说"把桌面上的 xxx 文件发给我"，即可接收文件。

## 平台支持

| 平台 | 图片 | 语音 | 视频 | 任意文件 | 额外配置 |
|------|------|------|------|---------|---------|
| 企业微信 (WeCom) | ✅ | ✅ | ✅ | ✅ | 无需额外配置 |
| 微信公众号 | ✅ | ✅ | ✅ | ⚠️ 文本预览 | 无需额外配置（可选优化） |
| 飞书 / Slack | — | — | — | — | 暂不支持 |

- **企业微信**：支持所有文件类型，包括文档、压缩包等任意格式
- **微信公众号**：支持图片/语音/视频直接发送；不支持的文件类型（如 .md、.pdf、.docx）会以文本预览形式发送（截取前 500 字）

## 企业微信 (WeCom)

企业微信通过 WeCom API 发送文件，relay 模式下自动启用（已配置 Corp ID + Secret）。

```bash
lingti-bot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

支持的媒体类型：`image`、`voice`、`video`、`file`（任意文件）。

详细配置：[企业微信集成指南](wecom-integration.md)

## 微信公众号

微信公众号支持两种文件发送方式。两种方式都使用微信[发送客服消息](https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Service_Center_messages.html)发送媒体消息，区别在于**谁来调用微信 API**。

### 背景：主动回复 vs 发送客服消息

微信公众号支持两种消息发送机制：

| | 主动回复 | 发送客服消息（又称被动回复） |
|--|----------|----------------------------|
| **工作方式** | 主动调用微信 HTTP API 推送消息 | 用户发消息后 48 小时内直接返回 XML 响应 |
| **发送时机** | 随时随地可以发送 | 必须在用户发消息后的 48 小时内回复 |
| **需要凭据** | 需要 access_token（通过 AppID + AppSecret 获取） | 不需要（直接返回 XML） |
| **媒体文件** | 先上传素材获取 media_id，再发送 | 需要 media_id，但无法在 48 小时内完成上传 |
| **公众号与 Bot 关系** | 一个公众号只能给一个 Bot 使用 | 一个公众号可以给多个 Bots 使用 |

> **关键区别**：发送媒体文件必须先通过临时素材上传接口获取 `media_id`，这个接口同样需要 `access_token`。因此，**发送媒体文件必须走主动回复接口**，都需要 access_token。

### 方式一：服务端中转（默认，无需额外配置）

> **零配置即可使用** — 不需要提供任何公众号凭据

```bash
# 和普通文本对话完全一样的启动方式，不需要额外参数
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

**工作原理：**

```
┌──────────────┐    ② base64    ┌─────────────────┐  ③ 上传素材   ┌──────────────┐
│  本地客户端   │ ────────────▶ │  云中继服务器     │ ───────────▶ │   微信 API   │
│  lingti-bot  │   webhook     │  bot.lingti.com  │  ④ 发送消息   │              │
│  读取本地文件 │              │  使用灵缇小秘凭据 │ ◀─────────── │              │
└──────────────┘              └─────────────────┘               └──────────────┘
       ① AI 调用                                                       │
       file_send                                                       │ ⑤ 推送消息
       读取文件                                                        ▼
                                                                ┌──────────────┐
                                                                │  微信公众号   │
                                                                │  用户收到图片 │
                                                                └──────────────┘
```

1. AI 调用 `file_send` 工具，客户端读取本地文件
2. 客户端将文件 base64 编码后通过 webhook 发送到云中继服务器 `bot.lingti.com`
3. 云中继服务器使用**灵缇小秘公众号自身的 AppID/AppSecret** 获取 access_token
4. 服务器调用微信临时素材上传接口获取 `media_id`，再通过客服消息接口发送给用户
5. 用户在公众号中收到图片/语音/视频消息

**优点：**
- 零配置，开箱即用，不需要你拥有自己的公众号
- 用户侧体验与方式二完全一致

**限制：**
- 文件需要通过网络传输到云中继服务器（base64 编码后体积增大约 33%）
- 依赖云中继服务器的可用性

### 方式二：客户端直传（可选，需自己的公众号凭据）

> **需要配置** `--wechat-app-id` 和 `--wechat-app-secret`

如果你拥有自己的已认证微信公众号或服务号，可以配置客户端直接调用微信 API，跳过云中继服务器。

```bash
lingti-bot relay \
  --user-id YOUR_USER_ID \
  --platform wechat \
  --wechat-app-id YOUR_APP_ID \
  --wechat-app-secret YOUR_APP_SECRET \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

也可使用环境变量：

```bash
export WECHAT_APP_ID="wx1234567890abcdef"
export WECHAT_APP_SECRET="your-app-secret"

lingti-bot relay --user-id YOUR_USER_ID --platform wechat --api-key YOUR_API_KEY
```

**工作原理：**

```
┌──────────────┐  ② 上传素材   ┌──────────────┐
│  本地客户端   │ ───────────▶ │   微信 API   │
│  lingti-bot  │  ③ 发送消息   │              │
│  直接调用API │ ◀─────────── │              │
└──────────────┘              └──────────────┘
       ① AI 调用                      │
       file_send                      │ ④ 推送消息
       读取文件                       ▼
                               ┌──────────────┐
                               │  微信公众号   │
                               │  用户收到图片 │
                               └──────────────┘
```

1. AI 调用 `file_send` 工具，客户端读取本地文件
2. 客户端使用**你自己的 AppID/AppSecret** 获取 access_token，直接上传素材
3. 客户端通过客服消息接口发送媒体消息
4. 用户在公众号中收到图片/语音/视频消息

**获取凭据：**
1. 登录[微信公众平台](https://mp.weixin.qq.com/)
2. 进入「设置与开发」→「基本配置」
3. 获取 **AppID** 和 **AppSecret**

**优点：**
- 文件直接从本地上传到微信，不经过云中继服务器，速度更快
- 不依赖云中继服务器的可用性
- 数据不经过第三方服务器

**限制：**
- 需要拥有自己的已认证微信公众号或服务号
- 需要额外配置 AppID 和 AppSecret

### 两种方式对比

| | 方式一：服务端中转 | 方式二：客户端直传 |
|--|-------------------|-------------------|
| **额外配置** | 无需 | 需要 AppID + AppSecret |
| **需要自己的公众号** | 不需要 | 需要 |
| **传输路径** | 本地 → 云中继 → 微信 | 本地 → 微信 |
| **速度** | 稍慢（多一跳中转） | 更快（直传） |
| **隐私** | 文件经过云中继服务器 | 文件不经过第三方 |
| **可用性** | 依赖云中继服务 | 仅依赖微信 API |
| **适用场景** | 快速上手、无自己公众号 | 有自己公众号、重视隐私/速度 |

> 当同时配置了 AppID/AppSecret 时，客户端优先使用**方式二（直传）**。未配置时自动回退到**方式一（服务端中转）**。

### 支持的媒体类型

微信公众号客服消息接口[仅支持以下媒体类型](https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Service_Center_messages.html)：

| 文件类型 | 扩展名 | 发送方式 | 说明 |
|---------|--------|---------|------|
| 图片 | .jpg .jpeg .png .gif .bmp | 图片消息 | 直接发送，用户可查看/保存 |
| 语音 | .amr .mp3 .speex | 语音消息 | 直接发送 |
| 视频 | .mp4 | 视频消息 | 直接发送 |
| 其他 | .md .txt .pdf .docx 等 | ⚠️ 文本预览 | 截取前 500 字以文本消息发送 |

与企业微信不同，**公众号 API 没有通用的 `file` 类型**。企业微信的应用消息接口支持发送任意文件附件（`msgtype: file`），但公众号客服消息接口不提供此能力，这是微信平台本身的限制。

对于不支持的文件类型（如 .md、.pdf、.docx），lingti-bot 会读取文件内容并以文本消息形式发送预览。由于微信文本消息有字数限制，内容会被截取至前 500 字并标注"内容过长，已截断"。这个行为在两种方式下是一致的。

详细配置：[微信公众号接入指南](wechat-integration.md)

## 工作原理

1. 用户发送类似"把桌面上的 a.png 发给我"的消息
2. AI 调用 `file_send` 工具，指定文件路径和媒体类型
3. relay 客户端根据平台类型和配置选择发送方式：
   - **企业微信**：调用 WeCom 临时素材上传 API → 发送应用消息
   - **微信公众号（方式二：有 AppID）**：客户端直接上传素材 → 通过客服消息接口发送
   - **微信公众号（方式一：无 AppID）**：base64 编码文件 → 通过 webhook 发送到云中继 → 服务端上传并发送
   - **不支持的文件类型**：读取文件内容 → 以文本消息发送预览（截取前 500 字）

## 配置参数

| 参数 | 环境变量 | 平台 | 说明 |
|------|---------|------|------|
| `--wechat-app-id` | `WECHAT_APP_ID` | 微信公众号 | 公众号 AppID（可选，用于客户端直传） |
| `--wechat-app-secret` | `WECHAT_APP_SECRET` | 微信公众号 | 公众号 AppSecret（可选，用于客户端直传） |
| `--wecom-corp-id` | `WECOM_CORP_ID` | 企业微信 | 企业 ID |
| `--wecom-secret` | `WECOM_SECRET` | 企业微信 | 应用 Secret |
