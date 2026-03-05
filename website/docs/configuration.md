# 配置优先级

lingti-bot 采用三层配置解析机制，优先级从高到低：

```
命令行参数  >  环境变量  >  配置文件 (~/.lingti.yaml)
```

每个配置项按此顺序查找，找到即停止。这意味着：

- **命令行参数**始终优先，适合临时覆盖或运行多个实例
- **环境变量**适合 CI/CD 或容器化部署
- **配置文件**适合日常使用，通过 `lingti-bot onboard` 生成

## 示例

以 AI Provider 为例，解析顺序为：

| 优先级 | 来源 | 示例 |
|--------|------|------|
| 1 | `--provider deepseek` | 命令行参数 |
| 2 | `AI_PROVIDER=deepseek` | 环境变量 |
| 3 | `ai.provider: deepseek` | ~/.lingti.yaml |

```bash
# 配置文件中设置了 provider: qwen
# 环境变量设置了 AI_PROVIDER=deepseek
# 命令行指定了 --provider claude
# 最终使用: claude（命令行参数最高优先）
```

## 配置文件

默认路径：`~/.lingti.yaml`

通过交互式向导生成：

```bash
lingti-bot onboard
```

### 完整结构

```yaml
mode: relay  # "relay" 或 "router"

# 推荐：命名 Provider 配置
# 每个 provider 独立定义 api_key / base_url / model，不会互相污染
providers:
  my-deepseek:
    provider: deepseek
    api_key: sk-xxx
    model: deepseek-chat
  my-kimi:
    provider: kimi
    api_key: ak-xxx
    model: kimi-k2.5

# 旧格式（仍然支持，向后兼容）
ai:
  provider: deepseek
  api_key: sk-xxx
  base_url: ""       # 自定义 API 地址（可选）
  model: ""          # 自定义模型名（可选，留空使用 provider 默认值）
  max_rounds: 100    # 每条消息最多工具调用轮次（默认 100）
  call_timeout_secs: 90  # 每次 AI API 调用的基础超时秒数（默认 90）；使用本地 Ollama 等慢速模型时可适当增大

  # 按平台/频道覆盖 AI 设置（旧格式，可选）
  # 匹配优先级：platform + channel_id > platform > 默认
  overrides:
    - platform: telegram        # 平台名（必填）
      provider: claude          # 覆盖 provider
      api_key: sk-ant-xxx       # 覆盖 api_key
      model: claude-sonnet-4-20250514
    - platform: slack
      channel_id: C12345        # 可选：指定频道
      provider: openai
      api_key: sk-xxx
      model: gpt-4o

  # 外部 MCP 服务器（可选）：bot 启动时自动连接并将其工具暴露给 AI
  # 工具名称格式：mcp_<name>_<tool_name>
  mcp_servers:
    - name: chrome          # 工具名前缀，如 mcp_chrome_take_snapshot
      command: npx
      args: ["chrome-devtools-mcp", "--browserUrl=http://127.0.0.1:9222"]
    # - name: my_server      # SSE 方式连接
    #   url: http://localhost:3000/sse

relay:
  platform: wecom    # "feishu", "slack", "wechat", "wecom"
  provider: my-kimi  # 引用 providers 中的命名条目
  user_id: ""        # 从 /whoami 获取（WeCom 不需要）

platforms:
  wecom:
    corp_id: ""
    agent_id: ""
    secret: ""
    token: ""
    aes_key: ""
  wechat:
    app_id: ""
    app_secret: ""
  feishu:
    app_id: ""
    app_secret: ""
  slack:
    bot_token: ""
    app_token: ""
  dingtalk:
    client_id: ""
    client_secret: ""
  telegram:
    token: ""
  discord:
    token: ""

browser:
  screen_size: fullscreen  # "fullscreen" 或 "宽x高"（如 "1024x768"），默认 fullscreen
  cdp_url: "127.0.0.1:9222"  # 可选：连接已运行的 Chrome（需以 --remote-debugging-port 启动）

security:
  allowed_paths:             # 限制文件操作的目录白名单（空=不限制）
    - ~/Documents
    - ~/Downloads
  blocked_commands:          # 禁止执行的命令前缀
    - "rm -rf /"
    - "mkfs"
    - "dd if="
  require_confirmation: []   # 需要用户确认的命令（预留）
```

## 安全配置

通过 `security` 配置项限制 bot 的文件系统访问和命令执行范围。

### allowed_paths — 目录白名单

限制 `file_read`、`file_write`、`file_list`、`file_trash` 和 `shell_execute` 只能访问指定目录：

