package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ruilisi/lsbot/internal/agent/history"
	"github.com/ruilisi/lsbot/internal/agent/mcpclient"
	"github.com/ruilisi/lsbot/internal/config"
	cronpkg "github.com/ruilisi/lsbot/internal/cron"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/security"
	"github.com/ruilisi/lsbot/internal/skills"
	"github.com/ruilisi/lsbot/internal/userprofile"
)

// Agent processes messages using AI providers and tools
type Agent struct {
	provider           Provider
	fallbackProviders  []Provider // ordered fallbacks tried when quota is exhausted
	memory             *ConversationMemory
	sessions           *SessionStore
	autoApprove        bool
	customInstructions string
	cronScheduler      *cronpkg.Scheduler
	currentMsg         router.Message // set during HandleMessage for cron_create context
	cronCreatedCount   int            // tracks cron_create calls per HandleMessage turn
	pathChecker        *security.PathChecker
	disableFileTools   bool
	maxToolRounds      int
	callTimeoutSecs    int
	mcpManager         *mcpclient.Manager
}

// Config holds agent configuration
type Config struct {
	Provider           string // "claude" or "deepseek" (default: "claude")
	APIKey             string
	BaseURL            string                   // Custom API base URL (optional)
	Model              string                   // Model name (optional, uses provider default)
	AutoApprove        bool                     // Skip all confirmation prompts (default: false)
	CustomInstructions string                   // Additional instructions appended to system prompt (optional)
	AllowedPaths       []string                 // Restrict file/shell operations to these directories (empty = no restriction)
	DisableFileTools   bool                     // Completely disable all file operation tools
	MaxToolRounds      int                      // Max tool-call iterations per message (0 = use default 100)
	CallTimeoutSecs    int                      // Base timeout in seconds for each AI API call (0 = use default 90s base)
	MCPServers         []mcpclient.ServerConfig // External MCP servers to connect to
	AllowTools         []string                 // Tool whitelist; empty = allow all
	DenyTools          []string                 // Tool blacklist; applied after allowlist
	Workspace          string                   // Working directory for this agent
	Fallbacks          []Config                 // Ordered fallback provider configs tried when quota is exhausted
}

// New creates a new Agent with the specified provider
func New(cfg Config) (*Agent, error) {
	if cfg.APIKey == "" && strings.ToLower(cfg.Provider) != "ollama" {
		return nil, fmt.Errorf("API key is required")
	}

	provider, err := createProvider(cfg)
	if err != nil {
		return nil, err
	}

	maxRounds := cfg.MaxToolRounds
	if maxRounds <= 0 {
		maxRounds = 100
	}

	var fallbackProviders []Provider
	for _, fbCfg := range cfg.Fallbacks {
		fbProvider, err := createProvider(fbCfg)
		if err != nil {
			logger.Warn("[Agent] Failed to create fallback provider %s: %v", fbCfg.Provider, err)
			continue
		}
		fallbackProviders = append(fallbackProviders, fbProvider)
	}

	return &Agent{
		provider:           provider,
		fallbackProviders:  fallbackProviders,
		memory:             NewMemory(50, 60*time.Minute), // Keep 50 messages, 60 min TTL
		sessions:           NewSessionStore(),
		autoApprove:        cfg.AutoApprove,
		customInstructions: cfg.CustomInstructions,
		pathChecker:        security.NewPathChecker(cfg.AllowedPaths),
		disableFileTools:   cfg.DisableFileTools,
		maxToolRounds:      maxRounds,
		callTimeoutSecs:    cfg.CallTimeoutSecs,
		mcpManager:         mcpclient.New(cfg.MCPServers),
	}, nil
}

// init sets up the FTS5 table for session search once per process.
func init() { go initHistory() }

// initHistory ensures the FTS5 table exists. Called lazily on first use.
func initHistory() {
	if hs, err := history.Global(); err == nil {
		if err := hs.EnsureFTS(); err != nil {
			logger.Warn("[Agent] FTS5 init: %v", err)
		}
	}
}

// openaiCompatProviders maps provider names to their default base URLs and models.
var openaiCompatProviders = map[string]struct {
	baseURL string
	model   string
}{
	"minimax":     {"https://api.minimax.chat/v1", "MiniMax-Text-01"},
	"doubao":      {"https://ark.cn-beijing.volces.com/api/v3", "doubao-pro-32k"},
	"zhipu":       {"https://open.bigmodel.cn/api/paas/v4", "glm-4-flash"},
	"openai":      {"https://api.openai.com/v1", "gpt-4o"},
	"yi":          {"https://api.lingyiwanwu.com/v1", "yi-large"},
	"stepfun":     {"https://api.stepfun.com/v1", "step-2-16k"},
	"siliconflow": {"https://api.siliconflow.cn/v1", "Qwen/Qwen2.5-72B-Instruct"},
	"grok":        {"https://api.x.ai/v1", "grok-2-latest"},
	"baichuan":    {"https://api.baichuan-ai.com/v1", "Baichuan4"},
	"spark":       {"https://spark-api-open.xf-yun.com/v1", "generalv3.5"},
	"hunyuan":     {"https://api.hunyuan.cloud.tencent.com/v1", "hunyuan-turbos-latest"},
	"ollama":      {"http://localhost:11434/v1", "llama3.2"},
}

// openaiCompatAliases maps alternative names to canonical provider names.
var openaiCompatAliases = map[string]string{
	"glm":         "zhipu",
	"chatglm":     "zhipu",
	"gpt":         "openai",
	"chatgpt":     "openai",
	"lingyiwanwu": "yi",
	"wanwu":       "yi",
	"google":      "gemini",
	"xai":         "grok",
	"bytedance":   "doubao",
	"volcengine":  "doubao",
	"iflytek":     "spark",
	"xunfei":      "spark",
	"tencent":     "hunyuan",
	"hungyuan":    "hunyuan",
}

// inferProviderFromModel guesses the provider from a model name when provider is unset.
func inferProviderFromModel(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "kimi-") || strings.HasPrefix(m, "moonshot-"):
		return "kimi"
	case strings.HasPrefix(m, "deepseek-"):
		return "deepseek"
	case strings.HasPrefix(m, "qwen-"):
		return "qwen"
	case strings.HasPrefix(m, "glm-"):
		return "zhipu"
	case strings.HasPrefix(m, "gpt-") || strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3"):
		return "openai"
	case strings.HasPrefix(m, "gemini-"):
		return "gemini"
	case strings.HasPrefix(m, "claude-"):
		return "claude"
	case strings.HasPrefix(m, "grok-"):
		return "grok"
	case strings.HasPrefix(m, "doubao-"):
		return "doubao"
	case strings.HasPrefix(m, "step-"):
		return "stepfun"
	case strings.HasPrefix(m, "yi-"):
		return "yi"
	case strings.HasPrefix(m, "hunyuan-"):
		return "hunyuan"
	case strings.HasPrefix(m, "minimax-") || strings.HasPrefix(m, "minimax-text"):
		return "minimax"
	}
	return ""
}

