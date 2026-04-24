package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

// OpenAICompatProvider implements the Provider interface for any OpenAI-compatible API.
// This covers: MiniMax, Doubao, Zhipu/GLM, OpenAI/GPT, Gemini, Yi, StepFun, SiliconFlow, Grok, etc.
type OpenAICompatProvider struct {
	client       *openai.Client
	model        string
	providerName string
}

// OpenAICompatConfig holds configuration for an OpenAI-compatible provider
type OpenAICompatConfig struct {
	ProviderName string // Display name (e.g., "minimax", "openai")
	APIKey       string
	BaseURL      string
	Model        string
	DefaultURL   string // Default base URL if not specified
	DefaultModel string // Default model if not specified
}

// NewOpenAICompatProvider creates a new OpenAI-compatible provider
func NewOpenAICompatProvider(cfg OpenAICompatConfig) (*OpenAICompatProvider, error) {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = "ollama" // dummy token for local providers that don't require auth
	}

	if cfg.Model == "" {
		cfg.Model = cfg.DefaultModel
	}

	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = cfg.DefaultURL
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = baseURL
	log.Printf("[%s] base URL: %s", cfg.ProviderName, baseURL)

	if cfg.ProviderName == "gemini" {
		config.HTTPClient = &http.Client{
			Transport: &geminiRoundTripper{base: http.DefaultTransport},
		}
	}

	return &OpenAICompatProvider{
		client:       openai.NewClientWithConfig(config),
		model:        cfg.Model,
		providerName: cfg.ProviderName,
	}, nil
}

// Name returns the provider name
func (p *OpenAICompatProvider) Name() string {
	return p.providerName
}

// Chat sends messages and returns a response
func (p *OpenAICompatProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		if msg.Role == "assistant" && msg.Content == "" && len(msg.ToolCalls) == 0 {
			continue
		}
		messages = append(messages, p.toOpenAIMessage(msg))
	}

	tools := make([]openai.Tool, 0, len(req.Tools))
	for _, tool := range req.Tools {
		var params map[string]any
		if err := json.Unmarshal(tool.InputSchema, &params); err != nil {
			params = map[string]any{"type": "object"}
		}
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  params,
			},
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	chatReq := openai.ChatCompletionRequest{
		Model:     p.model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}
	if len(tools) > 0 {
		chatReq.Tools = tools
	}

	resp, err := p.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("%s API error: %w", p.providerName, err)
	}

	return p.fromOpenAIResponse(resp), nil
}

func (p *OpenAICompatProvider) toOpenAIMessage(msg Message) openai.ChatCompletionMessage {
	switch msg.Role {
	case "user":
		if msg.ToolResult != nil {
			content := msg.ToolResult.Content
			if content == "" {
				content = "(empty)"
			}
			return openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    content,
				ToolCallID: msg.ToolResult.ToolCallID,
			}
		}
		return openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}

	case "assistant":
		m := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: msg.Content,
		}
		if len(msg.ToolCalls) > 0 {
			m.ToolCalls = make([]openai.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				m.ToolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: string(tc.Input),
					},
				}
			}
		}
		return m

	case "tool":
		content := msg.Content
		if content == "" && msg.ToolResult != nil {
			content = msg.ToolResult.Content
		}
		if content == "" {
			content = "(empty)"
		}
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    content,
			ToolCallID: msg.ToolResult.ToolCallID,
		}

	default:
		return openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msg.Content,
		}
	}
}

func (p *OpenAICompatProvider) fromOpenAIResponse(resp openai.ChatCompletionResponse) ChatResponse {
	if len(resp.Choices) == 0 {
		return ChatResponse{}
	}

	choice := resp.Choices[0]
	var toolCalls []ToolCall

	for _, tc := range choice.Message.ToolCalls {
		toolCalls = append(toolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	finishReason := "stop"
	if choice.FinishReason == openai.FinishReasonToolCalls {
		finishReason = "tool_use"
	}

	return ChatResponse{
		Content:      choice.Message.Content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}
}

// geminiRoundTripper wraps an http.RoundTripper to fix Gemini API incompatibilities:
// 1. Injects "thinking": {"thinking_budget": 0} into requests to avoid thought_signature errors
// 2. Unwraps array error responses [{error:...}] into {error:...} for go-openai compatibility
type geminiRoundTripper struct {
	base http.RoundTripper
}

func (g *geminiRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Inject thinking_budget: 0 into request body
	if req.Body != nil && req.Method == http.MethodPost {
		body, err := io.ReadAll(req.Body)
		req.Body.Close()
		if err == nil {
			var payload map[string]any
			if json.Unmarshal(body, &payload) == nil {
				payload["thinking"] = map[string]any{"thinking_budget": 0}
				if modified, err := json.Marshal(payload); err == nil {
					body = modified
				}
			}
			req.Body = io.NopCloser(bytes.NewReader(body))
			req.ContentLength = int64(len(body))
		}
	}

	resp, err := g.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Fix array error responses: [{error:...}] -> {error:...}
	if resp.StatusCode >= 400 {
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err == nil {
			body = bytes.TrimSpace(body)
			if len(body) > 0 && body[0] == '[' {
				var arr []json.RawMessage
				if json.Unmarshal(body, &arr) == nil && len(arr) > 0 {
					body = arr[0]
				}
			}
			resp.Body = io.NopCloser(bytes.NewReader(body))
			resp.ContentLength = int64(len(body))
		}
	}

	return resp, nil
}
