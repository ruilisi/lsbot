[English](./README_EN.md) | 中文

---

# lingti-bot (灵小缇 [bot.lingti.com](https://bot.lingti.com))

> 🐕⚡「**极简至上 效率为王 秒级接入 一链即用**」的 AI Bot

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/官网-cli.lingti.com-blue?style=flat)](https://bot.lingti.com)

> 📖 **文档网站：[bot.lingti.com/docs](https://bot.lingti.com/docs)** — 完整文档、使用示例、接入教程、API 参考

**灵小缇** 是一个集 **MCP Server**、**多平台消息网关**、**丰富工具集**、**智能对话**于一体的 AI Bot 平台。

**核心优势：**
- 🚀 **零依赖部署** — 单个 30MB 二进制文件，无需 Node.js/Python 运行时，**一行命令**安装即用
- ☁️ **[云中继](docs/cloud-relay.md)加持** — 无需公网服务器、域名备案、HTTPS 证书，5 分钟接入企业微信/微信公众号
- 🤖 **[浏览器自动化](docs/browser-automation.md)** — 内置完整 CDP 控制引擎，可接管已有 Chrome 窗口，快照-操作模式精准定位元素，无需 Puppeteer/Playwright/Node.js
- 🌐 **[社交平台自动化](docs/social-platform-automation.md)** — 通过 MCP + Chrome 浏览器能力，AI 代操作知乎、小红书等内容平台，自动发帖、评论、互动，解放运营双手
- 🛠️ **75+ MCP 工具** — 覆盖文件、Shell、系统、网络、日历、Git、GitHub 等全场景
- 🌏 **中国平台原生支持** — 钉钉、飞书、企业微信、微信公众号开箱即用
- 💬 **[内置 Web 聊天界面](#web-chat-ui)** — 无需任何客户端，浏览器直开，支持**多会话并行**，每个会话独立记忆，刷新不丢失
- 🔌 **嵌入式友好** — 可编译到 ARM/MIPS，轻松部署到树莓派、路由器、NAS
- 🧠 **多 AI 后端** — 集成 Claude、DeepSeek、Kimi、MiniMax、Gemini 等 [16 种 AI 服务](AI-PROVIDERS.md)，按需切换，支持[按平台/频道指定不同模型](#per-channel-model)
- 🔬 **Claude 深度思考** — 原生支持 Claude Extended Thinking API，`/think high` 即可启用真正的链式推理
- 🐳 **Docker 部署** — 提供 Dockerfile 和 docker-compose.yml，一键容器化部署
- 🩺 **健康诊断** — `lingti-bot doctor` 一键检查配置、连接、依赖，快速定位问题

支持企业微信、飞书、钉钉、Slack、Telegram、Discord、WhatsApp、LINE、Teams 等 [19 种聊天平台](docs/chat-platforms.md) 接入，以及内置**浏览器 Web 聊天界面**（多会话并行），既可通过**云中继 5 分钟秒接**，也可 [OpenClaw](docs/openclaw-reference.md) 式**传统自建部署**。查看 [开发路线图](docs/roadmap.md) 了解更多功能规划。

> 🐕⚡ **为什么叫"灵小缇"？** 灵缇犬（Greyhound）是世界上跑得最快的犬，以敏捷、忠诚著称。灵小缇同样敏捷高效，是你忠实的 AI 助手。

## 快速开始

**两步启动，无需平台账号，无需公网服务器：**

**第一步：安装**

```bash
# macOS / Linux / WSL
curl -fsSL https://files.lingti.com/install-bot.sh | bash

# Windows (PowerShell)
irm https://files.lingti.com/install-bot.ps1 | iex
```

**第二步：运行 relay，指定 AI 服务商和密钥**

```bash
lingti-bot relay --provider deepseek --api-key sk-xxx
```

启动后自动输出你的专属 Bot 页面：

```
Your bot page: https://bot.lingti.com/bots/xxx
```

打开链接即可在浏览器中与 Bot 对话 — 仅需一条命令，无需配置任何消息平台。

> 支持的 `--provider`：`deepseek`、`claude`、`kimi`、`minimax`、`gemini`、`openai` 等，详见 [AI-PROVIDERS.md](AI-PROVIDERS.md)。

---

### 进阶：接入消息平台（企业微信、飞书等）

如需接入企业微信、飞书、钉钉等平台，追加对应平台参数即可：

```bash
# 企业微信
lingti-bot relay --platform wecom \
  --wecom-corp-id ww... --wecom-agent-id 1000002 \
  --wecom-secret xxx --wecom-token xxx --wecom-aes-key xxx \
  --provider deepseek --api-key sk-xxx

# 飞书
lingti-bot relay --platform feishu \
  --feishu-app-id cli_xxx --feishu-app-secret xxx \
  --provider claude --api-key sk-ant-xxx
```

详见 [云中继文档](docs/cloud-relay.md)。

### 安装方式

## 样例

> 📺 **完整演示、截图和使用示例，请访问：[bot.lingti.com/docs/examples](https://bot.lingti.com/docs/examples)**

主要功能演示：

- **[内置 Web 聊天界面](https://bot.lingti.com/docs/examples#web-chat-ui)** — 多会话并行，零配置，`--webapp-port 8080` 一键启用
- **[定时任务](https://bot.lingti.com/docs/examples#cron-jobs)** — 用自然语言创建 Cron Job，AI 自动生成内容
- **[企业微信文件助手](https://bot.lingti.com/docs/examples#wecom-file)** — 自然语言管理和传输文件
- **[社交平台自动化](https://bot.lingti.com/docs/examples#social-automation)** — AI 代操作知乎、小红书等内容平台
- **[浏览器自动化](https://bot.lingti.com/docs/examples#browser-automation)** — 自然语言驾驭 Chrome，无需 Puppeteer/Node.js

---

## Agents 与 Channels

### 什么是 Agent？

**Agent** 是 lingti-bot 中的 AI 实例单元。每个 Agent 拥有独立的：

- **AI 配置** — 可指定不同的 provider、model、API Key，或继承全局配置
- **系统提示词（Instructions）** — 定义 Agent 的角色、风格、行为边界
- **工作目录（Workspace）** — Agent 读写文件的默认根目录，天然沙箱隔离

典型用途：`work-assistant`（工作助手）、`coder`（专注代码）、`writer`（内容写作）……每个 Agent 可以有完全不同的人格和能力边界。

### 什么是 Channel？

**Channel** 是消息的来源/目标通道，对应一个具体的聊天平台接入点（如某个企业微信应用、某个 Slack Bot、某个 Telegram Bot）。你可以为不同 Channel 绑定不同的 Agent，实现"同一个 lingti-bot 进程，多个平台、多套 AI 人格"的效果。

### 管理 Agents

```bash
# 交互式添加 Agent
lingti-bot agents add

# 列出所有 Agent
lingti-bot agents list

# 查看某个 Agent 详情
lingti-bot agents info <name>

# 删除 Agent
lingti-bot agents remove <name>
```

**添加 Agent 示例：**

```
$ lingti-bot agents add

  Agent name — a short nickname you choose, e.g. "mybot", "work-assistant"
  Agent name:
  > lingti

  AI Provider:
    1. (inherit from base config)
    2. deepseek     (recommended)
    3. qwen         (tongyi qianwen)
    4. claude       (Anthropic)
    ...
  Choice [1]: claude
  Model (blank = inherit):
  > claude-sonnet-4-20250514
  API Key (blank = inherit):
  > sk-ant-xxx
  Instructions (text or file path, optional):
  >
  Mark as default agent? [y/N]: y
  Workspace directory (default: ~/.lingti/agents/lingti):
  >
Agent "lingti" added (workspace: ~/.lingti/agents/lingti)
```

### 管理 Channels

```bash
# 交互式添加 Channel 凭证（保存到 ~/.lingti.yaml）
lingti-bot channels add

# 列出所有已配置的 Channel
lingti-bot channels list

# 删除 Channel
lingti-bot channels remove <name>
```

### 添加 Channel 后如何启动？

Channel 凭证保存到 `~/.lingti.yaml` 后，根据平台类型选择启动命令：

**`gateway` 模式**（适用于大多数平台：Telegram、Discord、Slack、钉钉、飞书等）

`gateway` 自动读取配置文件中所有已配置的平台，一条命令全部启动：

```bash
lingti-bot gateway
```

**`relay` 模式**（仅适用于需要云中继的平台：企业微信、微信公众号、飞书、Slack）

`relay` 通过云中继服务器转发消息，需用 `--platform` 指定平台：

```bash
lingti-bot relay --platform wecom    # 企业微信
lingti-bot relay --platform feishu   # 飞书
lingti-bot relay --platform wechat   # 微信公众号
```

> **一句话区别：**
> - `gateway` — 本地直连，支持所有平台，凭证来自配置文件，一条命令启动全部
> - `relay` — 云中继，仅支持 wecom/feishu/wechat/slack，无需公网服务器

---

## 为什么选择 lingti-bot？

### lingti-bot vs OpenClaw

|  | **lingti-bot** | **OpenClaw** |
|--|----------------|--------------|
| **语言** | 纯 Go 实现 | Node.js |
| **运行依赖** | 无（单一二进制） | 需要 Node.js 运行时 |
| **分发方式** | 单个可执行文件，复制即用 | npm 安装，依赖 node_modules |
| **嵌入式设备** | ✅ 可轻松部署到 ARM/MIPS 等小型设备 | ❌ 需要 Node.js 环境 |
| **安装大小** | ~15MB 单文件 | 100MB+ (含 node_modules) |
| **输出风格** | 纯文本，无彩色 | 彩色输出 |
| **设计哲学** | 极简主义，够用就好 | 功能丰富，灵活优先 |
| **Docker 部署** | ✅ 多阶段构建，~15MB 镜像 | ✅ Docker 支持 |
| **Extended Thinking** | ✅ Claude 原生 API | ✅ 支持 |
| **按平台模型切换** | ✅ 配置文件声明式 | ✅ 支持 |
| **健康诊断** | ✅ `doctor` 命令 | ✅ 支持 |
| **中国平台** | 原生支持飞书/企微/钉钉 | 需自行集成 |
| **云中继** | ✅ 免自建服务器，秒级接入微信/企微 | ❌ 需自建 Web 服务 |

> 详细功能对比请参考：[OpenClaw vs lingti-bot 技术特性对比](docs/openclaw-feature-comparison.md)

**为什么选择纯 Go + 纯文本输出？**

> *"Simplicity is the ultimate sophistication."* — Leonardo da Vinci

lingti-bot 将**简洁性**作为最高设计原则：

1. **零依赖部署** — 单一二进制，`scp` 到任何机器即可运行，无需安装 Node.js、Python 或其他运行时
2. **嵌入式友好** — 可编译到 ARM、MIPS 等架构，轻松部署到树莓派、路由器、NAS 等小型设备
3. **纯文本输出** — 不使用彩色终端输出，避免引入额外的渲染库或终端兼容性问题
4. **代码克制** — 每一行代码都有明确的存在理由，拒绝过度设计
5. **云中继加持** — 无需自建 Web 服务器，通过云中继秒级完成微信公众号、企业微信的回调验证，Bot 即刻上线

```bash
# 克隆即编译，编译即运行
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot && make
./dist/lingti-bot gateway --provider deepseek --api-key sk-xxx
```

### 单一二进制

```bash
# 编译
make build

# 即可使用
./dist/lingti-bot serve
```

无需数据库，无需云服务。也支持 [Docker 部署](#docker-部署)。

### 本地优先

所有功能都在本地运行，数据不会上传到云端。你的文件、日历、进程信息都安全地保留在本地。

### 跨平台支持

核心功能支持 macOS、Linux、Windows。macOS 用户可享受日历、提醒事项、备忘录、音乐控制等原生功能。

**支持的目标平台：**

| 平台 | 架构 | 编译命令 |
|------|------|----------|
| macOS | ARM64 (Apple Silicon) | `make darwin-arm64` |
| macOS | AMD64 (Intel) | `make darwin-amd64` |
| Linux | AMD64 | `make linux-amd64` |
| Linux | ARM64 | `make linux-arm64` |
| Linux | ARMv7 (树莓派等) | `make linux-arm` |
| Windows | AMD64 | `make windows-amd64` |

## 功能概览

### MCP Server — 标准协议，无缝集成

灵小缇实现了完整的 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 协议，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

| 客户端 | 状态 | 说明 |
|--------|------|------|
| Claude Desktop | ✅ | Anthropic 官方桌面客户端 |
| Cursor | ✅ | AI 代码编辑器 |
| Windsurf | ✅ | Codeium 的 AI IDE |
| 其他 MCP 客户端 | ✅ | 任何实现 MCP 协议的应用 |

**特点：** 无需额外配置、无需数据库、无需 Docker、无需云服务，单一二进制文件即可运行。

### 多平台消息网关 — 企业 IM 秒接入

支持国内外主流企业消息平台，让团队在熟悉的工具中直接与 AI 对话。

| 平台 | 协议 | 接入方式 | 文件发送 | 状态 |
|------|------|----------|---------|------|
| **企业微信** | 回调 API | 云中继 / 自建 | ✅ 全格式 | ✅ |
| **微信公众号** | 云中继 | 10秒接入 | ✅ 图片/语音/视频 | ✅ |
| **钉钉** | Stream Mode | 一键接入 | 🔜 计划中 | ✅ |
| **飞书/Lark** | WebSocket | 一键接入 | 🔜 计划中 | ✅ |
| **Slack** | Socket Mode | 一键接入 | 🔜 计划中 | ✅ |
| **Telegram** | Bot API | 一键接入 | 🔜 计划中 | ✅ |
| **Discord** | Gateway | 一键接入 | 🔜 计划中 | ✅ |
| **WhatsApp** | Webhook + Graph API | 自建 | 🔜 计划中 | ✅ |
| **LINE** | Webhook + Push API | 自建 | 🔜 计划中 | ✅ |
| **Microsoft Teams** | Bot Framework | 自建 | 🔜 计划中 | ✅ |
| **Matrix / Element** | HTTP Sync | 自建 | 🔜 计划中 | ✅ |
| **Google Chat** | Webhook + REST | 自建 | 🔜 计划中 | ✅ |
| **Mattermost** | WebSocket + REST | 自建 | 🔜 计划中 | ✅ |
| **iMessage** | BlueBubbles | 自建 | 🔜 计划中 | ✅ |
| **Signal** | signal-cli REST | 自建 | 🔜 计划中 | ✅ |
| **Twitch** | IRC | 自建 | — | ✅ |
| **NOSTR** | WebSocket Relays | 自建 | 🔜 计划中 | ✅ |
| **Zalo** | Webhook + REST | 自建 | 🔜 计划中 | ✅ |
| **Nextcloud Talk** | HTTP Polling | 自建 | 🔜 计划中 | ✅ |
| **Web 聊天界面** | WebSocket | `--webapp-port` | — | ✅ |

> 文件发送详情（配置方法、支持的文件类型、限制）：[文件发送指南](docs/file-sending.md)

> 完整列表（含配置参数、环境变量）：[聊天平台列表](docs/chat-platforms.md)

**云中继优势：** 无需公网服务器、无需域名备案、无需 HTTPS 证书、无需防火墙配置，5 分钟完成接入。

### MCP 工具集 — 75+ 本地系统工具

覆盖日常工作的方方面面，让 AI 成为你的全能助手。

| 分类 | 工具数 | 功能 |
|------|--------|------|
| **文件操作** | 9 | 读写、搜索、整理、批量删除、废纸篓 |
| **Shell 命令** | 2 | 命令执行、路径查找 |
| **系统信息** | 4 | CPU、内存、磁盘、环境变量 |
| **进程管理** | 3 | 列表、详情、终止 |
| **网络工具** | 4 | 接口、连接、Ping、DNS |
| **日历** | 6 | 查看、创建、搜索、删除日程 (macOS) |
| **提醒事项** | 5 | 列表、添加、完成、删除 (macOS) |
| **备忘录** | 6 | 文件夹、列表、读取、创建、搜索、删除 (macOS) |
| **天气** | 2 | 当前天气、多日预报 |
| **网页搜索** | 2 | DuckDuckGo 搜索、网页内容获取 |
| **剪贴板** | 2 | 读写剪贴板 |
| **截图** | 1 | 屏幕截图 |
| **系统通知** | 1 | 发送桌面通知 |
| **音乐控制** | 7 | 播放、暂停、切歌、音量、搜索 (macOS) |
| **Git** | 4 | 状态、日志、差异、分支 |
| **GitHub** | 6 | PR 列表/详情、Issue 管理、仓库信息 |
| **[浏览器自动化](docs/browser-automation.md)** | 14 | CDP 控制引擎，可接管已有 Chrome，快照定位、点击/输入/JS、多标签页、截图 |
| **定时任务** | 5 | 创建、列表、删除、暂停、恢复计划任务 |

### 定时任务 — 自动化你的工作流

使用标准 Cron 表达式调度周期性任务，支持两种模式：

| | **AI 智能任务** (`prompt`) | **静态消息** (`message`) |
|--|---------------------------|------------------------|
| **内容** | 每次触发生成全新内容 | 每次发送相同文本 |
| **工具调用** | 可调用 web_search、天气、日历等所有工具 | 无 |
| **适用场景** | 新闻摘要、每日简报、随机鸡汤、学习提醒 | 固定提醒、打卡通知 |
| **示例** | "搜索最新AI新闻整理摘要" | "该喝水了！" |

**AI 智能任务** — 每次触发运行完整 AI 对话，内容永远不重复：

```
"每小时43分发一段鸡汤激励我写代码"
"每天早上9点搜索AI新闻发给我摘要"
"每天中午教我一个Go语言技巧"
```

**静态消息** — 每次发送固定文本：

```
"每天早上9点提醒我开站会"
"每小时提醒我喝水"
```

> 详细文档（完整示例、Cron 表达式、管理命令）：[定时任务指南](docs/cron-jobs.md)

### 智能对话 — 多轮记忆，自然交流

支持多轮对话记忆，能够记住之前的对话内容，实现连续自然的交流体验。

| 特性 | 说明 |
|------|------|
| **上下文记忆** | 每个用户独立的对话上下文，最近 50 条消息 |
| **自动过期** | 对话 60 分钟无活动后自动清除 |
| **多 AI 后端** | [16 种 AI 服务](AI-PROVIDERS.md)按需切换 |
| **对话管理** | `/new`、`/reset`、`新对话` 命令重置对话 |

### 健康诊断 — 一键排查配置问题

```bash
lingti-bot doctor
```

```
lingti-bot doctor
=================
OS: darwin/arm64, Go: go1.24.0

Checks:
  ✓ Config file (~/.lingti.yaml) — loaded
  ✓ AI API key — set (sk-ant-a..., provider: claude)
  ✓ AI provider connectivity — reachable (HTTP 200)
  ✓ Platform credentials — wecom, telegram
  ✓ Binary: gh — found
  ✗ Binary: chrome — not found in PATH
  ✓ Binary: claude — found
  ✓ MCP server: chrome — command "npx" available
  ✓ Temp directory — writable

8 passed, 1 failed
```

检查项包括：配置文件、API 密钥、AI 连接、平台凭证、必需工具、CDP 连接、MCP 服务器、磁盘空间。

### <a id="per-channel-model"></a>按平台/频道模型切换

为不同平台或频道配置不同的 AI 模型，在 `~/.lingti.yaml` 中声明：

```yaml
ai:
  provider: deepseek
  api_key: sk-xxx
  model: deepseek-chat

  # 按平台/频道覆盖
  overrides:
    - platform: telegram
      provider: claude
      api_key: sk-ant-xxx
      model: claude-sonnet-4-20250514
    - platform: discord
      provider: openai
      api_key: sk-xxx
      model: gpt-4o
    - platform: slack
      channel_id: C12345
      provider: claude
      api_key: sk-ant-xxx
```

匹配规则：先匹配 `platform + channel_id`（精确），再匹配 `platform`（宽泛），最后使用默认配置。

### Claude 深度思考 — 原生 Extended Thinking API

对 Claude 模型启用真正的链式推理（非提示词模拟），通过会话命令切换：

| 命令 | 模式 | 思考 Token 预算 |
|------|------|-----------------|
| `/think off` | 关闭 | 0 |
| `/think low` | 简单 | 1,024 |
| `/think medium` | 中等（默认） | 4,096 |
| `/think high` | 深度 | 16,384 |

- **Claude 模型**：自动使用 Anthropic Thinking API，模型在回复前进行真正的内部推理
- **其他模型**：降级为系统提示词引导的思考模式

### Docker 部署

```bash
# 构建镜像
docker build -t lingti-bot .

# 运行
docker run -e AI_API_KEY=sk-xxx -e TELEGRAM_BOT_TOKEN=xxx lingti-bot

# 使用 docker-compose（推荐）
docker compose up -d
```

`docker-compose.yml` 支持配置文件挂载、环境变量注入，以及可选的 Chrome 容器用于浏览器自动化。

### Skills — 模块化能力扩展

Skills 是模块化的能力包，教会 lingti-bot 如何使用外部工具。每个 Skill 是一个包含 `SKILL.md` 文件的目录，通过 YAML frontmatter 声明依赖和元数据，通过 Markdown 正文提供 AI 指令。

```bash
# 列出所有已发现的 Skills
lingti-bot skills

# 查看就绪状态
lingti-bot skills check

# 查看某个 Skill 的详细信息
lingti-bot skills info github
```

内置 8 个 Skills：Discord、GitHub、Slack、Peekaboo（macOS UI 自动化）、Tmux、天气、1Password、Obsidian。支持用户自定义和项目级 Skills。

详细文档：[Skills 指南](docs/skills.md)

### 功能速览表

| 模块 | 说明 | 特点 |
|------|------|------|
| **MCP Server** | 标准 MCP 协议服务器 | 兼容 Claude Desktop、Cursor、Windsurf 等所有 MCP 客户端 |
| **多平台消息网关** | [19 种聊天平台](docs/chat-platforms.md) | 微信公众号、企业微信、Slack、飞书一键接入，支持云中继 |
| **[Web 聊天界面](#web-chat-ui)** | 内置浏览器 UI | 多会话并行，独立记忆隔离，Markdown 渲染，`--webapp-port` 一键启用 |
| **MCP 工具集** | 75+ 本地系统工具 | 文件、Shell、系统、网络、日历、Git、GitHub 等全覆盖 |
| **Skills** | 模块化能力扩展 | 8 个内置 Skill，支持自定义和项目级扩展 |
| **智能对话** | 多轮对话与记忆 | 上下文记忆、[16 种 AI 后端](AI-PROVIDERS.md)、按平台模型切换 |
| **深度思考** | Claude Extended Thinking | 原生 Thinking API，4 级思考深度 |
| **健康诊断** | `doctor` 命令 | 一键检查配置、连接、依赖 |
| **Docker 部署** | 容器化 | 多阶段构建，docker-compose 支持 |

## 云中继：零门槛接入企业消息平台

> **告别公网服务器、告别复杂配置，让 AI Bot 接入像配置 Wi-Fi 一样简单**

传统接入企业微信等平台需要：公网服务器 → 域名备案 → HTTPS 证书 → 防火墙配置 → 回调服务开发...

**lingti-bot 云中继** 将这一切简化为 3 步：

```bash
# 步骤 1: 安装
# macOS / Linux / WSL:
curl -fsSL https://files.lingti.com/install-bot.sh | bash
# Windows (PowerShell):
irm https://cli.lingti.com/install.ps1 -OutFile install.ps1; .\install.ps1 -Bot

# 步骤 2: 配置企业可信IP（应用管理 → 找到应用 → 企业可信IP → 添加 106.52.166.51）

# 步骤 3: 一条命令搞定验证和消息处理
lingti-bot relay --platform wecom \
  --wecom-corp-id ... --wecom-token ... --wecom-aes-key ... \
  --provider deepseek --api-key sk-xxx

# 然后去企业微信后台配置回调 URL: https://bot.lingti.com/wecom
```

**工作原理：**

```
企业微信(用户消息) --> bot.lingti.com(云中继) --WebSocket--> lingti-bot(本地AI处理)
```

**优势对比：**

| | 传统方案 | 云中继方案 |
|---|---|---|
| 公网服务器 | ✅ 需要 | ❌ 不需要 |
| 域名/备案 | ✅ 需要 | ❌ 不需要 |
| HTTPS证书 | ✅ 需要 | ❌ 不需要 |
| 回调服务开发 | ✅ 需要 | ❌ 不需要 |
| 接入时间 | 数天 | **5分钟** |
| AI处理位置 | 服务器 | **本地** |
| 数据安全 | 云端存储 | **本地处理** |

> 📖 **深入了解云中继的架构原理、协议规范和完整消息流程，请阅读 [云中继技术方案详解](docs/cloud-relay.md)**
>
> 详细对比请参考：[lingti-bot vs OpenClaw：简化 AI 集成的努力](docs/vs-openclaw-integration.md)

### 微信公众号一键接入

微信搜索公众号「**灵缇小秘**」，关注后发送任意消息获取接入教程，10秒将lingti-bot接入微信。
详细教程请参考：[微信公众号接入指南](docs/wechat-integration.md)
### 飞书接入

- 飞书商店应用正在上架流程中，目前可通过自建应用实现绑定。教程请参考：[飞书集成指南](https://github.com/ruilisi/lingti-bot/blob/master/docs/feishu-integration.md)

### 企业微信接入

通过**云中继模式**，无需公网服务器即可接入企业微信：

```bash
# 1. 先去企业微信后台配置企业可信IP
#    应用管理 → 找到应用 → 企业可信IP → 添加: 106.52.166.51

# 2. 一条命令搞定验证和消息处理
lingti-bot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY

# 3. 去企业微信后台配置回调 URL: https://bot.lingti.com/wecom
#    保存配置后验证自动完成，消息立即可以处理
```

详细教程请参考：[企业微信集成指南](docs/wecom-integration.md) | [云中继技术方案详解](docs/cloud-relay.md)

### 钉钉接入

使用 **Stream 模式**，无需公网服务器即可接入钉钉机器人：

```bash
# 一条命令搞定
lingti-bot gateway \
  --dingtalk-client-id YOUR_APP_KEY \
  --dingtalk-client-secret YOUR_APP_SECRET \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

**配置步骤：**
1. 登录 [钉钉开放平台](https://open.dingtalk.com/)，创建企业内部应用
2. 在应用详情页获取 AppKey (ClientID) 和 AppSecret (ClientSecret)
3. 开启机器人功能，配置消息接收模式为 **Stream 模式**
4. 运行上述命令即可

## Sponsors

- **[灵缇游戏加速](https://game.lingti.com)** - PC/Mac/iOS/Android 全平台游戏加速、热点加速、AI 及学术资源定向加速，And More
- **[灵缇路由](https://router.lingti.com)** - 您的路由管家、网游电竞专家

## lingti-cli 生态

**lingti-bot** 是 **lingti-cli** 五位一体平台的核心开源组件。

我们正在打造 **AI 时代开发者与知识工作者的终极效率平台**：

| 模块 | 定位 | 说明 |
|------|------|------|
| **CLI** | 操控总台 | 统一入口，如同操作系统的引导程序 |
| **Net** | 全球网络 | 跨洲 200Mbps 加速，畅享全球 AI 服务 |
| **Token** | 数字员工 | Token 即代码，代码即生产力 |
| **Bot** | 助理管理 | 数字员工接入与管理，简单到极致 ← *本项目* |
| **Code** | 开发环境 | Terminal 回归舞台中央，极致输入效率 |

> **为什么是 bot.lingti.com 而不是 bot.lingti.com？**
>
> 因为 Bot 是 CLI 生态的一部分。IDE 正在消亡，纯粹的 Terminal 界面正在回归。未来的生产力工具，将围绕 CLI 重新构建。

**联系我们 / 加入我们**

<table>
  <tr>
    <th align="center" width="56%">邮件联系</th>
    <th align="center" width="44%">扫码加群</th>
  </tr>
  <tr>
    <td width="56%">
      无论您是追求极致效率的顶尖开发者、关注 AI 时代生产力变革的投资人，还是想成为 Sponsor，
      欢迎联系：
      <code>jiefeng@ruc.edu.cn</code>
      /
      <code>jiefeng.hopkins@gmail.com</code>
    </td>
    <td width="44%" align="center">
      <img src="https://lingti-1302055788.cos.ap-guangzhou.myqcloud.com/contact_me_qr-2.png" alt="扫码加群" width="230" />
    </td>
  </tr>
</table>

---

```
                              lingti-bot
    +---------------+    +---------------+    +---------------+
    |  MCP Server   |    |   Message     |    |    Agent      |
    |   (stdio)     |    |   Gateway     |    |   (Claude)    |
    +-------+-------+    +-------+-------+    +-------+-------+
            |                    |                    |
            +--------------------+--------------------+
                                 |
                                 v
                       +-------------------+
                       |    MCP Tools      |
                       | Files, Shell, Net |
                       | System, Calendar  |
                       | Browser Automation|
                       +-------------------+
                                 |
            +--------------------+--------------------+
            |                                         |
            v                                         v
    +---------------+                       +------------------+
    | Claude Desktop|                       | Slack / Feishu   |
    | Cursor, etc.  |                       | Messaging Apps   |
    +---------------+                       +------------------+
```

---

## MCP Server

灵小缇作为标准 MCP (Model Context Protocol) 服务器，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

### 支持的客户端

- **Claude Desktop** - Anthropic 官方桌面客户端
- **Cursor** - AI 代码编辑器
- **其他 MCP 客户端** - 任何实现 MCP 协议的应用

### 快速配置

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`)：

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

**Cursor** (`.cursor/mcp.json`)：

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

就这么简单！重启客户端后，AI 助手即可使用所有 lingti-bot 提供的工具。

### 特点

- **无需额外配置** - 一个二进制文件，两行配置
- **无需数据库** - 无外部依赖
- **无需 Docker** - 单一静态二进制
- **无需云服务** - 完全本地运行

---

## 多平台消息网关

灵小缇支持多种企业消息平台，让你的团队在熟悉的工具中直接与 AI 对话。

### 支持的平台

| 平台 | 协议 | 状态 |
|------|------|------|
| **企业微信** | 回调 API | ✅ 已支持 |
| **微信公众号** | 云中继 | ✅ 已支持 |
| **钉钉** | Stream Mode | ✅ 已支持 |
| **飞书/Lark** | WebSocket | ✅ 已支持 |
| **Slack** | Socket Mode | ✅ 已支持 |
| **Telegram** | Bot API | ✅ 已支持 |
| **Discord** | Gateway | ✅ 已支持 |
| **WhatsApp** | Webhook + Graph API | ✅ 已支持 |
| **LINE** | Webhook + Push API | ✅ 已支持 |
| **Microsoft Teams** | Bot Framework | ✅ 已支持 |
| **Matrix / Element** | HTTP Sync | ✅ 已支持 |
| **Google Chat** | Webhook + REST | ✅ 已支持 |
| **Mattermost** | WebSocket + REST | ✅ 已支持 |
| **iMessage** | BlueBubbles | ✅ 已支持 |
| **Signal** | signal-cli REST | ✅ 已支持 |
| **Twitch** | IRC | ✅ 已支持 |
| **NOSTR** | WebSocket Relays | ✅ 已支持 |
| **Zalo** | Webhook + REST | ✅ 已支持 |
| **Nextcloud Talk** | HTTP Polling | ✅ 已支持 |
| **Web 聊天界面** | WebSocket (内置) | ✅ 已支持 |

> 完整列表：[聊天平台列表](docs/chat-platforms.md)

### 一键接入

灵小缇提供 **1 分钟内一键接入**方式，无需复杂配置：

```bash
# 设置 API 密钥
export ANTHROPIC_API_KEY="sk-ant-your-api-key"

# Slack 一键接入
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."

# 飞书一键接入
export FEISHU_APP_ID="cli_..."
export FEISHU_APP_SECRET="..."

# 启动网关
./lingti-bot gateway
```

### 多 AI 后端

支持 **16 种 AI 服务**，涵盖国内外主流大模型平台，按需切换：

| # | Provider | 名称 | 默认模型 |
|---|----------|------|----------|
| 1 | `deepseek` | DeepSeek (推荐) | `deepseek-chat` |
| 2 | `qwen` | 通义千问 (Qwen) | `qwen-plus` |
| 3 | `claude` | Claude (Anthropic) | `claude-sonnet-4-20250514` |
| 4 | `kimi` | Kimi / 月之暗面 | `moonshot-v1-8k` |
| 5 | `minimax` | MiniMax / 海螺 AI | `MiniMax-Text-01` |
| 6 | `doubao` | 豆包 (ByteDance) | `doubao-pro-32k` |
| 7 | `zhipu` | 智谱 GLM | `glm-4-flash` |
| 8 | `openai` | OpenAI (GPT) | `gpt-4o` |
| 9 | `gemini` | Gemini (Google) | `gemini-2.0-flash` |
| 10 | `yi` | 零一万物 (Yi) | `yi-large` |
| 11 | `stepfun` | 阶跃星辰 (StepFun) | `step-2-16k` |
| 12 | `baichuan` | 百川智能 (Baichuan) | `Baichuan4` |
| 13 | `spark` | 讯飞星火 (iFlytek) | `generalv3.5` |
| 14 | `siliconflow` | 硅基流动 (aggregator) | `Qwen/Qwen2.5-72B-Instruct` |
| 15 | `grok` | Grok (xAI) | `grok-2-latest` |
| 16 | `ollama` | Ollama (本地) | `llama3.2` |

> 完整列表（含 API Key 获取链接、别名）：[AI 服务列表](AI-PROVIDERS.md)

```bash
# 使用命令行参数指定
lingti-bot gateway --provider qwen --api-key "sk-xxx" --model "qwen-plus"

# 覆盖默认模型
lingti-bot relay --provider openai --api-key "sk-xxx" --model "gpt-4o-mini"
```

### 详细文档

- [配置优先级](CONFIGURATION.md) - 命令行参数 > 环境变量 > 配置文件的解析机制
- [AI 服务列表](AI-PROVIDERS.md) - 16 种 AI 服务详情、API Key 获取、别名
- [聊天平台列表](docs/chat-platforms.md) - 19 种聊天平台详情、配置参数、环境变量
- [命令行参考](docs/cli-reference.md) - 完整的命令行使用文档
- [Skills 指南](docs/skills.md) - Skills 系统详解：创建、发现、配置
- [Slack 集成指南](docs/slack-integration.md) - 完整的 Slack 应用配置教程
- [飞书集成指南](docs/feishu-integration.md) - 飞书/Lark 应用配置教程
- [企业微信集成指南](docs/wecom-integration.md) - 企业微信应用配置教程
- [文件发送指南](docs/file-sending.md) - 各平台文件传输能力、配置与限制
- [定时任务指南](docs/cron-jobs.md) - AI 智能任务 vs 静态消息、Cron 表达式、管理命令
- [浏览器自动化指南](docs/browser-automation.md) - CDP 引擎、接管已有 Chrome、14 个工具完整参考、典型场景与故障排除
- [浏览器 AI 操作规则](docs/browser-agent-rules.md) - AI agent 的所有浏览器行为规则：快照法则、搜索行为、弹窗处理、批量操作、连接模式、常见错误纠正
- [社交平台自动化指南](docs/social-platform-automation.md) - 知乎、小红书等内容平台的 AI 自动化运营
- [OpenClaw 技术特性对比](docs/openclaw-feature-comparison.md) - 详细功能差异分析

---

## MCP 工具集

灵小缇提供 **75+ MCP 工具**，覆盖日常工作的方方面面。包含全新的[浏览器自动化](docs/browser-automation.md)能力。

### 工具分类

| 分类 | 工具数 | 说明 |
|------|--------|------|
| 文件操作 | 9 | 读写、搜索、整理、废纸篓 |
| Shell 命令 | 2 | 命令执行、路径查找 |
| 系统信息 | 4 | CPU、内存、磁盘、环境变量 |
| 进程管理 | 3 | 列表、详情、终止 |
| 网络工具 | 4 | 接口、连接、Ping、DNS |
| 日历 (macOS) | 6 | 查看、创建、搜索、删除 |
| 提醒事项 (macOS) | 5 | 列表、添加、完成、删除 |
| 备忘录 (macOS) | 6 | 文件夹、列表、读取、创建、搜索、删除 |
| 天气 | 2 | 当前天气、预报 |
| 网页搜索 | 2 | DuckDuckGo 搜索、网页获取 |
| 剪贴板 | 2 | 读写剪贴板 |
| 截图 | 1 | 屏幕截图 |
| 系统通知 | 1 | 发送通知 |
| 音乐控制 (macOS) | 7 | 播放、暂停、切歌、音量 |
| Git | 4 | 状态、日志、差异、分支 |
| GitHub | 6 | PR、Issue、仓库信息 |
| [浏览器自动化](docs/browser-automation.md) | 14 | 快照、点击、输入、JS 执行、批量操作、截图、标签页 |

### 文件操作

| 工具 | 功能 |
|------|------|
| `file_read` | 读取文件内容 |
| `file_write` | 写入文件内容 |
| `file_list` | 列出目录内容 |
| `file_search` | 按模式搜索文件 |
| `file_info` | 获取文件详细信息 |
| `file_list_old` | 列出长时间未修改的文件 |
| `file_delete_old` | 删除长时间未修改的文件 |
| `file_delete_list` | 批量删除指定文件 |
| `file_trash` | 移动文件到废纸篓（macOS） |

### Shell 命令

| 工具 | 功能 |
|------|------|
| `shell_execute` | 执行 Shell 命令 |
| `shell_which` | 查找可执行文件路径 |

### 系统信息

| 工具 | 功能 |
|------|------|
| `system_info` | 获取系统信息（CPU、内存、OS） |
| `disk_usage` | 获取磁盘使用情况 |
| `env_get` | 获取环境变量 |
| `env_list` | 列出所有环境变量 |

### 进程管理

| 工具 | 功能 |
|------|------|
| `process_list` | 列出运行中的进程 |
| `process_info` | 获取进程详细信息 |
| `process_kill` | 终止进程 |

### 网络工具

| 工具 | 功能 |
|------|------|
| `network_interfaces` | 列出网络接口 |
| `network_connections` | 列出活动网络连接 |
| `network_ping` | TCP 连接测试 |
| `network_dns_lookup` | DNS 查询 |

### 日历（macOS）

| 工具 | 功能 |
|------|------|
| `calendar_today` | 获取今日日程 |
| `calendar_list_events` | 列出未来事件 |
| `calendar_create_event` | 创建日历事件 |
| `calendar_search` | 搜索日历事件 |
| `calendar_delete_event` | 删除日历事件 |
| `calendar_list_calendars` | 列出所有日历 |

### 提醒事项（macOS）

| 工具 | 功能 |
|------|------|
| `reminders_today` | 获取今日待办事项 |
| `reminders_add` | 添加新提醒 |
| `reminders_complete` | 标记提醒为已完成 |
| `reminders_delete` | 删除提醒 |
| `reminders_list_lists` | 列出所有提醒列表 |

### 备忘录（macOS）

| 工具 | 功能 |
|------|------|
| `notes_list_folders` | 列出备忘录文件夹 |
| `notes_list` | 列出备忘录 |
| `notes_read` | 读取备忘录内容 |
| `notes_create` | 创建新备忘录 |
| `notes_search` | 搜索备忘录 |
| `notes_delete` | 删除备忘录 |

### 天气

| 工具 | 功能 |
|------|------|
| `weather_current` | 获取当前天气 |
| `weather_forecast` | 获取天气预报 |

### 网页搜索

| 工具 | 功能 |
|------|------|
| `web_search` | DuckDuckGo 搜索 |
| `web_fetch` | 获取网页内容 |

### 剪贴板

| 工具 | 功能 |
|------|------|
| `clipboard_read` | 读取剪贴板内容 |
| `clipboard_write` | 写入剪贴板 |

### 系统通知

| 工具 | 功能 |
|------|------|
| `notification_send` | 发送系统通知 |

### 截图

| 工具 | 功能 |
|------|------|
| `screenshot` | 截取屏幕截图 |

### 音乐控制（macOS）

| 工具 | 功能 |
|------|------|
| `music_play` | 播放音乐 |
| `music_pause` | 暂停音乐 |
| `music_next` | 下一首 |
| `music_previous` | 上一首 |
| `music_now_playing` | 获取当前播放信息 |
| `music_volume` | 设置音量 |
| `music_search` | 搜索并播放音乐 |

### Git

| 工具 | 功能 |
|------|------|
| `git_status` | 查看仓库状态 |
| `git_log` | 查看提交日志 |
| `git_diff` | 查看文件差异 |
| `git_branch` | 查看分支信息 |

### GitHub

| 工具 | 功能 |
|------|------|
| `github_pr_list` | 列出 Pull Requests |
| `github_pr_view` | 查看 PR 详情 |
| `github_issue_list` | 列出 Issues |
| `github_issue_view` | 查看 Issue 详情 |
| `github_issue_create` | 创建新 Issue |
| `github_repo_view` | 查看仓库信息 |

### 浏览器自动化

基于 [go-rod](https://github.com/go-rod/rod) 的纯 Go 浏览器自动化引擎，采用**快照-操作（Snapshot-then-Act）**模式，可接管已有 Chrome 窗口，无需 Puppeteer/Playwright/Node.js。

> 📖 完整指南：[浏览器自动化文档](docs/browser-automation.md)

**连接已有 Chrome（推荐）：**

```yaml
# ~/.lingti.yaml
browser:
  cdp_url: "127.0.0.1:9222"   # Chrome 需以 --remote-debugging-port=9222 启动
  screen_size: "1920x1080"     # 或 "fullscreen"
```

**核心工作流：**

```
browser_snapshot  →  [1] textbox "搜索"  [2] button "搜索"  [3] link "登录"
browser_type ref=1 text="keyword"
browser_click ref=2
browser_snapshot  →  重新获取编号（页面变化后必须重新 snapshot）
```

**14 个工具完整列表：**

| 工具 | 功能 |
|------|------|
| `browser_start` | 启动新浏览器，或连接已有 Chrome（`cdp_url` 参数） |
| `browser_stop` | 关闭浏览器；CDP 连接模式下只断开，不关闭 Chrome |
| `browser_status` | 查看运行状态（是否 connected、页面数、当前 URL） |
| `browser_navigate` | 导航到 URL，自动等待页面加载，浏览器未启动时自动按优先级连接 |
| `browser_snapshot` | 获取无障碍树快照，为每个可交互元素分配数字 ref |
| `browser_screenshot` | 截图，支持视口截图或整页截图，`full_page=true` |
| `browser_click` | 按 ref 编号点击元素，自动滚动到可见位置 |
| `browser_type` | 按 ref 编号输入文本，`submit=true` 输入后按 Enter |
| `browser_press` | 按键（Enter、Tab、Escape、方向键、PageUp/Down 等） |
| `browser_execute_js` | 在当前页面执行任意 JavaScript，返回结果字符串 |
| `browser_click_all` | 按 CSS 选择器批量点击，支持间隔延迟和跳过条件 |
| `browser_tabs` | 列出所有标签页（target_id、url、title） |
| `browser_tab_open` | 打开新标签页，可指定初始 URL |
| `browser_tab_close` | 按 target_id 关闭标签页，或关闭当前活跃标签页 |

### 其他

| 工具 | 功能 |
|------|------|
| `open_url` | 在浏览器中打开 URL |

---

## 智能对话

灵小缇支持**多轮对话记忆**，能够记住之前的对话内容，实现连续自然的交流体验。

### 工作原理

- 每个用户在每个频道有独立的对话上下文
- 自动保存最近 **50 条消息**
- 对话 **60 分钟**无活动后自动过期
- 支持跨多轮对话的上下文理解

### 使用示例

```
用户：我叫小明，今年25岁
AI：你好小明！很高兴认识你。

用户：我叫什么名字？
AI：你叫小明。

用户：我多大了？
AI：你今年25岁。

用户：帮我创建一个日程，标题就用我的名字
AI：好的，我帮你创建了一个标题为"小明"的日程。
```

### 对话管理命令

| 命令 | 说明 |
|------|------|
| `/new` | 开始新对话，清除历史记忆 |
| `/reset` | 同上 |
| `/clear` | 同上 |
| `新对话` | 中文命令，开始新对话 |
| `清除历史` | 中文命令，清除对话历史 |

> **提示**：当你想让 AI "忘记"之前的内容重新开始时，只需发送 `/new` 即可。

---

## 快速开始

### 其他安装方式

**从源码编译**

```bash
git clone https://github.com/ruilisi/lingti-bot.git
cd lingti-bot
make build  # 或: make darwin-arm64 / make linux-amd64
```

**手动下载**

前往 [GitHub Releases](https://github.com/ruilisi/lingti-bot/releases) 下载对应平台的二进制文件。

### 使用方式

**方式一：MCP Server 模式**

配置 Claude Desktop 或 Cursor，详见 [MCP Server](#mcp-server) 章节。

**方式二：消息网关模式**

连接 Slack、飞书等平台，详见 [多平台消息网关](#多平台消息网关) 章节。

---

## 使用示例

> 📖 **完整使用示例（日历、文件、系统、浏览器、组合任务等）请访问：[bot.lingti.com/docs/examples](https://bot.lingti.com/docs/examples)**

配置完成后，你可以让 AI 助手执行日历管理、文件操作、系统信息查询、网页搜索、音乐控制、浏览器自动化等各类任务，支持自然语言组合调用多个工具。

---

## 项目结构

```
lingti-bot/
├── main.go                 # 程序入口
├── Makefile                # 构建脚本
├── Dockerfile              # 多阶段 Docker 构建
├── docker-compose.yml      # 容器编排
├── go.mod                  # Go 模块定义
│
├── cmd/                    # 命令行接口
│   ├── root.go             # 根命令
│   ├── serve.go            # MCP 服务器命令
│   ├── doctor.go           # 健康诊断命令
│   ├── service.go          # 系统服务管理
│   └── version.go          # 版本信息
│
├── internal/
│   ├── mcp/
│   │   └── server.go       # MCP 服务器实现
│   │
│   ├── browser/            # 浏览器自动化引擎
│   │   ├── browser.go      # 浏览器生命周期管理
│   │   ├── snapshot.go     # 无障碍树快照与 ref 映射
│   │   └── actions.go      # 元素交互（点击、输入、悬停）
│   │
│   ├── tools/              # MCP 工具实现
│   │   ├── filesystem.go   # 文件读写、列表、搜索
│   │   ├── shell.go        # Shell 命令执行
│   │   ├── system.go       # 系统信息、磁盘、环境变量
│   │   ├── process.go      # 进程列表、信息、终止
│   │   ├── network.go      # 网络接口、连接、DNS
│   │   ├── calendar.go     # macOS 日历集成
│   │   ├── filemanager.go  # 文件整理（清理旧文件）
│   │   ├── reminders.go    # macOS 提醒事项
│   │   ├── notes.go        # macOS 备忘录
│   │   ├── weather.go      # 天气查询（wttr.in）
│   │   ├── websearch.go    # 网页搜索和获取
│   │   ├── clipboard.go    # 剪贴板读写
│   │   ├── notification.go # 系统通知
│   │   ├── screenshot.go   # 屏幕截图
│   │   ├── browser.go      # 浏览器自动化工具（12个）
│   │   └── music.go        # 音乐控制（Spotify/Apple Music）
│   │
│   ├── router/
│   │   └── router.go       # 多平台消息路由器
│   │
│   ├── platforms/          # 消息平台集成
│   │   ├── slack/
│   │   │   └── slack.go    # Slack Socket Mode
│   │   └── feishu/
│   │       └── feishu.go   # 飞书 WebSocket
│   │
│   ├── agent/
│   │   ├── agent.go        # Agent 主逻辑与工具定义
│   │   ├── pool.go         # Agent 连接池（按平台模型切换）
│   │   ├── provider.go     # AI Provider 接口
│   │   ├── provider_claude.go  # Claude 实现（含 Extended Thinking）
│   │   ├── session.go      # 会话设置（思考级别）
│   │   ├── tools.go        # 工具执行
│   │   └── memory.go       # 会话记忆
│   │
│   └── service/
│       └── manager.go      # 系统服务管理
│
└── docs/                   # 文档
    ├── slack-integration.md    # Slack 集成指南
    ├── feishu-integration.md   # 飞书集成指南
    └── openclaw-reference.md   # 架构参考
```

---

## Make 目标

```bash
# 开发
make build          # 编译当前平台
make run            # 本地运行
make test           # 运行测试
make fmt            # 格式化代码
make lint           # 代码检查
make clean          # 清理构建产物
make version        # 显示版本

# 跨平台编译
make darwin-arm64   # macOS Apple Silicon
make darwin-amd64   # macOS Intel
make darwin-universal # macOS 通用二进制
make linux-amd64    # Linux x64
make linux-arm64    # Linux ARM64
make linux-all      # 所有 Linux 平台
make all            # 所有平台

# 服务管理
make install        # 安装为系统服务
make uninstall      # 卸载系统服务
make start          # 启动服务
make stop           # 停止服务
make status         # 查看服务状态

# macOS 签名
make codesign       # 代码签名（需要开发者证书）
```

---

## 命令行选项

### 全局选项

这些选项可用于所有命令，放在子命令之前使用。

| 选项 | 简写 | 说明 | 默认值 |
|------|------|------|--------|
| `--yes` | `-y` | **自动批准模式** - 跳过所有确认提示，直接执行操作 | `false` |
| `--debug` | - | **调试模式** - 启用详细日志和浏览器截图 | `false` |
| `--log <level>` | - | **日志级别** - silent, info, verbose, very-verbose | `info` |
| `--debug-dir <path>` | - | **调试目录** - 保存调试截图的路径 | `/tmp/lingti-bot` |

#### 自动批准模式 (`--yes`)

启用后，AI 将立即执行文件写入、删除、Shell 命令等操作，无需每次询问确认。

**适用场景：**
- ✅ 批量文件处理
- ✅ 代码生成和重构
- ✅ 文档自动更新
- ✅ CI/CD 自动化流程
- ✅ 信任环境下的快速操作

**不适用场景：**
- ❌ 生产环境服务器
- ❌ 共享系统
- ❌ 首次尝试新操作
- ❌ 涉及敏感数据

**使用示例：**

```bash
# 启用自动批准
lingti-bot --yes router --provider deepseek --api-key sk-xxx

# 简写形式
lingti-bot -y router --provider deepseek --api-key sk-xxx

# 结合调试模式
lingti-bot --yes --debug router --provider deepseek --api-key sk-xxx
```

**行为对比：**

```bash
# 不使用 --yes（默认）
用户：保存这个文件到 config.yaml
AI：  我已经准备好内容。是否确认保存到 config.yaml？
用户：是的
AI：  ✅ 已保存到 config.yaml

# 使用 --yes
用户：保存这个文件到 config.yaml
AI：  ✅ 已保存到 config.yaml (247 字节)
```

**安全提示：**
- 在 git 仓库中使用 `--yes` 最安全，可随时通过 `git diff` 查看变更
- 建议先在测试目录中尝试 `--yes` 模式
- 即使启用 `--yes`，危险操作（如 `rm -rf /`）仍会被拒绝

详细文档：
- [自动批准完整指南](docs/auto-approval.md)
- [快速参考](docs/auto-approval-quickref.md)

#### 调试模式 (`--debug`)

启用后自动设置日志级别为 `very-verbose`，并在浏览器操作出错时保存截图。

```bash
lingti-bot --debug router --provider deepseek --api-key sk-xxx
```

详细文档：
- [浏览器调试指南](docs/browser-debug.md)
- [快速参考](docs/browser-debug-quickref.md)

---

## 配置优先级

所有配置项按以下优先级解析：**命令行参数 > 环境变量 > 配置文件 (`~/.lingti.yaml`)**

```bash
# 配置文件已保存 provider: qwen，但命令行覆盖为 deepseek
lingti-bot relay --provider deepseek
```

完整的环境变量、命令行参数、配置文件对照表请参考：[配置优先级](CONFIGURATION.md)

---

## 数据文件

lingti-bot 在用户主目录下存储以下文件：

| 路径 | 说明 |
|------|------|
| `~/.lingti.yaml` | 配置文件（AI 服务、平台密钥等） |
| `~/.lingti.db` | SQLite 数据库（定时任务等持久化数据） |
| `~/.lingti.db-wal` | SQLite WAL 日志（自动管理，勿手动删除） |
| `~/.lingti.db-shm` | SQLite 共享内存（自动管理，勿手动删除） |
| `~/.lingti-bot/browser/` | 浏览器自动化的用户数据目录 |

---

## 安全注意事项

- lingti-bot 提供对本地系统的访问能力，请在可信环境中使用
- Shell 命令执行有基本的危险命令过滤，但仍需谨慎
- API 密钥等敏感信息请使用环境变量，不要提交到版本控制
- 生产环境建议使用专用服务账号运行

---

## 依赖

- [mcp-go](https://github.com/mark3labs/mcp-go) - MCP 协议 Go 实现
- [cobra](https://github.com/spf13/cobra) - CLI 框架
- [gopsutil](https://github.com/shirou/gopsutil) - 系统信息
- [slack-go](https://github.com/slack-go/slack) - Slack SDK
- [oapi-sdk-go](https://github.com/larksuite/oapi-sdk-go) - 飞书/Lark SDK
- [go-anthropic](https://github.com/liushuangls/go-anthropic) - Anthropic API 客户端

---

## 许可证

MIT License

---

## 贡献

欢迎提交 Issue 和 Pull Request！

---

## 开发环境

本项目完全在 **[lingti-code](https://cli.lingti.com/code)** 环境中编写完成。

### 关于 lingti-code

[lingti-code](https://github.com/ruilisi/lingti-code) 是一个一体化的 AI 就绪开发环境平台，基于 **Tmux + Neovim + Zsh** 构建，支持 macOS、Ubuntu 和 Docker 部署。

**核心组件：**

- **Shell** - ZSH + Prezto 框架，100+ 常用别名和函数，fasd 智能导航
- **Editor** - Neovim + SpaceVim 发行版，LSP 集成，GitHub Copilot 支持
- **Terminal** - Tmux 终端复用，vim 风格键绑定，会话管理
- **版本控制** - Git 最佳实践配置，丰富的 Git 别名
- **开发工具** - asdf 版本管理器，ctags，IRB/Pry 增强

**AI 集成：**

- Claude Code CLI 配置，支持项目感知的 CLAUDE.md 文件
- 自定义状态栏显示 Token 用量
- 预配置 LSP 插件（Python basedpyright、Go gopls）

**一键安装：**

```bash
bash -c "$(curl -fsSL https://raw.githubusercontent.com/lingti/lingti-code/master/install.sh)"
```

更多信息请访问：[官网](https://cli.lingti.com/code) | [GitHub](https://github.com/ruilisi/lingti-code)

---

**灵小缇** - 你的敏捷 AI 助手 🐕