// createProvider creates the appropriate AI provider based on config
func createProvider(cfg Config) (Provider, error) {
	name := strings.ToLower(cfg.Provider)
	if name == "" && cfg.Model != "" {
		name = inferProviderFromModel(cfg.Model)
	}
	if canonical, ok := openaiCompatAliases[name]; ok {
		name = canonical
	}

	switch name {
	case "deepseek":
		return NewDeepSeekProvider(DeepSeekConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "kimi", "moonshot":
		return NewKimiProvider(KimiConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "qwen", "qianwen", "tongyi":
		return NewQwenProvider(QwenConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "claude", "anthropic", "":
		return NewClaudeProvider(ClaudeConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	case "gemini":
		return NewGeminiProvider(GeminiConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
		})
	default:
		// Check OpenAI-compatible providers
		if defaults, ok := openaiCompatProviders[name]; ok {
			return NewOpenAICompatProvider(OpenAICompatConfig{
				ProviderName: name,
				APIKey:       cfg.APIKey,
				BaseURL:      cfg.BaseURL,
				Model:        cfg.Model,
				DefaultURL:   defaults.baseURL,
				DefaultModel: defaults.model,
			})
		}
		return nil, fmt.Errorf("unknown provider: %s (supported: claude, deepseek, kimi, qwen, minimax, doubao, zhipu, openai, gemini, yi, stepfun, siliconflow, grok, baichuan, spark, hunyuan, ollama)", cfg.Provider)
	}
}

// isQuotaError returns true when the error is due to insufficient API quota or balance.
func isQuotaError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	quotaKeywords := []string{
		"quota", "balance", "insufficient", "credits",
		"余额", "欠费", "账户余额", "insufficient_quota",
	}
	for _, kw := range quotaKeywords {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	// HTTP 402 Payment Required is always a quota error
	return strings.Contains(msg, "402")
}

// chatWithFallback calls the primary provider and, if a quota error is detected,
// retries with each fallback provider in order.
func (a *Agent) chatWithFallback(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	resp, err := a.provider.Chat(ctx, req)
	if err == nil || !isQuotaError(err) {
		return resp, err
	}
	for _, fb := range a.fallbackProviders {
		logger.Info("[Agent] Provider %s quota insufficient, falling back to %s", a.provider.Name(), fb.Name())
		resp, err = fb.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !isQuotaError(err) {
			return resp, err
		}
	}
	return ChatResponse{}, fmt.Errorf("all providers exhausted: %w", err)
}

// handleBuiltinCommand handles special commands without calling AI
func (a *Agent) handleBuiltinCommand(msg router.Message) (router.Response, bool) {
	text := strings.TrimSpace(msg.Text)
	textLower := strings.ToLower(text)
	convKey := ConversationKey(msg.Platform, msg.ChannelID, msg.UserID)

	// Exact match commands
	switch textLower {
	case "/whoami", "whoami", "我是谁", "我的id":
		return router.Response{
			Text: fmt.Sprintf("用户信息:\n- 用户ID: %s\n- 用户名: %s\n- 平台: %s\n- 频道ID: %s",
				msg.UserID, msg.Username, msg.Platform, msg.ChannelID),
		}, true

	case "/help", "help", "帮助", "/commands":
		return router.Response{
			Text: `可用命令:

会话管理:
  /new, /reset    开始新对话，清除历史
  /status         查看当前会话状态

思考模式:
  /think off      关闭深度思考
  /think low      简单思考
  /think medium   中等思考（默认）
  /think high     深度思考

显示设置:
  /verbose on     显示详细执行过程
  /verbose off    隐藏执行过程

其他:
  /whoami         查看用户信息
  /model          查看当前模型
  /tools          列出可用工具
  /help           显示帮助

直接用自然语言和我对话即可！`,
		}, true

	case "/new", "/reset", "/clear", "新对话", "清除历史":
		a.memory.Clear(convKey)
		a.sessions.Clear(convKey)
		return router.Response{
			Text: "已开始新对话，历史记录和会话设置已重置。",
		}, true

	case "/status", "状态":
		history := a.memory.GetHistory(convKey)
		settings := a.sessions.Get(convKey)
		return router.Response{
			Text: fmt.Sprintf(`会话状态:
- 平台: %s
- 用户: %s
- 历史消息: %d 条
- 思考模式: %s
- 详细模式: %v
- AI 模型: %s`,
				msg.Platform, msg.Username, len(history),
				settings.ThinkingLevel, settings.Verbose, a.provider.Name()),
		}, true

	case "/model", "模型":
		return router.Response{
			Text: fmt.Sprintf("当前模型: %s", a.provider.Name()),
		}, true

	case "/tools", "工具", "工具列表":
		toolsText := `可用工具:

📁 文件操作:
  file_send, file_list, file_read, file_write, file_trash, file_list_old

📅 日历 (macOS):
  calendar_today, calendar_list_events, calendar_create_event
  calendar_search, calendar_delete

✅ 提醒事项 (macOS):
  reminders_list, reminders_add, reminders_complete, reminders_delete

📝 备忘录 (macOS):
  notes_list, notes_read, notes_create, notes_search

🌤 天气:
  weather_current, weather_forecast

🌐 网页:
  web_search, web_fetch, open_url

📋 剪贴板:
  clipboard_read, clipboard_write

🔔 通知:
  notification_send

📸 截图:
  screenshot

🎵 音乐 (macOS):
  music_play, music_pause, music_next, music_previous
  music_now_playing, music_volume, music_search

💻 系统:
  system_info, shell_execute, process_list

⏰ 定时任务:
  cron_create, cron_list, cron_delete, cron_pause, cron_resume` + formatSkillsSection()
		return router.Response{Text: toolsText}, true

	case "/verbose on", "详细模式开":
		a.sessions.SetVerbose(convKey, true)
		return router.Response{Text: "详细模式已开启"}, true

	case "/verbose off", "详细模式关":
		a.sessions.SetVerbose(convKey, false)
		return router.Response{Text: "详细模式已关闭"}, true

	case "/think off", "思考关":
		a.sessions.SetThinkingLevel(convKey, ThinkOff)
		return router.Response{Text: "思考模式已关闭"}, true

	case "/think low", "简单思考":
		a.sessions.SetThinkingLevel(convKey, ThinkLow)
		return router.Response{Text: "思考模式: 简单"}, true

	case "/think medium", "中等思考":
		a.sessions.SetThinkingLevel(convKey, ThinkMedium)
		return router.Response{Text: "思考模式: 中等"}, true

	case "/think high", "深度思考":
		a.sessions.SetThinkingLevel(convKey, ThinkHigh)
		return router.Response{Text: "思考模式: 深度"}, true
	}

	return router.Response{}, false
}

// SetCronScheduler sets the cron scheduler for the agent
func (a *Agent) SetCronScheduler(s *cronpkg.Scheduler) {
	a.cronScheduler = s
}

// ExecuteTool implements the cron.ToolExecutor interface
func (a *Agent) ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
	result := a.callToolDirect(ctx, toolName, arguments)
	return result, nil
}

// ExecutePrompt runs a full AI conversation with tools and returns the text response.
// Used by cron scheduler for prompt-based jobs.
func (a *Agent) ExecutePrompt(ctx context.Context, platform, channelID, userID, prompt string) (string, error) {
	msg := router.Message{
		Platform:  platform,
		ChannelID: channelID,
		UserID:    userID,
		Username:  "cron",
		Text:      prompt,
	}
	resp, err := a.HandleMessage(ctx, msg)
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}

// HandleMessage processes a message and returns a response
func (a *Agent) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	a.currentMsg = msg
	a.cronCreatedCount = 0
	logger.Info("[Agent] Processing message from %s: %s (provider: %s)", msg.Username, msg.Text, a.provider.Name())

	// Handle built-in commands
	if resp, handled := a.handleBuiltinCommand(msg); handled {
		return resp, nil
	}

	// Generate conversation key
	convKey := ConversationKey(msg.Platform, msg.ChannelID, msg.UserID)

	// Build the tools list
	tools := a.buildToolsList()

	// Get conversation history (in-memory first, populate from SQLite on first access)
	hist := a.memory.GetHistory(convKey)
	if len(hist) == 0 {
		if hs, err := history.Global(); err == nil {
			if loaded, err := hs.Load(convKey, 50); err == nil && len(loaded) > 0 {
				for _, m := range loaded {
					a.memory.AddMessage(convKey, Message{Role: m.Role, Content: m.Content})
				}
				hist = a.memory.GetHistory(convKey)
				logger.Debug("[Agent] Loaded %d messages from persistent history for %s", len(hist), convKey)
			}
		}
	}
	logger.Trace("[Agent] Conversation key: %s, history messages: %d", convKey, len(hist))

	// Create messages with history; auto-compress if the conversation is too long.
	messages := make([]Message, 0, len(hist)+1)
	messages = append(messages, hist...)
	if estimateTokens(messages) > compressTokenThreshold {
		messages = a.compressMessages(ctx, messages)
		// Sync compressed history back to in-memory store so future turns are accurate.
		a.memory.Replace(convKey, messages)
		logger.Info("[Agent] Context auto-compressed for conversation %s", convKey)
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: msg.Text,
	})

	// Get system info for context
	homeDir, _ := os.UserHomeDir()
	if homeDir == "" {
		homeDir = "~"
	}

	// Get session settings
	settings := a.sessions.Get(convKey)

	// For Claude provider, use real extended thinking API; for others, use prompt-based
	var thinkingPrompt string
	var thinkingBudget int
	if a.provider.Name() == "claude" {
		thinkingBudget = ThinkingBudgetTokens(settings.ThinkingLevel)
	} else {
		thinkingPrompt = ThinkingPrompt(settings.ThinkingLevel)
	}

	// Auto-approval mode notice
	autoApprovalNotice := ""
	if a.autoApprove {
		autoApprovalNotice = `

## 🚀 AUTO-APPROVAL MODE ENABLED
**You are running in auto-approval mode. DO NOT ask for permission before taking actions.**
- Execute file writes, deletions, and modifications immediately without confirmation
- Run shell commands directly without asking first
- Create, modify, or delete files as requested without hesitation
- The user has explicitly disabled all safety prompts with --yes flag
- Only skip actions if they are IMPOSSIBLE or DANGEROUS (e.g., rm -rf /, destructive operations)
- For normal operations (file writes, reads, modifications), proceed immediately`
	}

	// System prompt with actual paths
	systemPrompt := fmt.Sprintf(`You are lsbot, a helpful AI assistant running on the user's computer.%s

## System Environment
- Operating System: %s
- Architecture: %s
- Home Directory: %s
- Desktop: %s/Desktop
- Documents: %s/Documents
- Downloads: %s/Downloads
- User: %s

## Available Tools

### File Operations
- file_send: Send/transfer a file to the user via messaging platform
- file_list: List directory contents (use ~/Desktop for desktop)
- file_read: Read file contents
- file_write: Write content to a file (creates parent directories if needed)
- file_trash: Move files to trash (for delete operations)
- file_list_old: Find old files not modified for N days

### Calendar (macOS)
- calendar_today: List today's calendar events/meetings (NOT for answering date/time questions)
- calendar_list_events: List upcoming events
- calendar_create_event: Create new event
- calendar_search: Search events
- calendar_delete: Delete event

### Reminders (macOS)
- reminders_list: List pending reminders
- reminders_add: Add new reminder
- reminders_complete: Mark as complete
- reminders_delete: Delete reminder

### Notes (macOS)
- notes_list: List notes
- notes_read: Read note content
- notes_create: Create new note
- notes_search: Search notes

### Weather
- weather_current: Current weather
- weather_forecast: Weather forecast

### Web
- web_search: Search the web (DuckDuckGo)
- web_fetch: Fetch URL content
- open_url: Open URL in browser

### Clipboard
- clipboard_read: Read clipboard
- clipboard_write: Write to clipboard

### System
- system_info: System information
- shell_execute: Execute shell command
- process_list: List processes
- notification_send: Send notification
- screenshot: Capture screen

### Music (macOS)
- music_play/pause/next/previous: Playback control
- music_now_playing: Current track info
- music_volume: Set volume
- music_search: Search and play

### Scheduled Tasks (Cron)
- cron_create: Create ONE scheduled task with 'prompt' parameter. The AI runs a full conversation each trigger (can use web_search, weather, etc.) and sends the result to the user. For raw tool execution, use 'tool'+'arguments' instead.
- cron_list: List all scheduled tasks with their status
- cron_delete: Delete a scheduled task by ID
- cron_pause: Pause a scheduled task
- cron_resume: Resume a paused scheduled task

### Database (SQLite)
- db_exec: Execute SQL (CREATE TABLE, INSERT, UPDATE, DELETE) on a named local SQLite database. Data persists across conversations. Use db="ledger" for finance, db="todos" for tasks, etc.
- db_query: SELECT from a local SQLite database. Returns JSON array of rows.

### Memory & Profile
- profile_update: Save/update your nickname and timezone (call during onboarding or when user changes preferences)
- memory_write: Persist important facts about the user to long-term memory (replaces full MEMORY.md content)
- user_model_write: Update USER.md — your evolving model of the user's personality, communication style, and preferences (separate from MEMORY.md)

### Browser Automation (snapshot-then-act pattern)
- browser_start: Start new browser or connect to existing Chrome via cdp_url (e.g. "127.0.0.1:9222")
- browser_navigate: Navigate to a URL (auto-connects to Chrome on port 9222 if available, otherwise launches new)
- browser_snapshot: Capture accessibility tree with numbered refs
- browser_click: Click an element by ref number
- browser_type: Type text into element by ref number (optional submit with Enter)
- browser_press: Press keyboard key (Enter, Tab, Escape, etc.)
- browser_execute_js: Run JavaScript on the page (dismiss modals, extract data, etc.)
- browser_click_all: Click ALL elements matching a CSS selector with delay (batch like/follow)
- browser_screenshot: Take page screenshot
- browser_tabs: List all open tabs
- browser_tab_open: Open new tab
- browser_tab_close: Close a tab
- browser_status: Check browser state
- browser_stop: Close browser (or disconnect from external Chrome)

## Browser Automation Rules
You MUST follow the **snapshot-then-act** pattern for ALL browser interactions:
1. **Navigate** to the target website using browser_navigate (skip if the browser is already on the right site)
2. **Snapshot** the page using browser_snapshot to discover UI elements and their ref numbers
3. **Interact** with elements step by step using browser_click / browser_type / browser_press
4. **Re-snapshot** after any page change (click, navigation, form submit) to get updated refs

**CRITICAL: If the browser is already open on a website, continue working on that page — do NOT re-navigate.**
- If you previously called browser_navigate to 知乎 and the user now says "搜索XXX", take browser_snapshot on the current 知乎 page, find the search input, and type into it.
- Do NOT call browser_navigate again to the same site just to search.

**CRITICAL: When asked to search on a website that is already open in the browser, ALWAYS use the browser search bar — NEVER use web_search or web_fetch.**
- BAD: User opened 知乎, then says "搜索OpenClaw" → you call web_search
- GOOD: User opened 知乎, then says "搜索OpenClaw" → browser_snapshot → find search input → browser_type ref=N text="OpenClaw" submit=true

**CRITICAL: NEVER construct or guess URLs to skip UI interaction steps.**
- BAD: Directly navigating to https://www.xiaohongshu.com/search/关键词
- GOOD: Navigate to https://www.xiaohongshu.com → snapshot → find search box → type keyword → submit

Use the page's UI elements (search boxes, buttons, menus) to accomplish the task step by step. Refs are invalidated after page changes — always re-snapshot.

**CRITICAL: Multi-step browser tasks MUST continue until fully complete.**
- After browser_click returns a page snapshot, you MUST examine the snapshot and take the NEXT required action (type, click, snapshot, etc.).
- NEVER stop after a single browser action unless the user's ENTIRE request has been fulfilled.
- Seeing a page snapshot in a tool result means: "here is the current state — what should I do next?"
- If you see a login modal or any obstacle, handle it (dismiss, log in, or report to user) — do not silently stop.

**Zhihu (知乎) commenting — EXACT VERIFIED METHOD:**
Zhihu uses a Draft.js editor. Direct DOM manipulation (innerHTML, value=, execCommand insertText) does NOT update Draft.js internal state — the 发布 button will stay DISABLED. You MUST use ClipboardEvent paste.

PREFERRED: call browser_comment_zhihu(comment="...") — handles everything automatically.
To reply to a specific person's comment (nested reply), use browser_comment_zhihu(comment="...", reply_to="username"). The tool will find that user's 回复 button and post a nested reply instead of a top-level comment.

If using mcp_chrome_evaluate_script instead, use EXACTLY this 3-step sequence:

Step 1 — Click the comment trigger (button with 评论 in text):
  function: "() => { var btn = Array.from(document.querySelectorAll('button,span')).find(function(e){ var t = e.textContent.replace(/\u200b/g,'').trim(); return /^[\d\s]*条?评论$/.test(t) || t === '添加评论'; }); if(btn){btn.click();return 'clicked:'+btn.textContent.trim();} return 'not found'; }"

Step 2 — Wait ~1s for editor to appear, then paste via ClipboardEvent (CRITICAL — do NOT use execCommand or innerHTML):
  function: "() => { var ed = document.querySelector('.public-DraftEditor-content'); if(!ed){return 'editor not found';} ed.click(); ed.focus(); document.execCommand('selectAll',false); var dt = new DataTransfer(); dt.setData('text/plain', 'COMMENT_TEXT'); ed.dispatchEvent(new ClipboardEvent('paste',{clipboardData:dt,bubbles:true,cancelable:true})); return 'pasted'; }"

Step 3 — Wait ~600ms then click 发布 by text match (querySelector returns search button first, MUST find by text):
  function: "() => { var btn = Array.from(document.querySelectorAll('button')).find(function(b){ return b.textContent.replace(/\u200b/g,'').trim() === '发布'; }); if(btn&&!btn.disabled){btn.click();return 'submitted';} if(btn&&btn.disabled){return 'button disabled — paste step likely failed';} return 'submit btn not found'; }"

Replace COMMENT_TEXT with actual comment. If 发布 is disabled after paste, retry step 2 — the editor may not have focused properly.
DO NOT click "写回答" — that writes a full answer, not a comment.

**Xiaohongshu (小红书) commenting — EXACT VERIFIED METHOD:**
Xiaohongshu uses a contenteditable <p id="content-textarea">. Direct DOM manipulation (textContent=, innerText=) does NOT update the framework state — the 发送 button stays DISABLED. You MUST use ClipboardEvent paste.

PREFERRED: call browser_comment_xiaohongshu(comment="...") — handles everything automatically.

If using mcp_chrome_evaluate_script instead, use EXACTLY this 2-step sequence:
Step 1 — Paste comment via ClipboardEvent:
  function: "() => { var ed = document.querySelector('#content-textarea'); if(!ed) return 'not found'; ed.focus(); ed.textContent=''; ed.dispatchEvent(new Event('input',{bubbles:true})); var dt = new DataTransfer(); dt.setData('text/plain', 'COMMENT_TEXT'); ed.dispatchEvent(new ClipboardEvent('paste',{clipboardData:dt,bubbles:true,cancelable:true})); return 'pasted'; }"
Step 2 — Click 发送:
  function: "() => { var btn = Array.from(document.querySelectorAll('button')).find(function(b){ return b.textContent.trim() === '发送'; }); if(btn&&!btn.disabled){btn.click();return 'submitted';} if(btn&&btn.disabled){return 'button disabled — paste step likely failed';} return 'submit btn not found'; }"

**Handling modals/overlays:** If an element is blocked by a modal or overlay (error message mentions "element covered by"), use browser_execute_js to dismiss it. Example scripts:
- document.querySelector('.modal-overlay').remove()
- document.querySelector('.dialog-close-btn').click()
Then re-snapshot and continue.

**Batch actions (like/follow/favorite):** When the user asks to like/点赞, follow/关注, or favorite/收藏 "all" content, you MUST use browser_click_all — NEVER try to click individual refs from snapshot. This applies regardless of how the user phrases it (markdown list, comma-separated, or prose). browser_click_all automatically scrolls and keeps clicking until no new elements appear. Use skip_selector to avoid toggling already-active items. For Chinese sites (小红书/抖音/微博), try these selectors DIRECTLY without inspecting first:
- 点赞 (like) → browser_click_all with selector ".like-wrapper", skip_selector ".like-wrapper.active, .like-wrapper.liked"
- 收藏 (favorite) → browser_click_all with selector "[class*='collect']", skip_selector "[class*='collect'].active"
- 关注 (follow) → browser_click_all with selector "[class*='follow']", skip_selector "[class*='follow'].active"
If click count is 0, inspect with: return Array.from(document.querySelectorAll('span,button')).filter(e=>e.children.length<5).slice(0,10).map(e=>e.className+' | '+e.textContent.trim().slice(0,15)).join('\n')
Do NOT waste rounds — try clicking first, inspect only if it fails.

**Iterative browsing (processing multiple pages):** When the user asks to iterate through search results (e.g., "逐个文章添加评论", "comment on all posts"), use browser_visited to track progress:
1. On the search results page, collect all article links/titles
2. For each article: call browser_visited(action="check", url=...) — if "visited", skip it
3. Click the article to open it, perform the action (comment, like, etc.)
4. Call browser_visited(action="mark", url=...) to record it
5. Navigate back to search results and continue with next article
This prevents re-processing articles and survives page reloads within the same session.

**CRITICAL: Xiaohongshu (小红书) anti-bot protection — NEVER navigate to XHS article URLs directly.**
XHS blocks direct URL navigation and returns 404 security pages. You MUST open articles by CLICKING on them from the search results page.
Correct workflow:
1. Navigate to https://www.xiaohongshu.com and search via the search box
2. On the search results page, use browser_execute_js to click on article covers/titles:
   function: "() => { var items = document.querySelectorAll('section.note-item a.cover'); if(items.length > INDEX) { items[INDEX].click(); return 'clicked item INDEX'; } return 'not found'; }"
   Replace INDEX with the 0-based index of the article to open.
3. After the article detail opens (as an overlay/modal), perform actions (comment, like)
4. Close the overlay by pressing Escape or clicking outside, then continue with the next article
5. NEVER use browser_navigate to go to a xiaohongshu.com/explore/... URL — it WILL fail with 404

**Xiaohongshu (小红书) like/点赞:** On a note detail page, click the heart icon. Use browser_execute_js:
  function: "() => { var likeBtn = document.querySelector('.like-wrapper:not(.active), [class*=\"like\"]:not(.active)'); if(likeBtn){likeBtn.click();return 'liked';} return 'already liked or not found'; }"

## Terminal Output Styling
When producing output that will be displayed in a terminal (shell scripts, CLI tool output, log lines), use ANSI escape codes for clarity. Wrap the reset code \033[0m at the end of every colored span.
- Warnings / config issues the user must act on: prefix with \033[1;91m (bold bright-red) and the symbol ⚠
- Errors / failures: prefix with \033[1;91m (bold bright-red) and the word ERROR
- Success / confirmations: prefix with \033[32m (green) and the symbol ✓
- Neutral info / status: wrap with \033[36m (cyan)
- Emphasis only: wrap with \033[1m (bold, no colour)
Do NOT use colour codes in conversational chat replies or in prose explanations — only in terminal-destined text.

## Important Rules
1. **ALWAYS use tools** - Never tell users to do things manually
2. **Be action-oriented** - Execute tasks, don't just describe them
3. **Use correct paths** - 桌面=~/Desktop, 下载=~/Downloads, 文档=~/Documents
4. **Full permission** - You have full permission to execute all tools
5. **Be concise** - Short, helpful responses
6. **NEVER claim success without tool execution** - If user asks to create/add/delete something, you MUST call the corresponding tool. Never say "已创建/已添加/已删除" unless you actually called the tool and it succeeded.
7. **Date format for calendar** - When creating calendar events, use YYYY-MM-DD HH:MM format. Convert relative dates (明天/下周一) to absolute dates based on today's date.
8. **CRITICAL: Database rules** - For ANY persistent structured data (records, finances, ledger, todos, contacts, logs, inventory, etc.):
   - ALWAYS use db_exec / db_query — NEVER use file_write or shell_execute for structured data storage.
   - On first use: call db_exec to CREATE TABLE IF NOT EXISTS, then INSERT.
   - sqlite3 CLI is NOT available. The db_exec/db_query tools provide native SQLite — use them directly.
   - Example flow for "帮我记一笔账":
     1. db_exec(db="ledger", sql="CREATE TABLE IF NOT EXISTS transactions (id INTEGER PRIMARY KEY AUTOINCREMENT, date TEXT, amount REAL, category TEXT, note TEXT)")
     2. db_exec(db="ledger", sql="INSERT INTO transactions (date, amount, category, note) VALUES ('2024-01-15', 38.0, '餐饮', '午饭')")
     3. Reply with confirmation
9. **CRITICAL: Cron job rules** - When user asks for periodic/scheduled tasks:
   - Call cron_create EXACTLY ONCE with the 'prompt' parameter.
   - Example: cron_create(name="motivation", schedule="43 * * * *", prompt="生成一条独特的编程激励鸡汤，鼓励用户写代码创造新产品")
   - NEVER call cron_create multiple times. NEVER use shell_execute or file_write for cron tasks.
9. **Progress updates** — For iterative/multi-step tasks (e.g., commenting on multiple articles, processing a list), output a brief status message after each completed item (e.g., "✅ 已完成第3篇，继续下一篇"). The user will see these updates in real time.

Current date: %s%s%s`, autoApprovalNotice, runtime.GOOS, runtime.GOARCH, homeDir, homeDir, homeDir, homeDir, msg.Username, time.Now().Format("2006-01-02"), thinkingPrompt, formatSkillsSection())

	// Inject user profile and long-term memory
	profile, _ := userprofile.Load()
	memory := userprofile.LoadMemory()

	if profile.IsOnboarded() {
		tz := profile.Timezone
		if tz == "" {
			tz = "UTC"
		}
		systemPrompt += fmt.Sprintf("\n\n## User Profile\n- Nickname: %s\n- Timezone: %s\n  Use this timezone when displaying times, creating calendar events, or scheduling tasks.",
			profile.Nickname, tz)
	}

	if memory != "" {
		systemPrompt += "\n\n## Your Memory About This User\n" + memory
	}

	userModel := userprofile.LoadUserModel()
	if userModel != "" {
		systemPrompt += "\n\n## User Model (personality, style, preferences)\n" + userModel
	}

	if !profile.IsOnboarded() {
		systemPrompt = `## ONBOARDING REQUIRED
This is your first time talking to this user. Before doing ANYTHING else, greet them warmly and ask:
1. What nickname should I call you?
2. What is your timezone? (e.g., Asia/Shanghai, America/New_York, Europe/London)

Once you have both answers, call the profile_update tool to save them. Then proceed normally.

` + systemPrompt
	}

	if a.customInstructions != "" {
		systemPrompt += "\n\n## Custom Instructions\n" + a.customInstructions
	}

	// Call AI provider (with quota-based fallback)
	resp, err := a.chatWithFallback(ctx, ChatRequest{
		Messages:       messages,
		SystemPrompt:   systemPrompt,
		Tools:          tools,
		MaxTokens:      4096,
		ThinkingBudget: thinkingBudget,
	})
	if err != nil {
		return router.Response{}, fmt.Errorf("AI error: %w", err)
	}

	// Handle tool use if needed
	maxToolRounds := a.maxToolRounds
	var pendingFiles []router.FileAttachment
	toolCallCounts := map[string]int{} // track per-tool call counts
	for round := range maxToolRounds {
		if resp.FinishReason != "tool_use" {
			break
		}

		// Process tool calls and track counts; detect stalls
		stallHint := ""
		for _, tc := range resp.ToolCalls {
			toolCallCounts[tc.Name]++
			count := toolCallCounts[tc.Name]
			if count > 1 {
				logger.Warn("[Agent] Tool %s called %d times (round %d/%d, user: %s)", tc.Name, count, round+1, maxToolRounds, msg.Username)
			}
			if count >= 3 && strings.HasPrefix(tc.Name, "browser_") {
				stallHint = fmt.Sprintf(
					"\n\n[SYSTEM HINT] You have called %s %d times in a row. STOP and use the dedicated comment tool instead. "+
						"For Zhihu: call browser_comment_zhihu(comment=\"...\"). "+
						"For Xiaohongshu: call browser_comment_xiaohongshu(comment=\"...\"). "+
						"These handle everything automatically. Do NOT keep clicking buttons or interacting manually.",
					tc.Name, count,
				)
			}
		}

		toolResults, files := a.processToolCalls(ctx, resp.ToolCalls)
		pendingFiles = append(pendingFiles, files...)

		// Log tool results that look like errors
		for _, result := range toolResults {
			if result.IsError || strings.HasPrefix(result.Content, "Error") {
				logger.Warn("[Agent] Tool error (round %d/%d): %s", round+1, maxToolRounds, result.Content)
			}
		}

		// Add assistant response with tool calls
		messages = append(messages, Message{
			Role:             "assistant",
			Content:          resp.Content,
			ReasoningContent: resp.ReasoningContent,
			ToolCalls:        resp.ToolCalls,
		})

		// Add tool results; append stall hint to last result if detected
		for i, result := range toolResults {
			if stallHint != "" && i == len(toolResults)-1 {
				result.Content += stallHint
			}
			messages = append(messages, Message{
				Role:       "user",
				ToolResult: &result,
			})
		}

		// Detect if any browser tool was used this round.
		hasBrowserTool := false
		for _, tc := range resp.ToolCalls {
			if strings.HasPrefix(tc.Name, "browser_") {
				hasBrowserTool = true
				break
			}
		}

		// Continue the conversation.
		// Force tool_choice="required" when the last round used browser tools so DeepSeek
		// cannot return a bare text response — it must call another tool to continue.
		// Use a per-call timeout so a stalled API call doesn't hang the agent forever.
		// Scale timeout with message count: base + 1s per message (capped at base+90s).
		// Base defaults to 90s but can be overridden via CallTimeoutSecs config.
		baseTimeout := 90 * time.Second
		if a.callTimeoutSecs > 0 {
			baseTimeout = time.Duration(a.callTimeoutSecs) * time.Second
		}
		callTimeout := baseTimeout + time.Duration(min(len(messages), 90))*time.Second
		logger.Info("[Agent] Calling AI (round %d/%d, forceToolUse=%v, timeout=%s, user: %s)", round+2, maxToolRounds, hasBrowserTool, callTimeout, msg.Username)
		chatReq := ChatRequest{
			Messages:       messages,
			SystemPrompt:   systemPrompt,
			Tools:          tools,
			MaxTokens:      4096,
			ForceToolUse:   hasBrowserTool,
			ThinkingBudget: thinkingBudget,
		}
		callCtx, callCancel := context.WithTimeout(ctx, callTimeout)
		resp, err = a.chatWithFallback(callCtx, chatReq)
		callCancel()
		// Retry once on timeout — the API may have been temporarily slow.
		if err != nil && ctx.Err() == nil && (strings.Contains(err.Error(), "deadline exceeded") || strings.Contains(err.Error(), "context canceled")) {
			logger.Warn("[Agent] AI call timed out (round %d), retrying once...", round+2)
			callCtx2, callCancel2 := context.WithTimeout(ctx, callTimeout)
			resp, err = a.chatWithFallback(callCtx2, chatReq)
			callCancel2()
		}
		if err != nil {
			logger.Warn("[Agent] AI call failed (round %d, forceToolUse=%v): %v", round+2, hasBrowserTool, err)
			// Log message count and last few messages to help diagnose what triggered the failure
			logger.Warn("[Agent] Request had %d messages; last message role=%s content_len=%d",
				len(messages),
				func() string {
					if len(messages) > 0 {
						return messages[len(messages)-1].Role
					}
					return "none"
				}(),
				func() int {
					if len(messages) > 0 {
						m := messages[len(messages)-1]
						if m.ToolResult != nil {
							return len(m.ToolResult.Content)
						}
						return len(m.Content)
					}
					return 0
				}(),
			)
			return router.Response{}, fmt.Errorf("AI error: %w", err)
		}
		logger.Info("[Agent] AI response (round %d): finish_reason=%s tools=%d content_len=%d", round+2, resp.FinishReason, len(resp.ToolCalls), len(resp.Content))

		// Send intermediate text as a progress update if the AI produced content while continuing tool use.
		if resp.Content != "" && resp.FinishReason == "tool_use" {
			if progress := router.ProgressFromContext(ctx); progress != nil {
				progress(resp.Content)
			}
		}
	}
	if resp.FinishReason == "tool_use" {
		logger.Warn("[Agent] Tool loop hit max rounds (%d), forcing stop (user: %s)", maxToolRounds, msg.Username)
	}

	// Save conversation to in-memory store
	a.memory.AddExchange(convKey,
		Message{Role: "user", Content: msg.Text},
		Message{Role: "assistant", Content: resp.Content},
	)

	// Persist to SQLite for cross-session recall
	if hs, err := history.Global(); err == nil {
		_ = hs.Save(convKey, msg.Platform, msg.Username, []history.Message{
			{Role: "user", Content: msg.Text},
			{Role: "assistant", Content: resp.Content},
		})
	}

	// Log response at verbose level
	logger.Debug("[Agent] Response: %s", resp.Content)

	return router.Response{Text: resp.Content, Files: pendingFiles}, nil
}

