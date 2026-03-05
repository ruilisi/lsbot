# 浏览器自动化

> lingti-bot 内置完整的浏览器自动化能力，基于 **Chrome DevTools Protocol (CDP)** 和 **go-rod**，采用**快照-操作（Snapshot-then-Act）**模式，让 AI 能够像人一样操作浏览器。

> 📋 **AI agent 操作规则（搜索行为、弹窗处理、批量操作、连接模式等）：[浏览器 AI 操作规则](browser-agent-rules.md)**

---

## 核心能力

- 控制真实 Chrome/Brave/Edge 浏览器（有界面或无头模式）
- **连接已有 Chrome** — 无需新开窗口，直接接管正在使用的浏览器
- 读取页面无障碍树（Accessibility Tree），精准定位元素
- 点击、输入、按键、滚动、拖拽
- 多标签页管理
- 截图（视口或整页）
- 执行任意 JavaScript
- 批量点击（适合爬取、批量操作）

---

## 连接已有 Chrome（推荐工作流）

默认情况下，每次触发浏览器时 lingti-bot 会启动一个新的独立 Chrome 窗口。通过以下配置，可以让 bot 直接在你**正在使用的 Chrome 里**开新标签页操作，实现人机共享浏览器。

### 第一步：用调试端口启动 Chrome

Chrome 必须以 `--remote-debugging-port` 参数启动，才能接受 CDP 连接。

**macOS：**

```bash
# 新开一个带调试端口的 Chrome 窗口（不影响已有进程）
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.lingti-bot/chrome-profile"
```

> **推荐：** 使用独立的 `--user-data-dir`，避免与个人 Chrome 账号/扩展产生冲突。

**Linux：**

```bash
google-chrome \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.lingti-bot/chrome-profile"
```

**验证端口已开放：**

```bash
curl http://localhost:9222/json/version
```

成功响应示例：

```json
{
  "Browser": "Chrome/121.0.6167.160",
  "Protocol-Version": "1.3",
  "webSocketDebuggerUrl": "ws://localhost:9222/devtools/browser/..."
}
```

### 第二步：配置 lingti-bot

在 `~/.lingti.yaml` 中添加：

```yaml
browser:
  cdp_url: "127.0.0.1:9222"
```

之后所有 `browser_navigate`、`browser_click` 等操作都会在这个 Chrome 里执行，不再另开新窗口。

### 配置优先级

`EnsureRunning()` 按以下顺序决定使用哪个浏览器：

```
1. cfg.Browser.CDPURL  （~/.lingti.yaml 中的 cdp_url）
2. 127.0.0.1:9222      （well-known 默认调试端口，无需配置）
3. 启动新 Chrome 实例   （fallback）
```

---

## 快速上手

```
"打开知乎首页"                        → browser_navigate url="https://www.zhihu.com"
"看看页面上有什么"                    → browser_snapshot
"点击搜索框并搜索 Go 语言"           → browser_type ref=3 text="Go 语言" submit=true
"截图保存"                            → browser_screenshot
"打开新标签页看看微博"                → browser_tab_open url="https://www.weibo.com"
```

---

## 工具完整参考

### 生命周期管理

#### `browser_start` — 启动或连接浏览器

| 参数 | 类型 | 说明 |
|------|------|------|
| `headless` | bool | 无头模式（无界面），默认 false |
| `url` | string | 启动后立即导航的 URL |
| `executable_path` | string | Chrome 可执行文件路径（留空自动检测） |
| `cdp_url` | string | 连接已有 Chrome 的 CDP 地址（如 `127.0.0.1:9222`） |

```
# 启动有界面浏览器
browser_start headless=false

# 无头模式
browser_start headless=true

# 连接已有 Chrome（需已用 --remote-debugging-port 启动）
browser_start cdp_url="127.0.0.1:9222"

# 启动并直接导航
browser_start url="https://www.zhihu.com"
```

#### `browser_stop` — 关闭浏览器

如果是连接到已有 Chrome（`cdp_url` 模式），只断开连接，**不关闭浏览器**。

#### `browser_status` — 查看浏览器状态

返回：

```json
{
  "running": true,
  "headless": false,
  "connected": true,
  "pages": 3,
  "active_url": "https://www.zhihu.com"
}
```

`connected: true` 表示当前是连接到已有 Chrome（不是 bot 自己启动的）。

---

### 导航与内容

#### `browser_navigate` — 导航到 URL

```
browser_navigate url="https://www.baidu.com"
```

- 自动等待页面加载完成（`load` 事件）
- 如果浏览器未启动，自动按优先级连接/启动

#### `browser_snapshot` — 获取页面无障碍快照

返回页面的可交互元素列表，每个元素带数字编号（ref）：

```
[1] link "首页"
[2] link "发现"
[3] textbox "搜索"
[4] button "搜索"
[5] heading "今日推荐"
  [6] link "为什么 Go 语言这么流行？"
  [7] link "深度学习入门指南"
```

