[English](./README_EN.md) | 中文

---

# lsbot — Lean Secure Bot

> **我们不能相信任何聊天服务器，也不能允许把我们电脑上最重要的数据分享给任何聊天服务器。**

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Website](https://img.shields.io/badge/官网-bot.lingti.com-blue?style=flat)](https://bot.lingti.com)

> 📖 **文档：[bot.lingti.com/docs](https://bot.lingti.com/docs)**

---

## Bot Needs to be Secured

当你运行一个 AI Bot，你实际上把机器的钥匙交了出去：文件、终端、数据库、浏览器会话、凭证。Bot 读取你的代码、笔记、私人文档，并以你的名义执行命令。

问题不是 Bot 是否有用。**问题是：还有谁能看到它看到的东西？**

你每天操作电脑，层层加密加锁——密码管理器、磁盘加密、双重验证。但当你打开一个 AI Bot，这一切防线往往荡然无存。你的每一条消息、每一个工具调用结果，都在你不知情的情况下流经某台你不控制的服务器。

这不是理论风险。这是今天几乎所有 AI Bot 工具的默认配置。

---

## Why OpenClaw is Not Secured

OpenClaw（以及大多数同类工具）以便利性和功能广度为设计目标，安全性是事后考虑，而非核心设计原则。

| 问题 | 意味着什么 |
|---|---|
| 消息通过开发者的云服务路由 | 服务商可以读取每一条消息 |
| 没有端到端加密 | 任何中继节点都可以审查内容 |
| npm 依赖链 | 500+ 第三方包，任何一个都可能被攻击 |
| Node.js 运行时 | 动态执行环境，攻击面庞大 |
| 没有本地数据保证 | 对话历史、上下文、记忆——存储位置不明 |

OpenClaw 在功能和集成方面做得出色。但它的架构假设你信任中继基础设施。对于个人电脑、企业环境，或任何数据敏感的场合，这个假设不成立。

当你把终端和文件的访问权限交给 AI Bot，传输层必须被当作敌对环境对待。OpenClaw 没有做到这一点。

---

## Why lsbot

`lsbot` 是 **Lean Secure Bot** 的缩写。名字也呼应了 `ls`——Unix 最基础的命令，第一次拿到新机器时你敲的第一个命令。像 `ls` 一样，lsbot 是一个基础工具：精简、专注、永远可用。

**lsbot 建立在一个核心原则上：你的数据属于你。**

这个原则决定了每一个架构选择：

- **端到端加密，默认开启。** 所有 relay 流量使用 P-256 ECDH + AES-256-GCM 加密。中继服务器只看到密文，无法读取你的消息。
- **单一静态二进制。** 没有运行时，没有包管理器，没有依赖链。攻击面就是这一个二进制文件——可审计、确定性、可复现。
- **本地优先。** 对话历史、记忆、配置、凭证——全部存储在你的机器上。没有任何内容写入云端数据库。
- **密钥验证带外进行。** 浏览器不会自动从服务器获取 Bot 的公钥。你从 `lsbot e2e pubkey` 手动粘贴，并核对指纹。中继服务器无法替换密钥。

---

## Lean Secure Bot

```bash
lsbot relay --provider deepseek --api-key sk-xxx
```

一条命令启动安全 Bot：

1. 首次运行自动在 `~/.lingti-e2e.pem` 生成持久化 P-256 密钥对
2. 通过 WebSocket 连接中继服务器
3. 仅发布**公钥**——私钥永远不离开你的机器
4. 每一条响应在离开你的进程前加密
5. 每一条消息到达你的进程后解密

`bot.lingti.com` 的中继服务器只看到密文。它无法读取你的消息，无法记录你的对话，无法推断你在使用哪些工具或访问哪些文件。

```bash
lsbot e2e pubkey
# Key file:    ~/.lingti-e2e.pem
# Public key:  BK3x9f2...
# Fingerprint: sha256:29f8954f
```

打开 Bot 页面，点击 **Secure**，粘贴公钥，核对指纹与命令行输出一致。此后每次访问自动激活加密。

---

## 我们不能相信任何聊天服务器

**信任中继服务器是 AI Bot 安全的原罪。**

几乎所有主流 AI Bot 工具都依赖一个隐含假设：消息通过开发者的服务器路由是安全的。这个假设在以下任何场景中都会失效：

- 服务商被攻击或遭受数据泄露
- 政策变更导致日志保留或内容审查
- 内部员工滥用访问权限
- 法律程序要求披露用户数据
- 服务商出售或被收购

你的 AI Bot 可以访问你的终端、你的文件、你的代码库。它知道你在做什么项目，知道你的工作流程，知道你的私人文档。**这些信息通过一个你不控制的服务器转发——这不是可接受的风险，这是一个架构缺陷。**

lsbot 的设计前提是：中继服务器是不可信的。这不是对 bot.lingti.com 的不信任，而是一个工程原则：**不需要信任的地方，就不应该建立信任依赖。**

---

## 数据库保存在本地

lsbot 没有云端数据库。

| 数据 | 存储位置 | 云端是否存在 |
|---|---|:---:|
| 对话历史 | 浏览器 IndexedDB（本地） | ❌ |
| E2EE 私钥 | `~/.lingti-e2e.pem` | ❌ |
| 配置文件 | `~/.lingti.yaml` | ❌ |
| AI API 密钥 | `~/.lingti.yaml` | ❌ |
| Cron 任务数据库 | `~/.lingti.db` | ❌ |
| 技能文件 | `~/.lingti/skills/` | ❌ |

云中继服务器（`bot.lingti.com`）只做一件事：**路由加密后的消息**。它不保存任何内容，不分析任何内容，不记录任何内容。消息转发后立即丢弃。

即使 bot.lingti.com 明天关闭，你所有的数据、配置、历史记录都完好无损在你的机器上。这不是承诺，这是架构决定的结果。

---

## 快速开始

**两步启动，无需平台账号，无需公网服务器：**

**第一步：安装**

```bash
# macOS / Linux / WSL
curl -fsSL https://files.lingti.com/install-bot.sh | bash

# Windows (PowerShell)
irm https://files.lingti.com/install-bot.ps1 | iex
```

**第二步：运行**

```bash
lsbot relay --provider deepseek --api-key sk-xxx
```

启动后自动输出你的专属 Bot 页面：

```
Your bot page: https://bot.lingti.com/bots/xxx
E2E fingerprint: sha256:a3f7c91b2d4e8f06
```

打开链接，点击状态栏的 **Secure** → 粘贴公钥 → 核对指纹，即可在浏览器中与 Bot 进行端到端加密对话。

> 支持的 `--provider`：`deepseek`、`claude`、`kimi`、`minimax`、`gemini`、`openai` 等，详见 [AI-PROVIDERS.md](AI-PROVIDERS.md)。

#### 端到端加密（E2EE）

Bot Page 默认启用**端到端加密**，relay 服务器（bot.lingti.com）无法读取消息内容：

- bot 首次启动时，在 `~/.lingti-e2e.pem` 自动生成持久化 P-256 密钥，日志打印指纹
- 浏览器打开 Bot Page，点击状态栏 **Secure** 进入设置面板
- 从终端运行 `lsbot e2e pubkey`，将公钥粘贴到面板，核对指纹一致后点击 **Activate**
- 状态栏出现**蓝色锁图标**，此后页面刷新自动恢复加密（密钥存储在浏览器本地）
- 使用 `--no-e2ee` 可跳过 E2EE（不推荐）

```bash
# 查看本机公钥和指纹
lsbot e2e pubkey

# 手动生成密钥对
lsbot e2e keygen --save
```

详情：[端到端加密文档](https://bot.lingti.com/docs/e2e-encryption)

### 桌面客户端（MacOS/Windows）

桌面端不光拥有完整的安装、升级和使用 lsbot 的交互界面，还提供了 AI 网关、AI 加速等功能。

下载地址: [bot.lingti.com/download](https://bot.lingti.com/download)

---

### 进阶：接入消息平台（企业微信、飞书等）

如需接入企业微信、飞书、钉钉等平台，追加对应平台参数即可：

```bash
# 企业微信
lsbot relay --platform wecom \
  --wecom-corp-id ww... --wecom-agent-id 1000002 \
  --wecom-secret xxx --wecom-token xxx --wecom-aes-key xxx \
  --provider deepseek --api-key sk-xxx

# 飞书
lsbot relay --platform feishu \
  --feishu-app-id cli_xxx --feishu-app-secret xxx \
  --provider claude --api-key sk-ant-xxx
```

详见 [云中继文档](https://bot.lingti.com/docs/cloud-relay)。

---

## 为什么选择 lsbot？

### lsbot vs OpenClaw

|  | **lsbot** | **OpenClaw** |
|--|----------------|--------------|
| **端到端加密** | ✅ 默认开启，P-256 ECDH + AES-256-GCM | ❌ 无 |
| **服务器读取消息** | ❌ 不可能（只看密文） | ✅ 可以 |
| **数据存储位置** | 本地 | 云端 |
| **语言** | 纯 Go 实现 | Node.js |
| **运行依赖** | 无（单一二进制） | 需要 Node.js 运行时 |
| **依赖链风险** | 极低（Go 静态编译） | 高（500+ npm 包） |
| **安装大小** | ~15MB 单文件 | 100MB+ (含 node_modules) |
| **嵌入式设备** | ✅ ARM/MIPS | ❌ 需要 Node.js |
| **中国平台** | 原生支持飞书/企微/钉钉 | 需自行集成 |
| **云中继** | ✅ 免自建服务器 | ❌ 需自建 Web 服务 |

> 详细功能对比：[OpenClaw vs lsbot 技术特性对比](https://bot.lingti.com/docs/openclaw-feature-comparison)

### 核心能力

- 🔒 **端到端加密，默认开启** — P-256 ECDH + AES-256-GCM，中继服务器只路由密文
- 🚀 **零依赖部署** — 单个 ~15MB 二进制文件，无需 Node.js/Python 运行时
- ☁️ **[云中继](https://bot.lingti.com/docs/cloud-relay)加持** — 无需公网服务器、域名备案、HTTPS 证书，5 分钟接入企业微信/微信公众号
- 🤖 **[浏览器自动化](https://bot.lingti.com/docs/browser-automation)** — 内置完整 CDP 控制引擎，无需 Puppeteer/Playwright/Node.js
- 🌐 **[社交平台自动化](https://bot.lingti.com/docs/social-platform-automation)** — AI 代操作知乎、小红书等内容平台
- 🛠️ **75+ MCP 工具** — 覆盖文件、Shell、系统、网络、日历、Git、GitHub 等全场景
- 🌏 **中国平台原生支持** — 钉钉、飞书、企业微信、微信公众号开箱即用
- 💬 **[内置 Web 聊天界面](#web-chat-ui)** — 无需任何客户端，浏览器直开，支持多会话并行
- 🧠 **多 AI 后端** — 集成 Claude、DeepSeek、Kimi、MiniMax、Gemini 等 [16 种 AI 服务](AI-PROVIDERS.md)
- 🔬 **Claude 深度思考** — 原生支持 Claude Extended Thinking API
- 🩺 **健康诊断** — `lsbot doctor` 一键检查配置、连接、依赖

---

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

**Agent** 是 lsbot 中的 AI 实例单元。每个 Agent 拥有独立的：

- **AI 配置** — 可指定不同的 provider、model、API Key，或继承全局配置
- **系统提示词（Instructions）** — 定义 Agent 的角色、风格、行为边界
- **工作目录（Workspace）** — Agent 读写文件的默认根目录，天然沙箱隔离

典型用途：`work-assistant`（工作助手）、`coder`（专注代码）、`writer`（内容写作）……每个 Agent 可以有完全不同的人格和能力边界。

### 什么是 Channel？

**Channel** 是消息的来源/目标通道，对应一个具体的聊天平台接入点（如某个企业微信应用、某个 Slack Bot、某个 Telegram Bot）。你可以为不同 Channel 绑定不同的 Agent，实现"同一个 lsbot 进程，多个平台、多套 AI 人格"的效果。

### 管理 Agents

```bash
lsbot agents add          # 交互式添加 Agent
lsbot agents list         # 列出所有 Agent
lsbot agents info <name>  # 查看某个 Agent 详情
lsbot agents remove <name>
```

### 管理 Channels

```bash
lsbot channels add           # 交互式添加 Channel 凭证
lsbot channels list          # 列出所有已配置的 Channel
lsbot channels remove <name>
```

### 添加 Channel 后如何启动？

**`gateway` 模式**（适用于大多数平台：Telegram、Discord、Slack、钉钉、飞书等）

```bash
lsbot gateway
```

**`relay` 模式**（仅适用于需要云中继的平台：企业微信、微信公众号、飞书、Slack）

```bash
lsbot relay --platform wecom    # 企业微信
lsbot relay --platform feishu   # 飞书
lsbot relay --platform wechat   # 微信公众号
```

> - `gateway` — 本地直连，支持所有平台，一条命令启动全部
> - `relay` — 云中继，仅支持 wecom/feishu/wechat/slack，无需公网服务器

---

## 功能概览

### MCP Server — 标准协议，无缝集成

lsbot 实现了完整的 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 协议，让任何支持 MCP 的 AI 客户端都能访问本地系统资源。

| 客户端 | 状态 |
|--------|------|
| Claude Desktop | ✅ |
| Cursor | ✅ |
| Windsurf | ✅ |
| 其他 MCP 客户端 | ✅ |

### 多平台消息网关 — 企业 IM 秒接入

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

> 完整列表（含配置参数、环境变量）：[聊天平台列表](https://bot.lingti.com/docs/chat-platforms)

### MCP 工具集 — 75+ 本地系统工具

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
| **[浏览器自动化](https://bot.lingti.com/docs/browser-automation)** | 14 | CDP 控制引擎，可接管已有 Chrome，快照定位、点击/输入/JS、多标签页、截图 |
| **定时任务** | 5 | 创建、列表、删除、暂停、恢复计划任务 |

### 健康诊断

```bash
lsbot doctor
```

```
lsbot doctor
=================
OS: darwin/arm64, Go: go1.24.0

Checks:
  ✓ Config file (~/.lingti.yaml) — loaded
  ✓ AI API key — set (sk-ant-a..., provider: claude)
  ✓ AI provider connectivity — reachable (HTTP 200)
  ✓ Platform credentials — wecom, telegram
  ✓ Binary: gh — found
  ✗ Binary: chrome — not found in PATH

8 passed, 1 failed
```

### 按平台/频道模型切换

```yaml
ai:
  provider: deepseek
  api_key: sk-xxx
  model: deepseek-chat
  overrides:
    - platform: telegram
      provider: claude
      api_key: sk-ant-xxx
      model: claude-sonnet-4-20250514
    - platform: slack
      channel_id: C12345
      provider: claude
      api_key: sk-ant-xxx
```

### Claude 深度思考 — 原生 Extended Thinking API

| 命令 | 模式 | 思考 Token 预算 |
|------|------|-----------------|
| `/think off` | 关闭 | 0 |
| `/think low` | 简单 | 1,024 |
| `/think medium` | 中等（默认） | 4,096 |
| `/think high` | 深度 | 16,384 |

### Skills — 模块化能力扩展

```bash
lsbot skills          # 列出所有 Skills
lsbot skills check    # 查看就绪状态
lsbot skills info github
```

内置 8 个 Skills：Discord、GitHub、Slack、Peekaboo（macOS UI 自动化）、Tmux、天气、1Password、Obsidian。

详细文档：[Skills 指南](https://bot.lingti.com/docs/skills)

### 功能速览表

| 模块 | 说明 | 特点 |
|------|------|------|
| **端到端加密** | P-256 ECDH + AES-256-GCM | 默认开启，中继服务器只路由密文 |
| **MCP Server** | 标准 MCP 协议服务器 | 兼容 Claude Desktop、Cursor、Windsurf 等 |
| **多平台消息网关** | [19 种聊天平台](https://bot.lingti.com/docs/chat-platforms) | 微信公众号、企业微信、Slack、飞书一键接入 |
| **[Web 聊天界面](#web-chat-ui)** | 内置浏览器 UI | 多会话并行，独立记忆隔离，Markdown 渲染 |
| **MCP 工具集** | 75+ 本地系统工具 | 文件、Shell、系统、网络、日历、Git、GitHub |
| **Skills** | 模块化能力扩展 | 8 个内置 Skill，支持自定义和项目级扩展 |
| **智能对话** | 多轮对话与记忆 | 上下文记忆、[16 种 AI 后端](AI-PROVIDERS.md) |
| **深度思考** | Claude Extended Thinking | 原生 Thinking API，4 级思考深度 |
| **健康诊断** | `doctor` 命令 | 一键检查配置、连接、依赖 |
| **Docker 部署** | 容器化 | 多阶段构建，docker-compose 支持 |

---

## 云中继：零门槛接入企业消息平台

> **告别公网服务器、告别复杂配置，让 AI Bot 接入像配置 Wi-Fi 一样简单**

**工作原理：**

```
企业微信(用户消息) --> bot.lingti.com(云中继，只看密文) --WebSocket--> lsbot(本地AI处理)
```

| | 传统方案 | 云中继方案 |
|---|---|---|
| 公网服务器 | ✅ 需要 | ❌ 不需要 |
| 域名/备案 | ✅ 需要 | ❌ 不需要 |
| HTTPS证书 | ✅ 需要 | ❌ 不需要 |
| 接入时间 | 数天 | **5分钟** |
| AI处理位置 | 服务器 | **本地** |
| 数据安全 | 云端存储 | **本地处理 + E2EE** |

> 📖 [云中继技术方案详解](https://bot.lingti.com/docs/cloud-relay)

### 微信公众号

微信搜索公众号「**灵缇小秘**」，关注后发送任意消息获取接入教程。
详细教程：[微信公众号接入指南](https://bot.lingti.com/docs/wechat-integration)

### 企业微信

```bash
lsbot relay --platform wecom \
  --wecom-corp-id YOUR_CORP_ID \
  --wecom-agent-id YOUR_AGENT_ID \
  --wecom-secret YOUR_SECRET \
  --wecom-token YOUR_TOKEN \
  --wecom-aes-key YOUR_AES_KEY \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

详细教程：[企业微信集成指南](https://bot.lingti.com/docs/wecom-integration)

### 钉钉

```bash
lsbot gateway \
  --dingtalk-client-id YOUR_APP_KEY \
  --dingtalk-client-secret YOUR_APP_SECRET \
  --provider deepseek \
  --api-key YOUR_API_KEY
```

---

## Sponsors

- **[灵缇游戏加速](https://game.lingti.com)** - PC/Mac/iOS/Android 全平台游戏加速、热点加速、AI 及学术资源定向加速
- **[灵缇路由](https://router.lingti.com)** - 您的路由管家、网游电竞专家

---

## lingti-cli 生态

**lsbot** 是 **lingti-cli** 五位一体平台的核心开源组件。

| 模块 | 定位 | 说明 |
|------|------|------|
| **CLI** | 操控总台 | 统一入口，如同操作系统的引导程序 |
| **Net** | 全球网络 | 跨洲 200Mbps 加速，畅享全球 AI 服务 |
| **Token** | 数字员工 | Token 即代码，代码即生产力 |
| **Bot** | 助理管理 | 数字员工接入与管理，简单到极致 ← *本项目* |
| **Code** | 开发环境 | Terminal 回归舞台中央，极致输入效率 |

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
                              lsbot
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

## MCP Server 快速配置

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`)：

```json
{
  "mcpServers": {
    "lsbot": {
      "command": "/path/to/lsbot",
      "args": ["serve"]
    }
  }
}
```

**Cursor** (`.cursor/mcp.json`)：

```json
{
  "mcpServers": {
    "lsbot": {
      "command": "/path/to/lsbot",
      "args": ["serve"]
    }
  }
}
```

---

## 其他安装方式

**从源码编译**

```bash
git clone https://github.com/ruilisi/lsbot.git
cd lsbot
make build  # 或: make darwin-arm64 / make linux-amd64
```

**手动下载**

- **命令行版本**：前往 [GitHub Releases](https://github.com/ruilisi/lsbot/releases) 下载
- **桌面客户端**（Windows x64 / macOS Apple Silicon / macOS Intel）：前往 [bot.lingti.com/download](https://bot.lingti.com/download) 下载

**支持的目标平台：**

| 平台 | 架构 | 编译命令 |
|------|------|----------|
| macOS | ARM64 (Apple Silicon) | `make darwin-arm64` |
| macOS | AMD64 (Intel) | `make darwin-amd64` |
| Linux | AMD64 | `make linux-amd64` |
| Linux | ARM64 | `make linux-arm64` |
| Linux | ARMv7 (树莓派等) | `make linux-arm` |
| Windows | AMD64 | `make windows-amd64` |

---

## 配置优先级

所有配置项按以下优先级解析：**命令行参数 > 环境变量 > 配置文件 (`~/.lingti.yaml`)**

完整的环境变量、命令行参数、配置文件对照表：[配置优先级](CONFIGURATION.md)

---

## 详细文档

- [配置优先级](CONFIGURATION.md)
- [AI 服务列表](AI-PROVIDERS.md) — 16 种 AI 服务详情、API Key 获取、别名
- [端到端加密](https://bot.lingti.com/docs/e2e-encryption) — E2EE 架构、密钥管理、指纹验证
- [聊天平台列表](https://bot.lingti.com/docs/chat-platforms) — 19 种聊天平台详情
- [命令行参考](https://bot.lingti.com/docs/cli-reference) — 完整的命令行使用文档
- [Skills 指南](https://bot.lingti.com/docs/skills)
- [浏览器自动化指南](https://bot.lingti.com/docs/browser-automation) — CDP 引擎完整参考
- [社交平台自动化指南](https://bot.lingti.com/docs/social-platform-automation)
- [定时任务指南](https://bot.lingti.com/docs/cron-jobs)
- [云中继技术方案详解](https://bot.lingti.com/docs/cloud-relay)
- [OpenClaw 技术特性对比](https://bot.lingti.com/docs/openclaw-feature-comparison)

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

**lsbot** — Lean Secure Bot · [bot.lingti.com](https://bot.lingti.com) · [GitHub](https://github.com/ruilisi/lsbot)
