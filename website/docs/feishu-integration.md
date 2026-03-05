# 飞书集成指南

本文档介绍如何为 lingti-bot 消息路由器配置飞书/Lark 集成。

## 架构

```
┌─────────────────────────────────────────────────────────┐
│                    lingti-bot Router                     │
├─────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │    Router    │  │    Agent     │  │  MCP Tools   │   │
│  │   (消息路由)  │──│  (Claude)    │──│   (工具)     │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────┘
          │
          ▼
┌─────────────────┐
│      飞书       │
│   (WebSocket)   │
└─────────────────┘
```

## 前置条件

- 拥有创建企业自建应用权限的飞书账号
- Anthropic API Key (用于 Claude)
- 已编译的 lingti-bot 二进制文件

## 第一步：创建飞书应用

1. 访问 [飞书开放平台](https://open.feishu.cn/app)
2. 点击 **「创建企业自建应用」**
3. 填写应用信息：
   - 应用名称：`lingti-bot`
   - 应用描述：填写机器人描述
   - 应用图标：上传图标
4. 点击 **「创建」**

## 第二步：获取应用凭证

1. 在应用设置中，进入 **「凭证与基础信息」**
2. 复制以下信息：
   - **App ID** - 应用唯一标识
   - **App Secret** - 点击「查看」显示后复制

> **重要**：请妥善保管 App Secret，切勿提交到版本控制系统。

## 第三步：配置机器人能力

1. 进入 **「添加应用能力」** → **「机器人」**
2. 开启 **「启用机器人」**
3. 配置机器人信息：
   - 机器人名称：`lingti-bot`
   - 机器人描述：填写描述

## 第四步：配置权限

1. 进入 **「权限管理」**
2. 添加以下权限：

| 权限名称 | Scope ID | 说明 |
|---------|----------|------|
| 获取与发送单聊、群组消息 | `im:message` | 收发消息 |
| 获取用户基本信息 | `contact:user.base:readonly` | 读取用户名 |
| 获取群组信息 | `im:chat:readonly` | 读取群信息 |

3. 点击 **「批量开通」**

## 第五步：首次启动 Bot 建立连接（重要）

> **注意**：飞书要求应用先与平台建立首次通信，之后才能在后台配置长连接模式。如果跳过此步骤，第六步的长连接选项将无法设置。

1. 使用第二步获取的 App ID 和 App Secret，启动一次 lingti-bot gateway：

```bash
export FEISHU_APP_ID="cli_your_app_id"
export FEISHU_APP_SECRET="your_app_secret"
export ANTHROPIC_API_KEY="sk-ant-your-api-key"

lingti-bot gateway
```

2. 观察日志输出，确认已尝试连接飞书
3. 按 `Ctrl+C` 停止程序
4. 此时飞书后台会记录应用的首次通信，解锁长连接配置选项

## 第六步：启用长连接模式

> **注意**：必须先完成第五步的首次连接，否则此选项无法设置。

1. 返回飞书开放平台，进入应用的 **「机器人」** 设置页面
2. 找到 **「消息接收方式」**
3. 选择 **「使用长连接接收消息」**
4. 点击 **「保存」**

## 第七步：配置事件订阅

1. 进入 **「事件与回调」** → **「事件配置」**
2. 在页面顶部找到 **「订阅方式」**，选择 **「使用长连接接收事件」**
3. 点击 **「添加事件」**，搜索并添加以下事件：

| 事件名称 | 事件 ID | 说明 |
|---------|---------|------|
| 接收消息 | `im.message.receive_v1` | 接收发送给机器人的消息 |

4. 点击 **「保存」**

## 第八步：发布应用

1. 进入 **「版本管理与发布」**
2. 点击 **「创建版本」**
3. 填写版本信息
4. 点击 **「申请发布」**
5. 如果你是管理员，审批发布申请

> 开发测试阶段，可以在「开发中」状态下使用应用（仅限自己的账号）。

## 第九步：运行 lingti-bot Router

### 使用环境变量

```bash
export FEISHU_APP_ID="cli_your_app_id"
export FEISHU_APP_SECRET="your_app_secret"
export ANTHROPIC_API_KEY="sk-ant-your-api-key"
# 可选配置
export ANTHROPIC_BASE_URL="https://your-proxy.com/v1"  # 自定义 API 地址
export ANTHROPIC_MODEL="claude-sonnet-4-20250514"       # 指定模型

lingti-bot gateway
```

### 使用命令行参数

```bash
lingti-bot gateway \
  --feishu-app-id "cli_your_app_id" \
  --feishu-app-secret "your_app_secret" \
  --api-key "sk-ant-your-api-key" \
  --base-url "https://your-proxy.com/v1" \
  --model "claude-sonnet-4-20250514"
```

### 使用 .env 文件

创建 `.env` 文件：

```bash
FEISHU_APP_ID=cli_your_app_id
FEISHU_APP_SECRET=your_app_secret
ANTHROPIC_API_KEY=sk-ant-your-api-key
ANTHROPIC_BASE_URL=https://your-proxy.com/v1
ANTHROPIC_MODEL=claude-sonnet-4-20250514
```

然后运行：

```bash
source .env && lingti-bot gateway
```

### 同时运行 Slack 和飞书

可以同时运行多个平台：

```bash
export SLACK_BOT_TOKEN="xoxb-..."
export SLACK_APP_TOKEN="xapp-..."
export FEISHU_APP_ID="cli_..."
export FEISHU_APP_SECRET="..."
export ANTHROPIC_API_KEY="sk-ant-..."
export ANTHROPIC_BASE_URL="https://your-proxy.com/v1"  # 可选
export ANTHROPIC_MODEL="claude-sonnet-4-20250514"       # 可选

lingti-bot gateway
```

## 第十步：测试集成

1. 打开飞书 App
2. 找到机器人：
   - 在搜索栏搜索机器人名称
   - 或进入 **「工作台」** 找到你的机器人应用
3. 开始对话：
   - **私聊**：直接发送消息给机器人
   - **群聊**：将机器人添加到群组，然后 @机器人

示例消息：
- `今天有什么日程？`
- `@lingti-bot 列出 ~/Desktop 的文件`
- `@lingti-bot 查看系统信息`

## 可用功能

连接成功后，机器人可以：

| 类别 | 示例 |
|-----|------|
| **日历** | "今天有什么日程？"、"明天下午2点安排一个会议" |
| **文件** | "列出 ~/Downloads 的文件"、"查找桌面上的旧文件" |
| **系统** | "CPU 使用率是多少？"、"查看磁盘空间" |
| **Shell** | "运行 `ls -la`"、"查看 git status" |
| **进程** | "列出运行中的进程"、"哪个程序占用内存最多？" |

## 故障排除

### 机器人无响应

1. 检查 App ID 和 App Secret 是否正确
2. 查看 router 日志确认机器人正在运行
3. 确保飞书应用设置中已启用 WebSocket 模式
4. 检查应用是否已发布，或者是否使用正确的账号测试

### "应用未建立长连接" 错误

1. 进入 **「事件与回调」** → **「事件配置」**
2. 确认 **「订阅方式」** 已选择 **「使用长连接接收事件」**
3. 确认已添加至少一个事件（如 `im.message.receive_v1`）
4. 点击保存后重新启动 router

### "failed to verify credentials" 错误

`FEISHU_APP_ID` 或 `FEISHU_APP_SECRET` 无效。请在飞书开放平台重新检查凭证。

### 机器人只在私聊有效，群聊无响应

请确认：
1. 机器人已被添加到群组
2. 在群消息中 @机器人
3. `im:message` 权限已启用

### 收不到消息

1. 确认已启用 **「使用长连接接收消息」**
2. 检查是否已订阅 `im.message.receive_v1` 事件
3. 确保包含机器人能力的应用版本已发布

### 权限不足错误

1. 在飞书开放平台进入「权限管理」
2. 确保所有必需权限已启用
3. 如果最近添加了新权限，可能需要创建新版本

## 消息格式说明

- **私聊**：机器人响应所有私聊消息
- **群聊**：机器人仅在被 @提及时响应
- **@提及**：消息文本中的 `@机器人名` 会在处理前自动移除

## 安全注意事项

- 切勿将 App Secret 提交到版本控制
- 使用环境变量或密钥管理器
- 限制应用安装范围到可信组织
- 定期审查机器人权限
- 生产环境建议配置 IP 白名单

## 作为服务运行

### macOS (launchd)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.lingti.bot.router</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/lingti-bot</string>
        <string>router</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>FEISHU_APP_ID</key>
        <string>cli_...</string>
        <key>FEISHU_APP_SECRET</key>
        <string>...</string>
        <key>ANTHROPIC_API_KEY</key>
        <string>sk-ant-...</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/lingti-bot-router.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/lingti-bot-router.log</string>
</dict>
</plist>
```

### Linux (systemd)

```ini
[Unit]
Description=Lingti Bot Router
After=network.target

[Service]
Type=simple
Environment=FEISHU_APP_ID=cli_...
Environment=FEISHU_APP_SECRET=...
Environment=ANTHROPIC_API_KEY=sk-ant-...
ExecStart=/usr/local/bin/lingti-bot gateway
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## Lark（国际版）与飞书（中国版）

本集成同时支持：
- **飞书** - 中国版，访问 `open.feishu.cn`
- **Lark** - 国际版，访问 `open.larksuite.com`

SDK 会自动处理地区差异。请使用对应版本的开发者控制台。

## 参考链接

- [飞书开放平台](https://open.feishu.cn/)
- [Lark 开发者文档](https://open.larksuite.com/document)
- [机器人开发指南](https://open.feishu.cn/document/home/develop-a-bot-in-5-minutes/create-an-app)
- [事件订阅指南](https://open.feishu.cn/document/ukTMukTMukTM/uUTNz4SN1MjL1UzM)
- [Lark SDK for Go](https://github.com/larksuite/oapi-sdk-go)