// formatSkillsSection returns a formatted string listing eligible skills, or empty if none.
// Default skills (metadata.default=true) have their full content inlined so the AI can
// use them immediately without being told to "check" the skill first.
func formatSkillsSection() string {
	cfg, err := config.Load()
	var disabled, extraDirs []string
	if err == nil {
		disabled = cfg.Skills.Disabled
		extraDirs = cfg.Skills.ExtraDirs
	}
	report := skills.BuildStatusReport(disabled, extraDirs)
	eligible := report.EligibleSkills()
	if len(eligible) == 0 {
		return ""
	}
	var sb strings.Builder

	// Inline full content of default skills first so the AI has them in context.
	for _, s := range eligible {
		if s.Metadata.Default && s.Content != "" {
			fmt.Fprintf(&sb, "\n\n## Skill: %s\n%s", s.Name, s.Content)
		}
	}

	// List remaining (non-default) skills by name + description.
	var nonDefault []skills.SkillStatus
	for _, s := range eligible {
		if !s.Metadata.Default {
			nonDefault = append(nonDefault, s)
		}
	}
	if len(nonDefault) > 0 {
		sb.WriteString("\n\nSkills:\n")
		for _, s := range nonDefault {
			fmt.Fprintf(&sb, "  %s: %s\n", s.Name, s.Description)
		}
	}

	fmt.Fprintf(&sb, "\n安装 Skill: 将 skill 文件夹放入 %s 即可", skills.ShortenHomePath(report.ManagedDir))
	return sb.String()
}

