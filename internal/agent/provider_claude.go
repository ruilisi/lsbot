package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/liushuangls/go-anthropic/v2"
	"github.com/ruilisi/lsbot/internal/logger"
)

// debugTransport logs outgoing HTTP requests and responses at debug level.
// It redacts the Authorization header value, showing only the first/last 4 chars.
type debugTransport struct {
	name string
	base http.RoundTripper
}

func (d *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if logger.IsDebug() {
		auth := req.Header.Get("Authorization")
		authSummary := "(missing)"
		if auth != "" {
			if idx := len(auth); idx > 12 {
				authSummary = auth[:8] + "..." + auth[idx-4:]
			} else {
				authSummary = "(present)"
			}
		}
		name := d.name
		if name == "" {
			name = "http"
		}
		logger.Debug("[%s] → %s %s  Authorization: %s", name, req.Method, req.URL, authSummary)

		// Also write full dump for claude oauth debug
		if name == "claude-oauth" {
			dump, _ := httputil.DumpRequestOut(req, true)
			_ = os.WriteFile("/tmp/claude-request-dump.txt", dump, 0644)
			log.Printf("[Claude OAuth DEBUG] Full request written to /tmp/claude-request-dump.txt")
		}
	}
	resp, err := d.base.RoundTrip(req)
	if logger.IsDebug() {
		name := d.name
		if name == "" {
			name = "http"
		}
		if err != nil {
			logger.Debug("[%s] ← error: %v", name, err)
		} else {
			logger.Debug("[%s] ← %s", name, resp.Status)
		}
	}
	return resp, err
}

const (
	anthropicSetupTokenPrefix    = "sk-ant-oat01-"
	anthropicSetupTokenMinLength = 80
	claudeCodeFallbackVersion    = "2.1.37"
	claudeCodeSystemPrefix       = "You are Claude Code, Anthropic's official CLI for Claude."

	streamMaxRetries    = 3
	streamRetryBaseWait = 1 * time.Second
)

var (
	detectedClaudeVersion string
	detectVersionOnce     sync.Once
)

// getClaudeVersion returns the installed Claude CLI version, detected once and cached.
func getClaudeVersion() string {
	detectVersionOnce.Do(func() {
		out, err := exec.Command("claude", "--version").Output()
		if err != nil {
			detectedClaudeVersion = claudeCodeFallbackVersion
			return
		}
		// Output format: "2.1.37 (Claude Code)" — extract the version number.
		version := strings.TrimSpace(strings.SplitN(string(out), " ", 2)[0])
		if version == "" {
			detectedClaudeVersion = claudeCodeFallbackVersion
			return
		}
		detectedClaudeVersion = version
		logger.Info("[Claude] Detected Claude CLI version: %s", version)
	})
	return detectedClaudeVersion
}

func isOAuthToken(key string) bool {
	return strings.HasPrefix(key, anthropicSetupTokenPrefix) && len(key) >= anthropicSetupTokenMinLength
}

// isTransientError returns true for errors that are worth retrying (connection drops, EOF).
func isTransientError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "unexpected EOF") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "connection refused") ||
		(msg == "EOF")
}

// oauthAdapter mimics Claude Code's headers so OAuth setup tokens are accepted.
type oauthAdapter struct {
	anthropic.DefaultAdapter
	token string
}

func (a *oauthAdapter) SetRequestHeaders(_ *anthropic.Client, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("Anthropic-Beta", "claude-code-20250219,oauth-2025-04-20")
	req.Header.Set("User-Agent", "claude-cli/"+getClaudeVersion()+" (external, cli)")
	req.Header.Set("X-App", "cli")
	req.Header.Set("Anthropic-Dangerous-Direct-Browser-Access", "true")
	// Remove X-Api-Key if the default adapter set it before us
	req.Header.Del("X-Api-Key")
	return nil
}

// ClaudeProvider implements the Provider interface for Claude/Anthropic
type ClaudeProvider struct {
	client   *anthropic.Client
	model    string
	isOAuth  bool
}

// ClaudeConfig holds Claude provider configuration
type ClaudeConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(cfg ClaudeConfig) (*ClaudeProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.Model == "" {
		cfg.Model = "claude-sonnet-4-20250514"
	}

	oauth := isOAuthToken(cfg.APIKey)

	opts := []anthropic.ClientOption{}
	if cfg.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(cfg.BaseURL))
	}
	if oauth {
		adapter := &oauthAdapter{token: cfg.APIKey}
		opts = append(opts, func(c *anthropic.ClientConfig) {
			c.Adapter = adapter
		})
		// Force HTTP/1.1 to avoid HTTP/2 issues with local proxies that can
		// turn API error responses into "unexpected EOF".
		transport := &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
			TLSClientConfig:      &tls.Config{},
			ForceAttemptHTTP2:     false,
			TLSHandshakeTimeout:  10 * time.Second,
			ResponseHeaderTimeout: 120 * time.Second,
		}
		// Disable HTTP/2 by setting TLSNextProto to empty map.
		transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
		opts = append(opts, anthropic.WithHTTPClient(&http.Client{
			Transport: &debugTransport{name: "claude-oauth", base: transport},
		}))
	}

	client := anthropic.NewClient(cfg.APIKey, opts...)

	return &ClaudeProvider{
		client:  client,
		model:   cfg.Model,
		isOAuth: oauth,
	}, nil
}