**ref 规则：**
- 每次 snapshot 重新编号，导航后必须重新 snapshot
- 只包含可交互元素和重要内容节点
- 缩进表示层级关系

#### `browser_screenshot` — 截图

| 参数 | 类型 | 说明 |
|------|------|------|
| `path` | string | 保存路径，默认 `~/Desktop/browser_screenshot_<时间戳>.png` |
| `full_page` | bool | true = 整页截图，false = 当前视口，默认 false |

```
browser_screenshot
browser_screenshot path="/tmp/result.png" full_page=true
```

---

### 元素交互

> 所有交互工具都需要先执行 `browser_snapshot` 获取 ref 编号。

#### `browser_click` — 点击元素

```
browser_click ref=4
```

- 自动滚动到元素可见位置
- 等待元素可交互
- ref 失效时自动重新 snapshot 并重试一次

#### `browser_type` — 输入文本

| 参数 | 类型 | 说明 |
|------|------|------|
| `ref` | number | 必需，元素 ref 编号 |
| `text` | string | 必需，输入内容 |
| `submit` | bool | true = 输入后按 Enter，默认 false |

```
browser_type ref=3 text="lingti-bot"
browser_type ref=3 text="搜索内容" submit=true
```

#### `browser_press` — 按键

支持的按键：

| 按键名 | 说明 |
|--------|------|
| `Enter` | 回车 |
| `Tab` | 制表符 / 切换焦点 |
| `Escape` | 取消 |
| `Backspace` | 退格 |
| `Delete` | 删除 |
| `Space` | 空格 |
| `ArrowUp/Down/Left/Right` | 方向键 |
| `Home` / `End` | 行首 / 行尾 |
| `PageUp` / `PageDown` | 翻页 |

```
browser_press key="Enter"
browser_press key="Tab"
browser_press key="Escape"
```

#### `browser_execute_js` — 执行 JavaScript

```
browser_execute_js script="return document.title"
browser_execute_js script="window.scrollTo(0, document.body.scrollHeight)"
browser_execute_js script="return document.querySelectorAll('a').length"
```

#### `browser_click_all` — 批量点击

适合批量操作（如全选、批量关闭通知等）。

| 参数 | 类型 | 说明 |
|------|------|------|
| `selector` | string | CSS 选择器，匹配要点击的元素 |
| `delay_ms` | number | 每次点击间隔毫秒数，默认 500 |
| `skip_selector` | string | 跳过匹配此选择器的元素（可选） |

```
browser_click_all selector=".notification-item .close-btn" delay_ms=200
```

---

### 标签页管理

#### `browser_tabs` — 列出所有标签页

```json
[
  {"target_id": "abc123", "url": "https://www.zhihu.com", "title": "知乎"},
  {"target_id": "def456", "url": "https://www.weibo.com", "title": "微博"}
]
```

#### `browser_tab_open` — 打开新标签页

```
browser_tab_open url="https://www.weibo.com"
browser_tab_open                              # 打开空白标签页
```

#### `browser_tab_close` — 关闭标签页

```
browser_tab_close target_id="abc123"
browser_tab_close                             # 关闭当前活跃标签页
```

---

## 典型使用场景

### 场景一：信息查询

```
用户: "帮我查一下今天的 BTC 价格"

bot:
1. browser_navigate url="https://www.coindesk.com"
2. browser_snapshot
   → [1] heading "Bitcoin Price" [2] text "$67,234.50" ...
3. 直接返回价格信息，无需截图
```

### 场景二：登录并操作

```
用户: "帮我登录知乎并关注 xxx"

bot:
1. browser_navigate url="https://www.zhihu.com/signin"
2. browser_snapshot
   → [1] textbox "手机号或邮箱" [2] textbox "密码" [3] button "登录"
3. browser_type ref=1 text="your@email.com"
4. browser_type ref=2 text="yourpassword"
5. browser_click ref=3
6. browser_navigate url="https://www.zhihu.com/people/xxx"
7. browser_snapshot
   → [N] button "关注"
8. browser_click ref=N
```

### 场景三：网页内容提取

```
用户: "抓取这个页面所有文章标题"

bot:
1. browser_navigate url="https://example.com/blog"
2. browser_snapshot
   → 看到所有 heading 和 link 元素
3. 直接从快照中提取标题信息，返回列表
```

### 场景四：表单填写

```
用户: "帮我填写这个报名表"

bot:
1. browser_navigate url="https://example.com/register"
2. browser_snapshot
3. browser_type ref=1 text="张三"          # 姓名
4. browser_type ref=2 text="138xxxx1234"   # 手机
5. browser_type ref=3 text="example@qq.com" # 邮箱
6. browser_click ref=10                     # 提交按钮
7. browser_screenshot                       # 截图确认
```