// buildToolsList creates the tools list for the AI provider
func (a *Agent) buildToolsList() []Tool {
	tools := []Tool{
		// === FILE OPERATIONS ===
		{
			Name:        "file_send",
			Description: "Send a file to the user via the messaging platform. Use this when the user asks you to send/transfer/share a file. Use ~ for home directory.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":       map[string]string{"type": "string", "description": "Path to the file (use ~ for home, e.g., ~/Desktop/report.pdf)"},
					"media_type": map[string]string{"type": "string", "description": "Media type: file, image, voice, or video (default: file)"},
				},
				"required": []string{"path"},
			}),
		},
		{
			Name:        "file_read",
			Description: "Read the contents of a file. Use ~ for home directory.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Path to the file (use ~ for home, e.g., ~/Desktop/file.txt)"}},
				"required":   []string{"path"},
			}),
		},
		{
			Name:        "file_write",
			Description: "Write content to a file. Creates parent directories if needed. Use ~ for home directory.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]string{"type": "string", "description": "Path to the file (use ~ for home, e.g., ~/Desktop/file.txt)"},
					"content": map[string]string{"type": "string", "description": "Content to write to the file"},
				},
				"required": []string{"path", "content"},
			}),
		},
		{
			Name:        "file_list",
			Description: "List contents of a directory. Use ~/Desktop for desktop, ~/Downloads for downloads, etc.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]string{"type": "string", "description": "Directory path (use ~ for home, e.g., ~/Desktop)"}},
			}),
		},
		{
			Name:        "file_list_old",
			Description: "List files not modified for specified days. Use ~/Desktop for desktop, etc.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{"type": "string", "description": "Directory path (use ~ for home, e.g., ~/Desktop)"},
					"days": map[string]string{"type": "number", "description": "Minimum days since modification"},
				},
				"required": []string{"path"},
			}),
		},
		{
			Name:        "file_trash",
			Description: "Move files to Trash",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"files": map[string]any{"type": "array", "items": map[string]string{"type": "string"}, "description": "File paths to trash"},
				},
				"required": []string{"files"},
			}),
		},

		// === CALENDAR ===
		{
			Name:        "calendar_today",
			Description: "Get today's calendar events",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "calendar_list_events",
			Description: "List upcoming calendar events",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"days": map[string]string{"type": "number", "description": "Days ahead (default 7)"}},
			}),
		},
		{
			Name:        "calendar_create_event",
			Description: "Create a new calendar event",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":      map[string]string{"type": "string", "description": "Event title"},
					"start_time": map[string]string{"type": "string", "description": "Start time (YYYY-MM-DD HH:MM)"},
					"duration":   map[string]string{"type": "number", "description": "Duration in minutes (default 60)"},
					"calendar":   map[string]string{"type": "string", "description": "Calendar name (optional)"},
					"location":   map[string]string{"type": "string", "description": "Event location (optional)"},
					"notes":      map[string]string{"type": "string", "description": "Event notes (optional)"},
				},
				"required": []string{"title", "start_time"},
			}),
		},
		{
			Name:        "calendar_search",
			Description: "Search calendar events by keyword",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"keyword": map[string]string{"type": "string", "description": "Search keyword"},
					"days":    map[string]string{"type": "number", "description": "Days to search (default 30)"},
				},
				"required": []string{"keyword"},
			}),
		},
		{
			Name:        "calendar_delete",
			Description: "Delete a calendar event by title",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    map[string]string{"type": "string", "description": "Event title to delete"},
					"calendar": map[string]string{"type": "string", "description": "Calendar name (optional)"},
					"date":     map[string]string{"type": "string", "description": "Date (YYYY-MM-DD) to narrow search (optional)"},
				},
				"required": []string{"title"},
			}),
		},

		// === REMINDERS ===
		{
			Name:        "reminders_list",
			Description: "List all pending reminders",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "reminders_add",
			Description: "Create a new reminder",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title": map[string]string{"type": "string", "description": "Reminder title"},
					"list":  map[string]string{"type": "string", "description": "Reminder list name (default: Reminders)"},
					"due":   map[string]string{"type": "string", "description": "Due date (YYYY-MM-DD or YYYY-MM-DD HH:MM)"},
					"notes": map[string]string{"type": "string", "description": "Additional notes"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "reminders_complete",
			Description: "Mark a reminder as complete",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Reminder title"}},
				"required":   []string{"title"},
			}),
		},
		{
			Name:        "reminders_delete",
			Description: "Delete a reminder",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Reminder title"}},
				"required":   []string{"title"},
			}),
		},

		// === NOTES ===
		{
			Name:        "notes_list",
			Description: "List notes in a folder",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"folder": map[string]string{"type": "string", "description": "Folder name (default: Notes)"},
					"limit":  map[string]string{"type": "number", "description": "Max notes to show (default 20)"},
				},
			}),
		},
		{
			Name:        "notes_read",
			Description: "Read a note's content",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"title": map[string]string{"type": "string", "description": "Note title"}},
				"required":   []string{"title"},
			}),
		},
		{
			Name:        "notes_create",
			Description: "Create a new note",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":  map[string]string{"type": "string", "description": "Note title"},
					"body":   map[string]string{"type": "string", "description": "Note content"},
					"folder": map[string]string{"type": "string", "description": "Folder name (default: Notes)"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "notes_search",
			Description: "Search notes by keyword",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"keyword": map[string]string{"type": "string", "description": "Search keyword"}},
				"required":   []string{"keyword"},
			}),
		},

		// === WEATHER ===
		{
			Name:        "weather_current",
			Description: "Get current weather for a location",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]string{"type": "string", "description": "City name or location (e.g., 'London', 'Tokyo')"}},
			}),
		},
		{
			Name:        "weather_forecast",
			Description: "Get weather forecast for a location",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"location": map[string]string{"type": "string", "description": "City name or location"},
					"days":     map[string]string{"type": "number", "description": "Days to forecast (1-3)"},
				},
			}),
		},

		// === WEB ===
		{
			Name:        "web_search",
			Description: "Search the web using DuckDuckGo",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]string{"type": "string", "description": "Search query"}},
				"required":   []string{"query"},
			}),
		},
		{
			Name:        "web_fetch",
			Description: "Fetch content from a URL",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to fetch"}},
				"required":   []string{"url"},
			}),
		},
		{
			Name:        "open_url",
			Description: "Open a URL in the default web browser",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to open"}},
				"required":   []string{"url"},
			}),
		},

		// === CLIPBOARD ===
		{
			Name:        "clipboard_read",
			Description: "Read content from the clipboard",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "clipboard_write",
			Description: "Write content to the clipboard",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"content": map[string]string{"type": "string", "description": "Content to copy"}},
				"required":   []string{"content"},
			}),
		},

		// === NOTIFICATIONS ===
		{
			Name:        "notification_send",
			Description: "Send a system notification",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":    map[string]string{"type": "string", "description": "Notification title"},
					"message":  map[string]string{"type": "string", "description": "Notification message"},
					"subtitle": map[string]string{"type": "string", "description": "Subtitle (macOS only)"},
				},
				"required": []string{"title"},
			}),
		},

		// === SCREENSHOT ===
		{
			Name:        "screenshot",
			Description: "Capture a screenshot",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]string{"type": "string", "description": "Save path (default: Desktop)"},
					"type": map[string]string{"type": "string", "description": "Type: fullscreen, window, or selection"},
				},
			}),
		},

		// === MUSIC ===
		{
			Name:        "music_play",
			Description: "Start or resume music playback",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_pause",
			Description: "Pause music playback",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_next",
			Description: "Skip to the next track",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_previous",
			Description: "Go to the previous track",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_now_playing",
			Description: "Get currently playing track info",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "music_volume",
			Description: "Set music volume (0-100)",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"volume": map[string]string{"type": "number", "description": "Volume level 0-100"}},
				"required":   []string{"volume"},
			}),
		},
		{
			Name:        "music_search",
			Description: "Search and play music in Spotify",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"query": map[string]string{"type": "string", "description": "Search query (song, artist, album)"}},
				"required":   []string{"query"},
			}),
		},

		// === SYSTEM ===
		{
			Name:        "system_info",
			Description: "Get system information (CPU, memory, OS)",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "shell_execute",
			Description: "Execute a shell command",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]string{"type": "string", "description": "Command to execute"},
					"timeout": map[string]string{"type": "number", "description": "Timeout in seconds"},
				},
				"required": []string{"command"},
			}),
		},
		{
			Name:        "process_list",
			Description: "List running processes",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"filter": map[string]string{"type": "string", "description": "Filter by name"}},
			}),
		},

		// === GIT & GITHUB ===
		{
			Name:        "git_status",
			Description: "Show git working tree status",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "git_log",
			Description: "Show recent git commits",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"limit": map[string]string{"type": "number", "description": "Number of commits (default 10)"}},
			}),
		},
		{
			Name:        "git_diff",
			Description: "Show git diff",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"staged": map[string]string{"type": "boolean", "description": "Show staged changes"},
					"file":   map[string]string{"type": "string", "description": "Specific file to diff"},
				},
			}),
		},
		{
			Name:        "git_branch",
			Description: "List git branches",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "github_pr_list",
			Description: "List GitHub pull requests (requires gh CLI)",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]string{"type": "string", "description": "Filter by state: open, closed, all"},
					"limit": map[string]string{"type": "number", "description": "Max results (default 10)"},
				},
			}),
		},
		{
			Name:        "github_pr_view",
			Description: "View a GitHub pull request",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"number": map[string]string{"type": "number", "description": "PR number"}},
				"required":   []string{"number"},
			}),
		},
		{
			Name:        "github_issue_list",
			Description: "List GitHub issues (requires gh CLI)",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]string{"type": "string", "description": "Filter by state: open, closed, all"},
					"limit": map[string]string{"type": "number", "description": "Max results (default 10)"},
				},
			}),
		},
		{
			Name:        "github_issue_view",
			Description: "View a GitHub issue",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"number": map[string]string{"type": "number", "description": "Issue number"}},
				"required":   []string{"number"},
			}),
		},
		{
			Name:        "github_issue_create",
			Description: "Create a GitHub issue",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":  map[string]string{"type": "string", "description": "Issue title"},
					"body":   map[string]string{"type": "string", "description": "Issue body"},
					"labels": map[string]string{"type": "string", "description": "Comma-separated labels"},
				},
				"required": []string{"title"},
			}),
		},
		{
			Name:        "github_repo_view",
			Description: "View current GitHub repository info",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},

		// === BROWSER AUTOMATION ===
		{
			Name:        "browser_start",
			Description: "Start a new browser or connect to an existing Chrome. Use cdp_url to attach to a Chrome launched with --remote-debugging-port (e.g. \"127.0.0.1:9222\"). Without cdp_url, launches a new Chrome instance.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"cdp_url":  map[string]string{"type": "string", "description": "CDP address of existing Chrome (e.g. 127.0.0.1:9222). Chrome must be started with --remote-debugging-port flag."},
					"headless": map[string]string{"type": "boolean", "description": "Launch in headless mode (default: false, ignored when using cdp_url)"},
					"url":      map[string]string{"type": "string", "description": "Initial URL to navigate to"},
				},
			}),
		},
		{
			Name:        "browser_navigate",
			Description: "Navigate to a URL in the browser. Auto-starts browser if not running (connects to Chrome on port 9222 if available, otherwise launches new). If the browser is already on the target site, skip this and go directly to browser_snapshot.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to navigate to"}},
				"required":   []string{"url"},
			}),
		},
		{
			Name:        "browser_snapshot",
			Description: "Capture the page accessibility tree with numbered refs. Use these ref numbers with browser_click/browser_type to interact with elements. MUST re-run after any page change.",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_click",
			Description: "Click an element by its ref number from browser_snapshot",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"ref": map[string]string{"type": "number", "description": "Element ref number from browser_snapshot"}},
				"required":   []string{"ref"},
			}),
		},
		{
			Name:        "browser_type",
			Description: "Type text into an element by its ref number from browser_snapshot. Use submit=true to press Enter after typing.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"ref":    map[string]string{"type": "number", "description": "Element ref number from browser_snapshot"},
					"text":   map[string]string{"type": "string", "description": "Text to type"},
					"submit": map[string]string{"type": "boolean", "description": "Press Enter after typing (default: false)"},
				},
				"required": []string{"ref", "text"},
			}),
		},
		{
			Name:        "browser_press",
			Description: "Press a keyboard key (Enter, Tab, Escape, Backspace, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Space, Delete, Home, End, PageUp, PageDown)",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"key": map[string]string{"type": "string", "description": "Key name to press"}},
				"required":   []string{"key"},
			}),
		},
		{
			Name:        "browser_execute_js",
			Description: "Execute JavaScript on the current page. Use to dismiss modals/overlays blocking interaction, extract data, or interact with elements not reachable via refs.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"script": map[string]string{"type": "string", "description": "JavaScript code to execute in page context"}},
				"required":   []string{"script"},
			}),
		},
		{
			Name:        "browser_click_all",
			Description: "Click ALL elements matching a CSS selector. Automatically scrolls down to load more and keeps clicking until no new elements appear. Use skip_selector to skip already-active elements (e.g. already liked). Common: 点赞→selector '.like-wrapper', skip '.like-wrapper.liked' or '.like-wrapper.active'.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"selector":      map[string]string{"type": "string", "description": "CSS selector for elements to click (e.g. '.like-wrapper')"},
					"skip_selector": map[string]string{"type": "string", "description": "CSS selector to skip already-active elements (e.g. '.like-wrapper.active' to skip already-liked). Matches element itself or its children."},
					"delay_ms":      map[string]string{"type": "number", "description": "Milliseconds to wait between clicks (default: 500)"},
				},
				"required": []string{"selector"},
			}),
		},
		{
			Name:        "browser_comment_zhihu",
			Description: "Post a top-level comment OR a nested reply on Zhihu. Must already be on the Zhihu page. For a top-level comment omit reply_to. To reply to a specific person's comment, set reply_to to their username (e.g. \"Jockery\") — the tool will find their 回复 button and post a nested reply. Handles both Draft.js and plain textarea editors automatically.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"comment":  map[string]string{"type": "string", "description": "The comment text to post"},
					"reply_to": map[string]string{"type": "string", "description": "Username to reply to (nested reply). Omit for a top-level comment."},
				},
				"required": []string{"comment"},
			}),
		},
		{
			Name:        "browser_comment_xiaohongshu",
			Description: "Post a comment on a Xiaohongshu (小红书) note. Must already be on a Xiaohongshu note detail page (with the comment input visible at the bottom). Automatically types the comment into the editor via ClipboardEvent paste and clicks 发送 to submit. Use this instead of browser_click + browser_type for Xiaohongshu commenting.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"comment": map[string]string{"type": "string", "description": "The comment text to post"},
				},
				"required": []string{"comment"},
			}),
		},
		{
			Name:        "browser_visited",
			Description: "Track visited URLs during iterative browser operations (e.g., commenting on all search results). Use 'check' before processing a page to skip already-visited ones, 'mark' after processing, 'list' to see all visited URLs, 'clear' to reset. URLs are normalized (query params stripped) so the same page is recognized regardless of navigation path.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]string{"type": "string", "description": "One of: check, mark, list, clear"},
					"url":    map[string]string{"type": "string", "description": "The URL to check or mark (required for check/mark, ignored for list/clear)"},
				},
				"required": []string{"action"},
			}),
		},
		{
			Name:        "browser_screenshot",
			Description: "Take a screenshot of the current page",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":      map[string]string{"type": "string", "description": "Output file path (default: ~/Desktop/browser_screenshot_<timestamp>.png)"},
					"full_page": map[string]string{"type": "boolean", "description": "Capture full scrollable page (default: false)"},
				},
			}),
		},
		{
			Name:        "browser_tabs",
			Description: "List all open browser tabs with their target IDs and URLs",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_tab_open",
			Description: "Open a new browser tab",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"url": map[string]string{"type": "string", "description": "URL to open (default: about:blank)"}},
			}),
		},
		{
			Name:        "browser_tab_close",
			Description: "Close a browser tab by target ID, or close the active tab if no ID given",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"target_id": map[string]string{"type": "string", "description": "Target ID of the tab to close (from browser_tabs)"}},
			}),
		},
		{
			Name:        "browser_status",
			Description: "Check if the browser is running and get current state",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "browser_stop",
			Description: "Close the browser",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},

		// === SCHEDULED TASKS (CRON) ===
		{
			Name:        "cron_create",
			Description: "Create ONE scheduled task. Use 'prompt' to describe what the AI should do each time (generate text, search web, check weather, etc.). The AI runs a full conversation each trigger, so content is fresh every time. Use 'tool'+'arguments' only for raw MCP tool execution without AI. Schedule uses standard 5-field cron: minute hour day month weekday.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":      map[string]string{"type": "string", "description": "Human-readable task name"},
					"schedule":  map[string]string{"type": "string", "description": "Cron expression (e.g., '43 * * * *' for every hour at :43, '0 9 * * 1-5' for weekdays at 9am)"},
					"prompt":    map[string]string{"type": "string", "description": "What the AI should do each time this job triggers. AI runs a full conversation and sends the result to the user. Example: '生成一条独特的编程激励鸡汤'"},
					"tool":      map[string]string{"type": "string", "description": "MCP tool to execute periodically (for raw tool execution without AI)"},
					"arguments": map[string]string{"type": "object", "description": "Arguments for the tool (when using tool parameter)"},
				},
				"required": []string{"name", "schedule"},
			}),
		},
		{
			Name:        "cron_list",
			Description: "List all scheduled tasks with their status, schedule, and last run time",
			InputSchema: jsonSchema(map[string]any{"type": "object", "properties": map[string]any{}}),
		},
		{
			Name:        "cron_delete",
			Description: "Delete a scheduled task by its ID",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"id": map[string]string{"type": "string", "description": "Task ID to delete"}},
				"required":   []string{"id"},
			}),
		},
		{
			Name:        "cron_pause",
			Description: "Pause a scheduled task (it will stop running until resumed)",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"id": map[string]string{"type": "string", "description": "Task ID to pause"}},
				"required":   []string{"id"},
			}),
		},
		{
			Name:        "cron_resume",
			Description: "Resume a paused scheduled task",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"id": map[string]string{"type": "string", "description": "Task ID to resume"}},
				"required":   []string{"id"},
			}),
		},

		// === SQLITE DATABASE ===
		{
			Name:        "db_exec",
			Description: "Execute a SQL statement (CREATE TABLE, INSERT, UPDATE, DELETE) on a named local SQLite database. The database is persistent across conversations.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sql": map[string]string{"type": "string", "description": "SQL statement to execute"},
					"db":  map[string]string{"type": "string", "description": "Database name (default: \"default\"). Use descriptive names e.g. \"ledger\", \"todos\", \"notes\""},
				},
				"required": []string{"sql"},
			}),
		},
		{
			Name:        "db_query",
			Description: "Execute a SELECT query on a named local SQLite database. Returns JSON array of rows.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"sql": map[string]string{"type": "string", "description": "SELECT statement"},
					"db":  map[string]string{"type": "string", "description": "Database name (default: \"default\")"},
				},
				"required": []string{"sql"},
			}),
		},
	}

	// === MEMORY & PROFILE ===
	tools = append(tools,
		Tool{
			Name:        "profile_update",
			Description: "Update the user's persistent profile (nickname and/or timezone). Call this after collecting onboarding info or when the user asks to change their name/timezone.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nickname": map[string]string{"type": "string", "description": "User's preferred nickname"},
					"timezone": map[string]string{"type": "string", "description": "User's timezone (e.g. Asia/Shanghai, America/New_York)"},
				},
			}),
		},
		Tool{
			Name:        "memory_write",
			Description: "Persist notes about the user to long-term memory (MEMORY.md). Call this whenever you learn something important to remember: preferences, facts, context. Pass the COMPLETE updated memory content (not just a diff) — this replaces the previous memory.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"content": map[string]string{"type": "string", "description": "Full MEMORY.md content to persist"}},
				"required":   []string{"content"},
			}),
		},
		Tool{
			Name:        "delegate_task",
			Description: "Spawn one or more isolated child agents to run tasks in parallel. Each child has no conversation history and its result is returned here. Use for independent sub-tasks that can run concurrently (research, file processing, multi-step analysis). Do NOT use for tasks requiring user interaction.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task":    map[string]string{"type": "string", "description": "A single task for one child agent"},
					"tasks":   map[string]any{"type": "array", "items": map[string]string{"type": "string"}, "description": "Multiple tasks to run in parallel"},
					"context": map[string]string{"type": "string", "description": "Optional shared context injected into every child agent's prompt"},
				},
			}),
		},
		Tool{
			Name:        "session_search",
			Description: "Search past conversations using full-text search. Returns snippets and message context from matching sessions. Use this to recall what was discussed in previous conversations.",
			InputSchema: jsonSchema(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query":        map[string]string{"type": "string", "description": "Full-text search query"},
					"max_sessions": map[string]any{"type": "integer", "description": "Max sessions to return (default 3)"},
				},
				"required": []string{"query"},
			}),
		},
		Tool{
			Name:        "user_model_write",
			Description: "Update USER.md — your evolving model of this user's personality, communication style, preferences, and workflow habits. This is separate from MEMORY.md (general facts). Write the COMPLETE updated content; it replaces the previous file.",
			InputSchema: jsonSchema(map[string]any{
				"type":       "object",
				"properties": map[string]any{"content": map[string]string{"type": "string", "description": "Full USER.md content describing the user model"}},
				"required":   []string{"content"},
			}),
		},
	)

	// Append tools from external MCP servers
	for _, t := range a.mcpManager.AllTools() {
		schema := json.RawMessage(t.InputSchema)
		tools = append(tools, Tool{
			Name:        t.FullName,
			Description: t.Description,
			InputSchema: schema,
		})
	}

	return tools
}