// Name returns the provider name
func (p *ClaudeProvider) Name() string {
	return "claude"
}

// Chat sends messages and returns a response
func (p *ClaudeProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	// Convert messages to Anthropic format
	messages := make([]anthropic.Message, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, p.toAnthropicMessage(msg))
	}

	// Convert tools to Anthropic format
	tools := make([]anthropic.ToolDefinition, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, anthropic.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Enable extended thinking if requested
	if req.ThinkingBudget > 0 {
		// When thinking is enabled, MaxTokens must be > BudgetTokens
		minMax := req.ThinkingBudget + 4096
		if maxTokens < minMax {
			maxTokens = minMax
		}
	}

	// Build request
	apiReq := anthropic.MessagesRequest{
		Model:     anthropic.Model(p.model),
		MaxTokens: maxTokens,
		Messages:  messages,
		Tools:     tools,
	}

	// Set extended thinking
	if req.ThinkingBudget > 0 {
		apiReq.Thinking = &anthropic.Thinking{
			Type:         anthropic.ThinkingTypeEnabled,
			BudgetTokens: req.ThinkingBudget,
		}
		logger.Info("[Claude] Extended thinking enabled (budget: %d tokens, max: %d)", req.ThinkingBudget, maxTokens)
	}

	// For OAuth tokens, send system prompt as array with Claude Code identity as first block
	if p.isOAuth {
		parts := []anthropic.MessageSystemPart{
			anthropic.NewSystemMessagePart(claudeCodeSystemPrefix),
		}
		if req.SystemPrompt != "" {
			parts = append(parts, anthropic.NewSystemMessagePart(req.SystemPrompt))
		}
		apiReq.MultiSystem = parts
	} else {
		apiReq.System = req.SystemPrompt
	}

	// Call Anthropic API — OAuth tokens require streaming (Claude Code always streams)
	if p.isOAuth {
		var resp anthropic.MessagesResponse
		var lastErr error
		for attempt := range streamMaxRetries {
			resp, lastErr = p.client.CreateMessagesStream(ctx, anthropic.MessagesStreamRequest{
				MessagesRequest: apiReq,
			})
			if lastErr == nil {
				return p.fromAnthropicResponse(resp), nil
			}
			if !isTransientError(lastErr) {
				break
			}
			logger.Warn("[Claude] Transient streaming error (attempt %d/%d): %v", attempt+1, streamMaxRetries, lastErr)
			select {
			case <-ctx.Done():
				return ChatResponse{}, fmt.Errorf("anthropic API error: %w", ctx.Err())
			case <-time.After(streamRetryBaseWait << attempt):
			}
		}
		return ChatResponse{}, fmt.Errorf("anthropic API error: %w", lastErr)
	}

	resp, err := p.client.CreateMessages(ctx, apiReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("anthropic API error: %w", err)
	}
	return p.fromAnthropicResponse(resp), nil
}

// toAnthropicMessage converts a generic Message to Anthropic format
func (p *ClaudeProvider) toAnthropicMessage(msg Message) anthropic.Message {
	switch msg.Role {
	case "user":
		if msg.ToolResult != nil {
			// Tool result message
			return anthropic.Message{
				Role: anthropic.RoleUser,
				Content: []anthropic.MessageContent{
					anthropic.NewToolResultMessageContent(
						msg.ToolResult.ToolCallID,
						msg.ToolResult.Content,
						msg.ToolResult.IsError,
					),
				},
			}
		}
		return anthropic.Message{
			Role: anthropic.RoleUser,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		}

	case "assistant":
		if len(msg.ToolCalls) > 0 {
			// Assistant message with tool calls
			content := make([]anthropic.MessageContent, 0)
			if msg.Content != "" {
				content = append(content, anthropic.NewTextMessageContent(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				input := tc.Input
				if len(input) == 0 {
					input = json.RawMessage(`{}`)
				}
				content = append(content, anthropic.NewToolUseMessageContent(tc.ID, tc.Name, input))
			}
			return anthropic.Message{
				Role:    anthropic.RoleAssistant,
				Content: content,
			}
		}
		return anthropic.Message{
			Role: anthropic.RoleAssistant,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		}

	default:
		return anthropic.Message{
			Role: anthropic.RoleUser,
			Content: []anthropic.MessageContent{
				anthropic.NewTextMessageContent(msg.Content),
			},
		}
	}
}

// fromAnthropicResponse converts Anthropic response to generic format
func (p *ClaudeProvider) fromAnthropicResponse(resp anthropic.MessagesResponse) ChatResponse {
	var content string
	var toolCalls []ToolCall

	for _, c := range resp.Content {
		switch c.Type {
		case anthropic.MessagesContentTypeText:
			if c.Text != nil {
				content += *c.Text
			}
		case anthropic.MessagesContentTypeToolUse:
			if c.MessageContentToolUse != nil {
				toolCalls = append(toolCalls, ToolCall{
					ID:    c.MessageContentToolUse.ID,
					Name:  c.MessageContentToolUse.Name,
					Input: c.MessageContentToolUse.Input,
				})
			}
		}
	}

	finishReason := "stop"
	if resp.StopReason == anthropic.MessagesStopReasonToolUse {
		finishReason = "tool_use"
	}

	return ChatResponse{
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}
}
