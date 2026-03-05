# 开发路线图

> lingti-bot 后续开发方向与功能规划。参考 [OpenClaw 对比](openclaw-comparison.md) 了解差异化定位。

## 功能状态总览

### 消息平台

| 平台 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| Slack | ✅ | ✅ | 已实现 |
| Discord | ✅ | ✅ | 已实现 |
| Telegram | ✅ | ✅ | 已实现 |
| 飞书/Lark | ✅ | ❌ | 已实现 |
| 微信公众号 | ✅ | ❌ | 已实现（云中继）|
| 企业微信 | ✅ | ❌ | 已实现 |
| 钉钉 | ✅ | ❌ | 已实现 |
| WhatsApp | ❌ | ✅ | 待开发 |
| iMessage | ❌ | ✅ | 待开发 |
| Signal | ❌ | ✅ | 待开发 |
| Microsoft Teams | ❌ | ✅ | 待开发 |
| Matrix | ❌ | ✅ | 待开发 |
| Google Chat | ❌ | ✅ | 待开发 |

### 语音功能

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| 语音输入 (STT) | ✅ | ✅ | 已实现 |
| 语音输出 (TTS) | ✅ | ✅ | 已实现 |
| ElevenLabs TTS | ✅ | ✅ | 已实现 |
| Voice Wake（唤醒词）| ✅ | ✅ | 已实现 |
| PTT 悬浮窗 | ❌ | ✅ | 待开发 |

### 可视化交互

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| 浏览器自动化 | ✅ | ✅ | 已实现 |
| Live Canvas | ❌ | ✅ | 待开发 |
| WebChat UI | ❌ | ✅ | 待开发 |

### 客户端应用

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| macOS 菜单栏应用 | ❌ | ✅ | 待开发 |
| iOS 应用 | ❌ | ✅ | 待开发 |
| Android 应用 | ❌ | ✅ | 待开发 |

### 社交平台自动化（MCP + 浏览器）

| 平台 | 发帖/回答 | 评论 | 点赞/收藏 | 搜索/浏览 | 状态 |
|------|:---:|:---:|:---:|:---:|------|
| **知乎** | ✅ | ✅ | ✅ | ✅ | 已实现 |
| **小红书** | 🔜 | 🔜 | 🔜 | 🔜 | 规划中 |
| **微博** | 🔜 | 🔜 | 🔜 | 🔜 | 规划中 |
| **抖音（网页版）** | 🔜 | 🔜 | 🔜 | 🔜 | 规划中 |
| **B站** | 🔜 | 🔜 | 🔜 | 🔜 | 规划中 |
| **今日头条** | 🔜 | 🔜 | 🔜 | 🔜 | 规划中 |

### 自动化

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| 定时任务 (Cron) | ❌ | ✅ | 待开发 |
| Webhooks | ❌ | ✅ | 待开发 |
| Gmail 集成 | ❌ | ✅ | 待开发 |
| 主动唤醒 (Heartbeat) | ❌ | ✅ | 待开发 |

### AI 功能

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| 多模型支持 | ✅ | ✅ | 已实现 |
| 对话记忆 | ✅ | ✅ | 已实现 |
| 模型 Failover | ❌ | ✅ | 待开发 |
| Extended Thinking | ✅ | ✅ | 已实现（Claude 原生 API）|
| Agent 间通信 | ❌ | ✅ | 待开发 |
| 持久化记忆 (RAG) | ❌ | ✅ | 待开发 |

### 效率工具

| 功能 | lingti-bot | OpenClaw | 状态 |
|------|:---:|:---:|------|
| Apple Calendar | ✅ | ✅ | 已实现 |
| Apple Reminders | ✅ | ✅ | 已实现 |
| Apple Notes | ✅ | ✅ | 已实现 |
| Apple Music | ✅ | ✅ | 已实现 |
| GitHub | ✅ | ✅ | 已实现 |
| Things 3 | ❌ | ✅ | 待开发 |
| Notion | ❌ | ✅ | 待开发 |
| Obsidian | ❌ | ✅ | 待开发 |

---

## 待开发功能

### 高优先级

- [ ] 定时任务 (Cron) — 支持定时执行任务
- [ ] Webhooks — 支持外部事件触发
- [ ] 模型 Failover — 主模型失败时自动切换备用模型
- [ ] WebChat UI — 浏览器端聊天界面

### 中优先级

- [ ] macOS 菜单栏应用 — SwiftUI 原生应用
- [x] Extended Thinking — 支持 Claude 深度思考模式（原生 Thinking API）
- [x] 按平台/频道模型切换 — AI overrides 配置
- [x] `doctor` 健康诊断命令
- [x] Docker 部署 — Dockerfile + docker-compose.yml
- [ ] DM 配对验证 — 未知发送者需验证码配对
- [ ] 持久化记忆 (RAG) — 跨会话知识库

### 低优先级

- [ ] WhatsApp / iMessage / Signal 集成
- [ ] Microsoft Teams / Matrix / Google Chat 集成
- [ ] Notion / Obsidian / Trello 集成
- [ ] iOS / Android 原生应用
- [ ] Live Canvas — Agent 驱动的可视化画布
- [x] Docker 部署 — Dockerfile + docker-compose.yml
- [ ] Agent 间通信 — 多 Agent 协作
- [ ] 技能注册中心 — 类似 ClawHub 的技能市场

---

## lingti-bot 独有功能

| 功能 | 说明 |
|------|------|
| 飞书/Lark 原生支持 | OpenClaw 不支持 |
| 钉钉原生支持 | OpenClaw 不支持 |
| 企业微信原生支持 | OpenClaw 不支持 |
| 微信公众号接入 | 通过云中继实现 |
| 云中继模式 | 免公网服务器，秒级接入 |
| 浏览器自动化（纯 Go）| 基于 go-rod，无需 Node.js |
| 社交平台自动化 | 知乎已支持，小红书/微博等规划中 |
| 中文语音默认 | whisper `-l zh` |
| 数据完全本地化 | 云端不存储任何数据 |

---

## 贡献

欢迎为以上功能提交 PR！

1. Fork 本仓库
2. 创建功能分支：`git checkout -b feature/xxx`
3. 提交代码：`git commit -m "feat: add xxx"`
4. 推送分支：`git push origin feature/xxx`
5. 创建 Pull Request

如有问题，请在 [Issues](https://github.com/ruilisi/lingti-bot/issues) 中讨论。
