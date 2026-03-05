# Supported AI Providers / 支持的 AI 服务

lingti-bot 支持 **16 种 AI 服务**，涵盖国内外主流大模型平台及本地模型，按需切换。所有 provider 均通过 `--provider` 参数指定，也可在 `lingti-bot onboard` 交互式向导中选择。

lingti-bot supports **16 AI providers** covering mainstream LLM platforms globally plus local models. Select via `--provider` flag or the `lingti-bot onboard` interactive wizard.

## Provider List / 服务列表

| # | Provider | 名称 | Default Model / 默认模型 | Default Base URL | API Key URL |
|---|----------|------|--------------------------|------------------|-------------|
| 1 | `deepseek` | DeepSeek (recommended / 推荐) | `deepseek-chat` | `https://api.deepseek.com/v1` | [platform.deepseek.com](https://platform.deepseek.com/api_keys) |
| 2 | `qwen` | Qwen / 通义千问 | `qwen-plus` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | [bailian.console.aliyun.com](https://bailian.console.aliyun.com/) |
| 3 | `claude` | Claude (Anthropic) | `claude-sonnet-4-20250514` | Anthropic native API | [console.anthropic.com](https://console.anthropic.com/) |
| 4 | `kimi` | Kimi / Moonshot / 月之暗面 | `kimi-k2.5` | `https://api.moonshot.cn/v1` | [platform.moonshot.cn](https://platform.moonshot.cn/) |

> **Kimi thinking models / Kimi 思考模型：** 默认模型已升级为 `kimi-k2.5`（思考模型）。lingti-bot 会自动在对话历史中保留 `reasoning_content`，确保 tool call 多轮对话正常工作。如需使用非思考模型，可指定 `--model moonshot-v1-8k`。
| 5 | `minimax` | MiniMax / 海螺 AI | `MiniMax-Text-01` | `https://api.minimax.chat/v1` | [platform.minimaxi.com](https://platform.minimaxi.com/) |
| 6 | `doubao` | Doubao / 豆包 (ByteDance) | `doubao-pro-32k` | `https://ark.cn-beijing.volces.com/api/v3` | [console.volcengine.com/ark](https://console.volcengine.com/ark) |
| 7 | `zhipu` | Zhipu / 智谱 GLM | `glm-4-flash` | `https://open.bigmodel.cn/api/paas/v4` | [open.bigmodel.cn](https://open.bigmodel.cn/) |
| 8 | `openai` | OpenAI (GPT) | `gpt-4o` | `https://api.openai.com/v1` | [platform.openai.com](https://platform.openai.com/api-keys) |
| 9 | `gemini` | Gemini (Google) | `gemini-2.0-flash` | `https://generativelanguage.googleapis.com/v1beta/openai` | [aistudio.google.com](https://aistudio.google.com/apikey) |
| 10 | `yi` | Yi / 零一万物 (Lingyiwanwu) | `yi-large` | `https://api.lingyiwanwu.com/v1` | [platform.lingyiwanwu.com](https://platform.lingyiwanwu.com/) |
| 11 | `stepfun` | StepFun / 阶跃星辰 | `step-2-16k` | `https://api.stepfun.com/v1` | [platform.stepfun.com](https://platform.stepfun.com/) |
| 12 | `baichuan` | Baichuan / 百川智能 | `Baichuan4` | `https://api.baichuan-ai.com/v1` | [platform.baichuan-ai.com](https://platform.baichuan-ai.com/) |
| 13 | `spark` | Spark / 讯飞星火 (iFlytek) | `generalv3.5` | `https://spark-api-open.xf-yun.com/v1` | [console.xfyun.cn](https://console.xfyun.cn/) |
| 14 | `siliconflow` | SiliconFlow / 硅基流动 (aggregator) | `Qwen/Qwen2.5-72B-Instruct` | `https://api.siliconflow.cn/v1` | [cloud.siliconflow.cn](https://cloud.siliconflow.cn/) |
| 15 | `grok` | Grok (xAI) | `grok-2-latest` | `https://api.x.ai/v1` | [console.x.ai](https://console.x.ai/) |
| 16 | `ollama` | Ollama (local / 本地) | `llama3.2` | `http://localhost:11434/v1` | No API key needed / 无需密钥 |

## Aliases / 别名

以下别名可以替代 `--provider` 值：

| Alias / 别名 | Maps to / 对应 |
|---------------|----------------|
| `anthropic` | `claude` |
| `moonshot` | `kimi` |
| `qianwen`, `tongyi` | `qwen` |
| `gpt`, `chatgpt` | `openai` |
| `glm`, `chatglm` | `zhipu` |
| `google` | `gemini` |
| `lingyiwanwu`, `wanwu` | `yi` |
| `bytedance`, `volcengine` | `doubao` |
| `iflytek`, `xunfei` | `spark` |
| `xai` | `grok` |
| `tencent`, `hungyuan` | `hunyuan` |

## Usage / 用法

```bash
# Interactive wizard / 交互式向导
lingti-bot onboard

# Command line / 命令行指定
lingti-bot relay --provider deepseek --api-key sk-xxx
lingti-bot gateway --provider openai --api-key sk-xxx --model gpt-4o

# Custom base URL / 自定义 API 地址
lingti-bot relay --provider siliconflow --api-key sk-xxx --base-url https://api.siliconflow.cn/v1

# Override default model / 覆盖默认模型
lingti-bot relay --provider qwen --api-key sk-xxx --model qwen-max
```

---

## Named Providers / 命名 Provider 配置（推荐）

The recommended way to configure AI providers. Define each provider as a named entry, then reference by name.

推荐的 AI 配置方式。将每个 provider 定义为命名条目，然后按名称引用。

### Config Format / 配置格式

```yaml
providers:
  my-deepseek:
    provider: deepseek
    api_key: sk-xxx
    model: deepseek-chat
  my-kimi:
    provider: kimi
    api_key: ak-xxx
    model: kimi-k2.5
  my-minimax:
    provider: minimax
    api_key: sk-xxx
    base_url: https://api.minimax.chat/v1
    model: MiniMax-M2.5

relay:
  platform: wecom
  provider: my-kimi        # references named provider

ai:
  max_rounds: 100          # shared settings stay here
  mcp_servers: [...]
```

### Resolution / 解析规则

When `--provider` is specified (CLI or config), lingti-bot resolves the name:

1. **Exact key match** — `--provider my-kimi` matches `providers.my-kimi` directly
2. **Provider type match** — `--provider kimi` scans entries for `.provider == "kimi"`
3. **Backward compat** — if no `providers:` map exists, falls back to `ai:` block

CLI flags `--api-key`, `--base-url`, `--model` still override individual fields on the resolved entry.

```bash
# Use named provider directly
lingti-bot relay --provider my-kimi

# Use provider type (finds first matching entry)
lingti-bot relay --provider kimi

# Override model on a named provider
lingti-bot relay --provider my-kimi --model moonshot-v1-8k
```

### Benefits / 优势

- Each provider carries its own api_key, base_url, model — no cross-contamination
- `--provider kimi` won't inherit deepseek's base_url from the `ai:` block
- Easy to switch between providers: just change `relay.provider`
- Old `ai:` format still works unchanged (backward compatible)

---

## Per-Platform AI Overrides / 按平台覆盖 AI 设置（旧格式）

lingti-bot 的独有功能：可以为不同消息平台或频道指定不同的 AI 服务。一个实例同时服务多个平台，每个平台使用最适合的模型。

A unique lingti-bot feature: assign different AI providers to different messaging platforms or channels. One instance serves multiple platforms, each using its best-fit model.

### How It Works / 工作原理

在 `~/.lingti.yaml` 中配置 `ai.overrides` 数组。当收到消息时，lingti-bot 按以下优先级匹配：

1. **platform + channel_id** — 最精确，匹配特定平台的特定频道
2. **platform only** — 匹配该平台的所有消息
3. **default** — 无匹配时使用默认 `ai` 配置

Override matching priority when a message arrives:

1. **platform + channel_id** — most specific, matches a specific channel on a platform
2. **platform only** — matches all messages from that platform
3. **default** — falls back to the base `ai` config

### Override Fields / 覆盖字段

每条 override 可覆盖以下字段（均为可选，未设置的字段从默认配置继承）：

| Field / 字段 | Description / 说明 |
|---|---|
| `platform` | **必填**。匹配的平台名：`wecom`, `feishu`, `slack`, `telegram`, `discord`, `dingtalk`, `wechat` 等 |
| `channel_id` | 可选。匹配的频道/群组 ID，不填则匹配该平台所有消息 |
| `provider` | 覆盖 AI 服务商（如 `kimi`, `claude`, `openai`） |
| `api_key` | 覆盖 API 密钥 |
| `base_url` | 覆盖 API 地址 |
| `model` | 覆盖模型名 |

### Key Behavior: Provider Switch / 重要行为：切换 Provider

**当 override 切换了 provider 时（如从 deepseek 切到 kimi），`base_url` 和 `model` 会自动清空，使用新 provider 的默认值。** 这避免了错误地将 deepseek 的 API 地址传给 kimi。

**When an override changes the provider (e.g. from deepseek to kimi), `base_url` and `model` are automatically cleared and the new provider's defaults are used.** This prevents accidentally sending deepseek's API URL to kimi.

如果你需要同时覆盖 provider 和 model，必须在同一条 override 中同时指定：

```yaml
overrides:
  - platform: wecom
    provider: kimi
    api_key: sk-xxx
    model: moonshot-v1-128k    # 必须显式指定，否则使用 kimi 默认的 moonshot-v1-8k
```

---

### Example 1: Basic Override / 基础覆盖

默认使用 DeepSeek，企业微信使用 Kimi：

Default is DeepSeek, WeCom uses Kimi:

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx
  model: deepseek-chat

  overrides:
    - platform: wecom
      provider: kimi
      api_key: sk-kimi-xxx
```

**Result / 结果：**

| Message Source / 消息来源 | Provider Used / 使用的 Provider | Model / 模型 | Why / 原因 |
|---|---|---|---|
| WeCom user sends "hi" | **kimi** | `moonshot-v1-8k` (kimi default) | override matches `platform: wecom`, provider changed → base_url & model auto-cleared → kimi defaults |
| Telegram user sends "hi" | **deepseek** | `deepseek-chat` | no override matches telegram → uses default |
| Slack user sends "hi" | **deepseek** | `deepseek-chat` | no override matches slack → uses default |

---

### Example 2: Override with Custom Model / 覆盖并指定模型

默认使用 DeepSeek，飞书使用 Claude Sonnet，Telegram 使用 GPT-4o-mini（节省成本）：

Default is DeepSeek, Feishu uses Claude Sonnet, Telegram uses GPT-4o-mini (cost saving):

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx

  overrides:
    - platform: feishu
      provider: claude
      api_key: sk-ant-xxx
      model: claude-sonnet-4-20250514
    - platform: telegram
      provider: openai
      api_key: sk-openai-xxx
      model: gpt-4o-mini
```

**Result / 结果：**

| Message Source | Provider | Model | Why |
|---|---|---|---|
| Feishu message | **claude** | `claude-sonnet-4-20250514` | override sets provider + model explicitly |
| Telegram message | **openai** | `gpt-4o-mini` | override sets provider + model explicitly |
| WeCom message | **deepseek** | `deepseek-chat` | no override → default |

---

### Example 3: Channel-Specific Override / 频道级覆盖

同一个 Slack 里，VIP 频道用 Claude，其他频道用 DeepSeek：

Same Slack workspace, VIP channel uses Claude, others use DeepSeek:

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx

  overrides:
    - platform: slack
      channel_id: C06ABCDEF12          # VIP 频道
      provider: claude
      api_key: sk-ant-xxx
      model: claude-sonnet-4-20250514
```

**Result / 结果：**

| Message Source | Provider | Model | Why |
|---|---|---|---|
| Slack #vip-channel (C06ABCDEF12) | **claude** | `claude-sonnet-4-20250514` | channel_id matches → most specific override wins |
| Slack #general (C09XYZABC34) | **deepseek** | `deepseek-chat` | channel_id doesn't match → no override → default |
| Telegram message | **deepseek** | `deepseek-chat` | no override → default |

---

### Example 4: Same Provider, Different Model / 同 Provider 不同模型

同一个 provider，不同平台使用不同模型。这时 provider 没有改变，所以 base_url 和 model 不会被清空：

Same provider for different platforms but with different models. Since provider doesn't change, base_url and model are inherited and only the explicitly set fields are overridden:

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx
  model: deepseek-chat

  overrides:
    - platform: wecom
      model: deepseek-reasoner     # 企业微信用推理模型
```

**Result / 结果：**

| Message Source | Provider | Model | Why |
|---|---|---|---|
| WeCom message | **deepseek** | `deepseek-reasoner` | same provider → inherits api_key & base_url, only model overridden |
| Telegram message | **deepseek** | `deepseek-chat` | no override → default |

---

### Example 5: Override Only API Key / 仅覆盖密钥

多个平台共用同一 provider，但使用不同的 API 密钥（用于账单拆分或配额隔离）：

Multiple platforms share the same provider but use different API keys (for billing or quota isolation):

```yaml
ai:
  provider: openai
  api_key: sk-default-xxx
  model: gpt-4o

  overrides:
    - platform: discord
      api_key: sk-discord-team-xxx     # Discord 团队单独的 API Key
    - platform: telegram
      api_key: sk-telegram-team-xxx    # Telegram 团队单独的 API Key
```

**Result / 结果：**

| Message Source | Provider | Model | API Key | Why |
|---|---|---|---|---|
| Discord message | **openai** | `gpt-4o` | `sk-discord-team-xxx` | same provider → only api_key overridden |
| Telegram message | **openai** | `gpt-4o` | `sk-telegram-team-xxx` | same provider → only api_key overridden |
| Slack message | **openai** | `gpt-4o` | `sk-default-xxx` | no override → default |

---

### Example 6: Custom Base URL (API Proxy) / 自定义 API 地址（代理）

使用第三方 API 代理或聚合平台的兼容端点：

Using a third-party API proxy or aggregator's compatible endpoint:

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx

  overrides:
    - platform: wecom
      provider: openai
      api_key: sk-proxy-xxx
      base_url: https://my-proxy.example.com/v1    # 自定义代理地址
      model: gpt-4o
```

**Result / 结果：**

| Message Source | Provider | Base URL | Model |
|---|---|---|---|
| WeCom message | **openai** | `https://my-proxy.example.com/v1` | `gpt-4o` |
| Other platforms | **deepseek** | `https://api.deepseek.com/v1` (default) | `deepseek-chat` |

> **Note / 注意：** 当 override 同时改变了 provider 和 base_url，必须显式指定 base_url，因为切换 provider 会自动清空继承的 base_url。

---

### Example 7: Complex Multi-Platform Setup / 复杂多平台配置

一个 lingti-bot 实例服务 4 个平台，各用不同模型：

One lingti-bot instance serving 4 platforms with different models:

```yaml
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx
  model: deepseek-chat

  overrides:
    # 企业微信用 Kimi（中文对话优化）
    - platform: wecom
      provider: kimi
      api_key: sk-kimi-xxx

    # 飞书 VIP 群用 Claude（最强推理）
    - platform: feishu
      channel_id: oc_abc123
      provider: claude
      api_key: sk-ant-xxx
      model: claude-sonnet-4-20250514

    # 飞书其他群用 Qwen（性价比高）
    - platform: feishu
      provider: qwen
      api_key: sk-qwen-xxx
      model: qwen-plus

    # Telegram 用 Gemini（免费额度大）
    - platform: telegram
      provider: gemini
      api_key: AIza-xxx
```

**Result / 结果：**

| Message Source | Provider | Model | Why |
|---|---|---|---|
| WeCom | **kimi** | `moonshot-v1-8k` | matches `platform: wecom` |
| Feishu #vip (oc_abc123) | **claude** | `claude-sonnet-4-20250514` | matches `platform: feishu` + `channel_id: oc_abc123` (most specific) |
| Feishu #general (oc_xyz789) | **qwen** | `qwen-plus` | matches `platform: feishu` (no channel_id match, falls to platform-only) |
| Telegram | **gemini** | `gemini-2.0-flash` | matches `platform: telegram`, model auto-set to gemini default |
| Discord | **deepseek** | `deepseek-chat` | no override → default |
| Slack | **deepseek** | `deepseek-chat` | no override → default |

---

### Override Resolution Algorithm / 覆盖解析算法

```
message arrives (platform="feishu", channel_id="oc_abc123")
  │
  ├─ Pass 1: scan overrides for platform="feishu" AND channel_id="oc_abc123"
  │   └─ found? → use this override ✓
  │
  ├─ Pass 2: scan overrides for platform="feishu" AND channel_id="" (empty)
  │   └─ found? → use this override ✓
  │
  └─ Pass 3: no match → use default ai config ✓
```

### Override Field Merge Rules / 字段合并规则

```
if override.provider != default.provider:
    # Provider changed — clear inherited base_url and model
    # 切换了 provider — 清空继承的 base_url 和 model
    base_url = ""              → new provider's default
    model    = ""              → new provider's default

# Then apply non-empty override fields
# 然后应用 override 中非空的字段
if override.api_key != "":   api_key  = override.api_key
if override.base_url != "":  base_url = override.base_url   # explicit override
if override.model != "":     model    = override.model       # explicit override
```

---

### Common Mistakes / 常见错误

**Wrong: Switching provider without specifying base_url / 错误：切换 provider 但用了旧 base_url**

```yaml
# ❌ BAD — base_url will be cleared because provider changed,
#          but if you set base_url from default, it won't carry over
ai:
  provider: deepseek
  api_key: sk-deepseek-xxx
  base_url: https://api.deepseek.com/v1    # this will NOT carry to kimi

  overrides:
    - platform: wecom
      provider: kimi          # provider changes → base_url auto-cleared ✓
      api_key: sk-kimi-xxx    # kimi will use its own default base_url ✓
      # No need to worry — this is handled automatically
```

**Wrong: Expecting channel override without platform / 错误：期望仅 channel_id 匹配**

```yaml
# ❌ BAD — channel_id without platform won't match anything
ai:
  overrides:
    - channel_id: C12345      # missing platform! this override will never match
      provider: claude

# ✅ GOOD — always include platform
ai:
  overrides:
    - platform: slack
      channel_id: C12345
      provider: claude
```

---

## Ollama (Local Models / 本地模型)

Ollama 在本地运行开源大模型，无需 API 密钥，默认监听 `http://localhost:11434`。

Ollama runs open-source LLMs locally with no API key required, listening on `http://localhost:11434` by default.

### Setup / 安装

```bash
# Install Ollama / 安装
# macOS
brew install ollama

# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model / 拉取模型
ollama pull llama3.2

# Start the server (if not running as a service) / 启动服务
ollama serve

# Stop the server / 停止服务
# macOS (installed via brew or .dmg): quit from menu bar, or:
launchctl stop com.ollama.ollama
# Linux (systemd):
sudo systemctl stop ollama
# Foreground process: Ctrl+C
```

### Usage / 用法

```bash
# Default model (llama3.2) / 默认模型
lingti-bot relay --provider ollama

# Specify a model / 指定模型
lingti-bot relay --provider ollama --model mistral
lingti-bot relay --provider ollama --model qwen2.5:7b
lingti-bot relay --provider ollama --model deepseek-r1:8b

# Connect to a remote Ollama instance / 连接远程实例
lingti-bot relay --provider ollama --base-url http://192.168.1.100:11434/v1
```

### Available Models / 可用模型

Run `ollama list` to see installed models. Popular choices:

| Model | Size | Notes |
|-------|------|-------|
| `llama3.2` | 3B | Default, good general-purpose / 默认，通用 |
| `llama3.2:1b` | 1B | Lightweight / 轻量 |
| `mistral` | 7B | Strong reasoning / 推理能力强 |
| `qwen2.5:7b` | 7B | Good Chinese support / 中文支持好 |
| `deepseek-r1:8b` | 8B | Code & reasoning / 代码与推理 |

## Notes / 说明

- All non-Claude providers use the **OpenAI-compatible API** format, making it easy to add new providers.
- 除 Claude 外，所有 provider 均使用 **OpenAI 兼容 API** 格式，便于扩展。
- `siliconflow` is an aggregator platform that provides access to many open-source models (Qwen, DeepSeek, Llama, etc.) through a single API key.
- `siliconflow` 是一个聚合平台，通过一个 API Key 即可访问多种开源模型（Qwen、DeepSeek、Llama 等）。
- You can always override the default model with `--model` and the default API URL with `--base-url`.
- 可通过 `--model` 和 `--base-url` 覆盖默认模型和 API 地址。
