package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ruilisi/lsbot/internal/logger"
)

const geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

// GeminiProvider implements the Provider interface for Gemini's native generateContent API.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	model   string
}

// GeminiConfig holds Gemini provider configuration.
type GeminiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type geminiGenerateContentRequest struct {
	Contents          []geminiContent         `json:"contents"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	ToolConfig        *geminiToolConfig       `json:"toolConfig,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
	SystemInstruction *geminiContent          `json:"systemInstruction,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates     []geminiCandidate `json:"candidates"`
	PromptFeedback *struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content       geminiContent `json:"content"`
	FinishReason  string        `json:"finishReason"`
	FinishMessage string        `json:"finishMessage,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type geminiToolConfig struct {
	FunctionCallingConfig *geminiFunctionCallingConfig `json:"functionCallingConfig,omitempty"`
}

type geminiFunctionCallingConfig struct {
	Mode string `json:"mode,omitempty"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens,omitempty"`
}

// NewGeminiProvider creates a new native Gemini provider.
func NewGeminiProvider(cfg GeminiConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.0-flash"
	}

	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}
	baseURL = normalizeGeminiBaseURL(baseURL)

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 30 * time.Second}).DialContext,
		TLSClientConfig:       &tls.Config{},
		ForceAttemptHTTP2:     true,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 120 * time.Second,
	}
	client := &http.Client{Transport: transport}
	if logger.IsDebug() {
		client.Transport = &debugTransport{name: "gemini", base: transport}
	}

	return &GeminiProvider{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client:  client,
		model:   cfg.Model,
	}, nil
}

// Name returns the provider name.
func (p *GeminiProvider) Name() string {
	return "gemini"
}

// Chat sends messages and returns a response.
func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	apiReq := geminiGenerateContentRequest{
		Contents:         p.toGeminiContents(req.Messages),
		GenerationConfig: &geminiGenerationConfig{MaxOutputTokens: maxTokens},
	}
	if req.SystemPrompt != "" {
		apiReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}
	if len(req.Tools) > 0 {
		apiReq.Tools = []geminiTool{{
			FunctionDeclarations: p.toGeminiTools(req.Tools),
		}}
		if req.ForceToolUse {
			apiReq.ToolConfig = &geminiToolConfig{
				FunctionCallingConfig: &geminiFunctionCallingConfig{Mode: "ANY"},
			}
		}
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini request marshal error: %w", err)
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", p.baseURL, p.model)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini request build error: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", strings.TrimSpace(p.apiKey))
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.apiKey))

	resp, err := p.client.Do(httpReq)
	if err != nil && ctx.Err() == nil && isTransientError(err) {
		logger.Warn("[gemini] transient request error, retrying once: %v", err)
		retryReq, buildErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if buildErr != nil {
			return ChatResponse{}, fmt.Errorf("gemini request build error: %w", buildErr)
		}
		retryReq.Header = httpReq.Header.Clone()
		resp, err = p.client.Do(retryReq)
	}
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini API error: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("gemini response read error: %w", err)
	}

	var apiResp geminiGenerateContentResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		if resp.StatusCode >= 400 {
			return ChatResponse{}, fmt.Errorf("gemini API error: status %s", resp.Status)
		}
		return ChatResponse{}, fmt.Errorf("gemini response decode error: %w", err)
	}

	if resp.StatusCode >= 400 {
		return ChatResponse{}, fmt.Errorf("gemini API error: %s", p.errorMessage(apiResp, resp.Status))
	}
	if apiResp.Error != nil {
		return ChatResponse{}, fmt.Errorf("gemini API error: %s", p.errorMessage(apiResp, "request failed"))
	}
	if apiResp.PromptFeedback != nil && apiResp.PromptFeedback.BlockReason != "" && len(apiResp.Candidates) == 0 {
		return ChatResponse{}, fmt.Errorf("gemini prompt blocked: %s", apiResp.PromptFeedback.BlockReason)
	}
	if len(apiResp.Candidates) == 0 {
		return ChatResponse{}, nil
	}

	return p.fromGeminiResponse(apiResp.Candidates[0]), nil
}