```yaml
security:
  allowed_paths:
    - ~/Documents/work
    - ~/Downloads
```

- 空列表 `[]`（默认）= 不限制，可访问所有路径
- 设置后，所有文件操作必须在白名单目录内，否则返回权限错误
- 路径支持 `~` 展开为用户 home 目录

### blocked_commands — 命令黑名单

阻止 `shell_execute` 执行包含指定前缀的命令：

```yaml
security:
  blocked_commands:
    - "rm -rf /"
    - "mkfs"
    - "dd if="
```

## 环境变量

### AI 配置

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `AI_PROVIDER` | `--provider` | AI 服务商 |
| `AI_API_KEY` | `--api-key` | API 密钥 |
| `AI_BASE_URL` | `--base-url` | 自定义 API 地址 |
| `AI_MODEL` | `--model` | 模型名称 |
| `AI_MAX_ROUNDS` | `--max-rounds` | 每条消息最多工具调用轮次（默认 100） |
| `AI_CALL_TIMEOUT` | `--call-timeout` | 每次 AI API 调用的基础超时秒数（默认 90） |
| - | `--instructions` | 自定义指令文件路径（追加到系统提示词） |
| `ANTHROPIC_API_KEY` | `--api-key` | API 密钥（fallback） |
| `ANTHROPIC_BASE_URL` | `--base-url` | API 地址（fallback） |
| `ANTHROPIC_MODEL` | `--model` | 模型名称（fallback） |

### Relay 配置

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `RELAY_USER_ID` | `--user-id` | 用户 ID |
| `RELAY_PLATFORM` | `--platform` | 平台类型 |
| `RELAY_SERVER_URL` | `--server` | WebSocket 服务器地址 |
| `RELAY_WEBHOOK_URL` | `--webhook` | Webhook 地址 |

### 平台凭证

| 环境变量 | 对应参数 | 说明 |
|----------|----------|------|
| `WECOM_CORP_ID` | `--wecom-corp-id` | 企业微信 Corp ID |
| `WECOM_AGENT_ID` | `--wecom-agent-id` | 企业微信 Agent ID |
| `WECOM_SECRET` | `--wecom-secret` | 企业微信 Secret |
| `WECOM_TOKEN` | `--wecom-token` | 企业微信回调 Token |
| `WECOM_AES_KEY` | `--wecom-aes-key` | 企业微信 AES Key |
| `WECHAT_APP_ID` | `--wechat-app-id` | 微信公众号 App ID |
| `WECHAT_APP_SECRET` | `--wechat-app-secret` | 微信公众号 App Secret |
| `SLACK_BOT_TOKEN` | - | Slack Bot Token |
| `SLACK_APP_TOKEN` | - | Slack App Token |
| `FEISHU_APP_ID` | - | 飞书 App ID |
| `FEISHU_APP_SECRET` | - | 飞书 App Secret |
| `DINGTALK_CLIENT_ID` | - | 钉钉 Client ID |
| `DINGTALK_CLIENT_SECRET` | - | 钉钉 Client Secret |

## 典型用法

### 日常使用：配置文件

```bash
lingti-bot onboard        # 首次配置
lingti-bot relay           # 之后无需任何参数
```

### 临时覆盖：命令行参数

```bash
# 配置文件用 deepseek，临时切换到 qwen 测试
lingti-bot relay --provider qwen --model qwen-plus
```

### 容器部署：Docker Compose

```bash
# 使用 docker-compose（推荐）
AI_API_KEY=sk-xxx TELEGRAM_BOT_TOKEN=xxx docker compose up -d

# 或直接 docker run
docker run -e AI_PROVIDER=deepseek -e AI_API_KEY=sk-xxx lingti-bot gateway
```

挂载配置文件以使用 overrides 和其他高级功能：

```bash
docker run -v ~/.lingti.yaml:/root/.lingti.yaml:ro \
  -e AI_API_KEY=sk-xxx lingti-bot gateway
```

### 多实例运行：命令行参数覆盖

```bash
# 实例 1: 企业微信
lingti-bot relay --platform wecom --provider deepseek --api-key sk-aaa

# 实例 2: 飞书（不同 provider）
lingti-bot relay --platform feishu --user-id xxx --provider claude --api-key sk-bbb
```

### 本地模型：Ollama

Ollama 在本地运行大模型，无需 API 密钥：

```bash
# 使用默认模型 (llama3.2)
lingti-bot relay --provider ollama

# 指定模型
lingti-bot relay --provider ollama --model mistral

# 连接远程 Ollama 实例
lingti-bot relay --provider ollama --base-url http://remote-host:11434/v1
```