// processToolCalls executes tool calls and returns results plus any file attachments
func (a *Agent) processToolCalls(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, []router.FileAttachment) {
	results := make([]ToolResult, 0, len(toolCalls))
	var files []router.FileAttachment

	for _, tc := range toolCalls {
		if tc.Name == "file_send" {
			content, file := executeFileSend(tc.Input)
			if file != nil {
				files = append(files, *file)
			}
			results = append(results, ToolResult{
				ToolCallID: tc.ID,
				Content:    content,
				IsError:    file == nil,
			})
			continue
		}

		toolTimeout := 90 * time.Second
		if strings.HasPrefix(tc.Name, "browser_") {
			toolTimeout = 2 * time.Minute
		}
		toolCtx, toolCancel := context.WithTimeout(ctx, toolTimeout)
		result := a.executeTool(toolCtx, tc.Name, tc.Input)
		toolCancel()
		results = append(results, ToolResult{
			ToolCallID: tc.ID,
			Content:    result,
			IsError:    strings.HasPrefix(result, "Error"),
		})
	}

	return results, files
}

// executeTool runs a tool and returns the result
func (a *Agent) executeTool(ctx context.Context, name string, input json.RawMessage) string {
	logger.Info("[Agent] Executing tool: %s", name)

	// Parse input arguments
	var args map[string]any
	if err := json.Unmarshal(input, &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err)
	}

	// Handle profile and memory tools
	switch name {
	case "profile_update":
		p, _ := userprofile.Load()
		if n, ok := args["nickname"].(string); ok && n != "" {
			p.Nickname = n
		}
		if tz, ok := args["timezone"].(string); ok && tz != "" {
			p.Timezone = tz
		}
		if p.CreatedAt.IsZero() {
			p.CreatedAt = time.Now()
		}
		if err := p.Save(); err != nil {
			return `{"error": "failed to save profile: ` + err.Error() + `"}`
		}
		return `{"ok": true, "nickname": "` + p.Nickname + `", "timezone": "` + p.Timezone + `"}`
	case "memory_write":
		content, _ := args["content"].(string)
		if err := userprofile.WriteMemory(content); err != nil {
			return `{"error": "` + err.Error() + `"}`
		}
		return `{"ok": true}`

	case "user_model_write":
		content, _ := args["content"].(string)
		if err := userprofile.WriteUserModel(content); err != nil {
			return `{"error": "` + err.Error() + `"}`
		}
		return `{"ok": true}`

	case "session_search":
		query, _ := args["query"].(string)
		maxSessions := 3
		if v, ok := args["max_sessions"].(float64); ok && v > 0 {
			maxSessions = int(v)
		}
		hs, err := history.Global()
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		results, err := hs.Search(query, maxSessions, 20)
		if err != nil {
			return fmt.Sprintf(`{"error": %q}`, err.Error())
		}
		if len(results) == 0 {
			return `{"results": [], "message": "No matching sessions found."}`
		}
		out, _ := json.Marshal(results)
		return string(out)

	case "delegate_task":
		return a.DelegateTask(ctx, args)
	}

	// Handle cron tools that need Agent context
	switch name {
	case "cron_create":
		return a.executeCronCreate(args)
	case "cron_list":
		return a.executeCronList()
	case "cron_delete":
		return a.executeCronDelete(args)
	case "cron_pause":
		return a.executeCronPause(args)
	case "cron_resume":
		return a.executeCronResume(args)

	// SQLite
	case "db_exec":
		return executeSQLiteExec(ctx, args)
	case "db_query":
		return executeSQLiteQuery(ctx, args)
	}

	// Block file tools entirely if disabled
	if a.disableFileTools {
		if _, ok := fileToolPaths[name]; ok {
			return "ACCESS DENIED: file operations are disabled by security policy. Do NOT retry. Inform the user that file access is disabled."
		}
	}

	// Enforce allowed_paths restrictions
	if a.pathChecker.HasRestrictions() {
		if err := a.checkToolPathAccess(name, args); err != nil {
			return err.Error()
		}
	}

	// Call tools directly
	result := a.callToolDirect(ctx, name, args)

	// Log result at verbose level (truncate if too long)
	if len(result) > 500 {
		logger.Debug("[Agent] Tool %s result: %s... (truncated)", name, result[:500])
	} else {
		logger.Debug("[Agent] Tool %s result: %s", name, result)
	}

	return result
}