func (p *GeminiProvider) toGeminiContents(messages []Message) []geminiContent {
	contents := make([]geminiContent, 0, len(messages))
	toolNamesByID := map[string]string{}
	for _, msg := range messages {
		if msg.Role == "assistant" {
			for _, tc := range msg.ToolCalls {
				if tc.ID != "" && tc.Name != "" {
					toolNamesByID[tc.ID] = tc.Name
				}
			}
		}
		if content, ok := p.toGeminiContent(msg, toolNamesByID); ok {
			contents = append(contents, content)
		}
	}
	return contents
}

func (p *GeminiProvider) toGeminiContent(msg Message, toolNamesByID map[string]string) (geminiContent, bool) {
	switch msg.Role {
	case "assistant":
		parts := make([]geminiPart, 0, len(msg.ToolCalls)+1)
		if msg.Content != "" {
			parts = append(parts, geminiPart{Text: msg.Content})
		}
		for _, tc := range msg.ToolCalls {
			parts = append(parts, geminiPart{
				FunctionCall: &geminiFunctionCall{
					ID:   tc.ID,
					Name: tc.Name,
					Args: decodeToolInput(tc.Input),
				},
			})
		}
		if len(parts) == 0 {
			return geminiContent{}, false
		}
		return geminiContent{Role: "model", Parts: parts}, true
	case "tool":
		if msg.ToolResult == nil {
			return geminiContent{}, false
		}
		return geminiContent{
			Role: "user",
			Parts: []geminiPart{{
				FunctionResponse: buildGeminiFunctionResponse(*msg.ToolResult, toolNamesByID[msg.ToolResult.ToolCallID]),
			}},
		}, true
	case "user":
		if msg.ToolResult != nil {
			return geminiContent{
				Role: "user",
				Parts: []geminiPart{{
					FunctionResponse: buildGeminiFunctionResponse(*msg.ToolResult, toolNamesByID[msg.ToolResult.ToolCallID]),
				}},
			}, true
		}
		if msg.Content == "" {
			return geminiContent{}, false
		}
		return geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: msg.Content}},
		}, true
	default:
		if msg.Content == "" {
			return geminiContent{}, false
		}
		return geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: msg.Content}},
		}, true
	}
}

func (p *GeminiProvider) toGeminiTools(tools []Tool) []geminiFunctionDeclaration {
	out := make([]geminiFunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		var params map[string]any
		if err := json.Unmarshal(tool.InputSchema, &params); err != nil {
			params = map[string]any{"type": "object"}
		}
		out = append(out, geminiFunctionDeclaration{
			Name:        tool.Name,
			Description: tool.Description,
			Parameters:  params,
		})
	}
	return out
}

func (p *GeminiProvider) fromGeminiResponse(candidate geminiCandidate) ChatResponse {
	var textParts []string
	toolCalls := make([]ToolCall, 0)

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			rawArgs, _ := json.Marshal(part.FunctionCall.Args)
			id := part.FunctionCall.ID
			if id == "" {
				id = "gemini_" + uuid.NewString()
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:    id,
				Name:  part.FunctionCall.Name,
				Input: rawArgs,
			})
		}
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_use"
	}

	return ChatResponse{
		Content:      strings.Join(textParts, "\n"),
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
	}
}

func (p *GeminiProvider) errorMessage(resp geminiGenerateContentResponse, fallback string) string {
	if resp.Error == nil {
		return fallback
	}
	switch {
	case resp.Error.Message != "":
		return resp.Error.Message
	case resp.Error.Status != "":
		return resp.Error.Status
	default:
		return fallback
	}
}

func normalizeGeminiBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	switch {
	case strings.HasSuffix(baseURL, "/v1beta/openai"):
		return strings.TrimSuffix(baseURL, "/openai")
	case strings.HasSuffix(baseURL, "/v1beta"):
		return baseURL
	default:
		return baseURL + "/v1beta"
	}
}

func buildGeminiFunctionResponse(result ToolResult, toolName string) *geminiFunctionResponse {
	id := result.ToolCallID
	if id == "" {
		id = "gemini_" + uuid.NewString()
	}
	if toolName == "" {
		toolName = "tool_result"
	}
	response := map[string]any{
		"isError": result.IsError,
		"result":  decodeToolResultContent(result.Content),
	}
	return &geminiFunctionResponse{
		ID:       id,
		Name:     toolName,
		Response: response,
	}
}

func decodeToolInput(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err == nil && args != nil {
		return args
	}
	return map[string]any{"input": string(raw)}
}

func decodeToolResultContent(content string) any {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	var decoded any
	if json.Unmarshal([]byte(content), &decoded) == nil {
		return decoded
	}
	return content
}