### 场景五：批量操作

```
用户: "把我邮箱里所有营销邮件全部删除"

bot:
1. browser_navigate url="https://mail.example.com"
2. browser_snapshot → 找到邮件列表
3. browser_click_all selector=".email-item.marketing input[type=checkbox]"
4. browser_click ref=<删除按钮>
```

### 场景六：监控页面变化

```
用户: "每隔5分钟截一张这个页面的图"

bot:
1. browser_navigate url="https://example.com/dashboard"
2. browser_screenshot path="/tmp/monitor_1.png"
3. (等待)
4. browser_screenshot path="/tmp/monitor_2.png"
```

---

## 配置参考

`~/.lingti.yaml` 中的 `browser` 配置节：

```yaml
browser:
  # 连接已有 Chrome（推荐）
  # Chrome 需以 --remote-debugging-port=9222 启动
  cdp_url: "127.0.0.1:9222"

  # 浏览器窗口大小
  # "fullscreen" = 全屏（默认）
  # "1920x1080"  = 指定分辨率
  screen_size: "1920x1080"
```

---

## 技术架构

```
用户自然语言指令
      ↓
  AI Agent（理解意图，规划工具调用序列）
      ↓
MCP Tools (internal/tools/browser.go)
      ↓
Browser Manager (internal/browser/browser.go)
  ├── EnsureRunning()   → cdp_url > :9222 > 新启动
  ├── Start()           → 启动或连接
  └── ActivePage()      → 获取当前活跃页面
      ↓
Snapshot Engine (internal/browser/snapshot.go)
  └── CDP Accessibility.getFullAXTree → ref 映射
      ↓
Action Engine (internal/browser/actions.go)
  └── ref → BackendDOMNodeID → DOM 元素 → 交互
      ↓
go-rod/rod（Chrome DevTools Protocol）
      ↓
Chrome / Brave / Edge（有界面 或 无头）
```

### 为什么用无障碍树而不是 CSS 选择器？

| | 无障碍树（本项目） | CSS 选择器 |
|--|--|--|
| **稳定性** | 不受样式重构影响 | 类名变化即失效 |
| **AI 可读性** | role + name 语义清晰 | 需要理解 DOM 结构 |
| **简洁度** | `[3] button "登录"` | `.login-form > div > button.btn-primary` |
| **跨站通用** | 标准 ARIA 规范 | 每个网站不同 |

### Ref 生命周期

```
browser_snapshot      →  生成 ref 映射（存储在内存）
browser_click ref=3   →  通过 BackendDOMNodeID 定位 DOM 元素
browser_navigate      →  页面变化，旧 ref 全部失效
browser_snapshot      →  必须重新获取
```

---

## 故障排除

### 找不到浏览器

```
failed to launch browser: no chrome executable found
```

**解决：** 安装 Chrome、Brave 或 Edge，或用 `executable_path` 指定路径：

```
browser_start executable_path="/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"
```

---

### CDP 连接失败

```
failed to resolve CDP address 127.0.0.1:9222
```

**解决：** Chrome 未以调试端口启动。执行：

```bash
/Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
  --remote-debugging-port=9222 \
  --user-data-dir="$HOME/.lingti-bot/chrome-profile"
```

验证：`curl http://localhost:9222/json/version`

---

### ref 失效

```
ref 5 not found in snapshot
```

**原因：** 页面内容已变化（导航、动态加载等），旧 ref 不再有效。

**解决：** 重新执行 `browser_snapshot` 获取新 ref 编号。

---

### 快照为空

**原因：** 页面还在加载，或为纯 canvas/WebGL 应用（无障碍树为空）。

**解决：**
1. 等待页面稳定后重试 `browser_snapshot`
2. 对于无障碍树为空的页面，改用 `browser_execute_js` 提取内容

---

### 无头模式下页面渲染异常

部分网站会检测 headless 并显示验证码或重定向。改用有界面模式：

```
browser_start headless=false
```

---

### Linux 服务器无 GUI

确保使用 headless 模式并安装浏览器依赖：

```bash
# Ubuntu/Debian
apt-get install -y chromium-browser

# 启动 bot 时浏览器会自动以 headless 模式运行
```

---

## 与其他方案对比

| | lingti-bot | Playwright/Node.js | Selenium |
|--|--|--|--|
| **语言** | Go（单二进制） | Node.js | Java/Python |
| **依赖** | 仅需 Chrome | Node.js + 浏览器驱动 | JVM + 驱动 |
| **部署** | 复制一个文件 | npm install | 配置复杂 |
| **AI 集成** | 原生 MCP | 需要额外封装 | 需要额外封装 |
| **无障碍树** | 原生支持 | 支持 | 有限支持 |
| **连接已有浏览器** | ✅ cdp_url | ✅ connectOverCDP | ❌ |