// fileToolPaths maps tool names to the argument key that contains the path.
var fileToolPaths = map[string]string{
	"file_list":     "path",
	"file_list_old": "path",
	"file_read":     "path",
	"file_write":    "path",
	"file_trash":    "path",
	"file_search":   "path",
	"file_info":     "path",
}

// checkToolPathAccess validates that tool arguments respect allowed_paths.
func (a *Agent) checkToolPathAccess(name string, args map[string]any) error {
	if pathKey, ok := fileToolPaths[name]; ok {
		path := "."
		if p, ok := args[pathKey].(string); ok && p != "" {
			path = p
		}
		return a.pathChecker.CheckPath(path)
	}
	if name == "shell_execute" {
		if wd, ok := args["working_directory"].(string); ok && wd != "" {
			return a.pathChecker.CheckPath(wd)
		}
	}
	return nil
}

// callToolDirect calls a tool directly
func (a *Agent) callToolDirect(ctx context.Context, name string, args map[string]any) string {
	// Dispatch to external MCP servers first
	if mcpclient.IsMCPTool(name) {
		result, err := a.mcpManager.Call(ctx, name, args)
		if err != nil {
			return "Error: " + err.Error()
		}
		// Truncate large results (e.g. accessibility tree snapshots) to avoid
		// overflowing the model's context window.
		const mcpMaxLen = 8000
		if len(result) > mcpMaxLen {
			result = result[:mcpMaxLen] + fmt.Sprintf("\n... (truncated, total %d chars)", len(result))
		}
		return result
	}
	switch name {
	// File operations
	case "file_list":
		path := "."
		if p, ok := args["path"].(string); ok {
			path = p
		}
		return executeFileList(ctx, path)
	case "file_list_old":
		path := "."
		days := 30
		if p, ok := args["path"].(string); ok {
			path = p
		}
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeFileListOld(ctx, path, days)
	case "file_trash":
		return executeFileTrash(ctx, args)
	case "file_read":
		path := ""
		if p, ok := args["path"].(string); ok {
			path = p
		}
		return executeFileRead(ctx, path)
	case "file_write":
		path := ""
		content := ""
		if p, ok := args["path"].(string); ok {
			path = p
		}
		if c, ok := args["content"].(string); ok {
			content = c
		}
		return executeFileWrite(ctx, path, content)

	// Calendar
	case "calendar_today":
		return executeCalendarToday(ctx)
	case "calendar_list_events":
		days := 7
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeCalendarListEvents(ctx, days)
	case "calendar_create_event":
		return executeCalendarCreate(ctx, args)
	case "calendar_search":
		return executeCalendarSearch(ctx, args)
	case "calendar_delete":
		return executeCalendarDelete(ctx, args)

	// Reminders
	case "reminders_list":
		return executeRemindersToday(ctx)
	case "reminders_add":
		return executeRemindersAdd(ctx, args)
	case "reminders_complete":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeRemindersComplete(ctx, title)
	case "reminders_delete":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeRemindersDelete(ctx, title)

	// Notes
	case "notes_list":
		return executeNotesList(ctx, args)
	case "notes_read":
		title := ""
		if t, ok := args["title"].(string); ok {
			title = t
		}
		return executeNotesRead(ctx, title)
	case "notes_create":
		return executeNotesCreate(ctx, args)
	case "notes_search":
		keyword := ""
		if k, ok := args["keyword"].(string); ok {
			keyword = k
		}
		return executeNotesSearch(ctx, keyword)

	// Weather
	case "weather_current":
		location := ""
		if l, ok := args["location"].(string); ok {
			location = l
		}
		return executeWeatherCurrent(ctx, location)
	case "weather_forecast":
		location := ""
		days := 3
		if l, ok := args["location"].(string); ok {
			location = l
		}
		if d, ok := args["days"].(float64); ok {
			days = int(d)
		}
		return executeWeatherForecast(ctx, location, days)

	// Web
	case "web_search":
		query := ""
		if q, ok := args["query"].(string); ok {
			query = q
		}
		return executeWebSearch(ctx, query)
	case "web_fetch":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeWebFetch(ctx, url)
	case "open_url":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeOpenURL(ctx, url)

	// Clipboard
	case "clipboard_read":
		return executeClipboardRead(ctx)
	case "clipboard_write":
		content := ""
		if c, ok := args["content"].(string); ok {
			content = c
		}
		return executeClipboardWrite(ctx, content)

	// Notification
	case "notification_send":
		return executeNotificationSend(ctx, args)

	// Screenshot
	case "screenshot":
		return executeScreenshot(ctx, args)

	// Music
	case "music_play":
		return executeMusicPlay(ctx)
	case "music_pause":
		return executeMusicPause(ctx)
	case "music_next":
		return executeMusicNext(ctx)
	case "music_previous":
		return executeMusicPrevious(ctx)
	case "music_now_playing":
		return executeMusicNowPlaying(ctx)
	case "music_volume":
		volume := 50.0
		if v, ok := args["volume"].(float64); ok {
			volume = v
		}
		return executeMusicVolume(ctx, volume)
	case "music_search":
		query := ""
		if q, ok := args["query"].(string); ok {
			query = q
		}
		return executeMusicSearch(ctx, query)

	// System
	case "system_info":
		return executeSystemInfo(ctx)
	case "process_list":
		return executeProcessList(ctx, args)
	case "shell_execute":
		cmd := ""
		if c, ok := args["command"].(string); ok {
			cmd = c
		}
		// Intercept sqlite3 CLI calls — not available on iOS/mobile.
		// Redirect the model to use the built-in db_exec / db_query tools.
		if strings.Contains(cmd, "sqlite3") {
			return "ERROR: sqlite3 CLI is not available. Use the db_exec tool for INSERT/CREATE/UPDATE/DELETE and db_query tool for SELECT instead. These are native SQLite tools that work on all platforms including iOS."
		}
		return executeShell(ctx, cmd)

	// Git & GitHub
	case "git_status":
		return executeGitStatus(ctx)
	case "git_log":
		return executeGitLog(ctx, args)
	case "git_diff":
		return executeGitDiff(ctx, args)
	case "git_branch":
		return executeGitBranch(ctx)
	case "github_pr_list":
		return executeGitHubPRList(ctx, args)
	case "github_pr_view":
		return executeGitHubPRView(ctx, args)
	case "github_issue_list":
		return executeGitHubIssueList(ctx, args)
	case "github_issue_view":
		return executeGitHubIssueView(ctx, args)
	case "github_issue_create":
		return executeGitHubIssueCreate(ctx, args)
	case "github_repo_view":
		return executeGitHubRepoView(ctx)

	// Browser automation
	case "browser_start":
		return executeBrowserStart(ctx, args)
	case "browser_navigate":
		url := ""
		if u, ok := args["url"].(string); ok {
			url = u
		}
		return executeBrowserNavigate(ctx, url)
	case "browser_snapshot":
		return executeBrowserSnapshot(ctx)
	case "browser_click":
		ref := 0
		if r, ok := args["ref"].(float64); ok {
			ref = int(r)
		}
		return executeBrowserClick(ctx, ref)
	case "browser_type":
		ref := 0
		text := ""
		submit := false
		if r, ok := args["ref"].(float64); ok {
			ref = int(r)
		}
		if t, ok := args["text"].(string); ok {
			text = t
		}
		if s, ok := args["submit"].(bool); ok {
			submit = s
		}
		return executeBrowserType(ctx, ref, text, submit)
	case "browser_press":
		key := ""
		if k, ok := args["key"].(string); ok {
			key = k
		}
		return executeBrowserPress(ctx, key)
	case "browser_execute_js":
		script := ""
		if s, ok := args["script"].(string); ok {
			script = s
		}
		return executeBrowserExecuteJS(ctx, script)
	case "browser_comment_zhihu":
		comment, _ := args["comment"].(string)
		replyTo, _ := args["reply_to"].(string)
		return executeBrowserCommentZhihu(ctx, comment, replyTo)
	case "browser_comment_xiaohongshu":
		comment, _ := args["comment"].(string)
		return executeBrowserCommentXiaohongshu(ctx, comment)
	case "browser_visited":
		return executeBrowserVisited(ctx, args)
	case "browser_click_all":
		return executeBrowserClickAll(ctx, args)
	case "browser_screenshot":
		return executeBrowserScreenshot(ctx, args)
	case "browser_tabs":
		return executeBrowserTabs(ctx)
	case "browser_tab_open":
		return executeBrowserTabOpen(ctx, args)
	case "browser_tab_close":
		return executeBrowserTabClose(ctx, args)
	case "browser_status":
		return executeBrowserStatus(ctx)
	case "browser_stop":
		return executeBrowserStop(ctx)

	default:
		return fmt.Sprintf("Tool '%s' not implemented", name)
	}
}

func jsonSchema(schema map[string]any) json.RawMessage {
	data, _ := json.Marshal(schema)
	return data
}
