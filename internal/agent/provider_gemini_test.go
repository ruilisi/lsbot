package agent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGeminiProvider_UsesNativeGenerateContentEndpoint(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("x-goog-api-key")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {"role":"model","parts":[{"text":"hello from gemini"}]},
				"finishReason":"STOP"
			}]
		}`))
	}))
	defer srv.Close()

	p, err := NewGeminiProvider(GeminiConfig{
		APIKey:  "gem-test-key",
		BaseURL: srv.URL,
		Model:   "gemini-test-model",
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider() error = %v", err)
	}

	resp, err := p.Chat(context.Background(), ChatRequest{
		SystemPrompt: "be brief",
		Messages:     []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotPath != "/v1beta/models/gemini-test-model:generateContent" {
		t.Fatalf("request path = %q, want native generateContent endpoint", gotPath)
	}
	if gotAPIKey != "gem-test-key" {
		t.Fatalf("x-goog-api-key = %q, want gem-test-key", gotAPIKey)
	}
	if gotAuth != "Bearer gem-test-key" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer gem-test-key")
	}
	if resp.Content != "hello from gemini" {
		t.Fatalf("resp.Content = %q, want %q", resp.Content, "hello from gemini")
	}
}

func TestNormalizeGeminiBaseURL(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://token.lsbot.org", "https://token.lsbot.org/v1beta"},
		{"https://token.lsbot.org/", "https://token.lsbot.org/v1beta"},
		{"https://token.lsbot.org/v1beta", "https://token.lsbot.org/v1beta"},
		{"https://token.lsbot.org/v1beta/openai", "https://token.lsbot.org/v1beta"},
	}

	for _, tt := range tests {
		if got := normalizeGeminiBaseURL(tt.in); got != tt.want {
			t.Fatalf("normalizeGeminiBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGeminiProvider_MapsToolsAndToolResponses(t *testing.T) {
	type requestShape struct {
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text         string `json:"text,omitempty"`
				FunctionCall *struct {
					ID   string         `json:"id,omitempty"`
					Name string         `json:"name"`
					Args map[string]any `json:"args,omitempty"`
				} `json:"functionCall,omitempty"`
				FunctionResponse *struct {
					ID       string         `json:"id,omitempty"`
					Name     string         `json:"name"`
					Response map[string]any `json:"response,omitempty"`
				} `json:"functionResponse,omitempty"`
			} `json:"parts"`
		} `json:"contents"`
		ToolConfig *struct {
			FunctionCallingConfig *struct {
				Mode string `json:"mode"`
			} `json:"functionCallingConfig"`
		} `json:"toolConfig,omitempty"`
		Tools []struct {
			FunctionDeclarations []struct {
				Name string `json:"name"`
			} `json:"functionDeclarations"`
		} `json:"tools"`
	}

	var gotReq requestShape

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if err := json.Unmarshal(body, &gotReq); err != nil {
			t.Fatalf("request json decode error = %v\nbody=%s", err, string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": {"role":"model","parts":[
					{"functionCall":{"id":"call_123","name":"read_file","args":{"path":"/tmp/a.txt"}}}
				]},
				"finishReason":"STOP"
			}]
		}`))
	}))
	defer srv.Close()

	p, err := NewGeminiProvider(GeminiConfig{
		APIKey:  "gem-test-key",
		BaseURL: srv.URL,
		Model:   "gemini-test-model",
	})
	if err != nil {
		t.Fatalf("NewGeminiProvider() error = %v", err)
	}

	resp, err := p.Chat(context.Background(), ChatRequest{
		SystemPrompt: "use tools",
		ForceToolUse: true,
		Tools: []Tool{{
			Name:        "read_file",
			Description: "Read a file",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		}},
		Messages: []Message{
			{Role: "user", Content: "read it"},
			{Role: "assistant", ToolCalls: []ToolCall{{
				ID:    "call_1",
				Name:  "read_file",
				Input: json.RawMessage(`{"path":"/tmp/a.txt"}`),
			}}},
			{Role: "user", ToolResult: &ToolResult{
				ToolCallID: "call_1",
				Content:    `{"body":"file text"}`,
			}},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if gotReq.ToolConfig == nil || gotReq.ToolConfig.FunctionCallingConfig == nil || gotReq.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("toolConfig.functionCallingConfig.mode = %#v, want ANY", gotReq.ToolConfig)
	}
	if len(gotReq.Tools) != 1 || len(gotReq.Tools[0].FunctionDeclarations) != 1 || gotReq.Tools[0].FunctionDeclarations[0].Name != "read_file" {
		t.Fatalf("tools = %#v, want read_file declaration", gotReq.Tools)
	}
	if len(gotReq.Contents) != 3 {
		t.Fatalf("contents length = %d, want 3", len(gotReq.Contents))
	}
	if gotReq.Contents[1].Role != "model" || gotReq.Contents[1].Parts[0].FunctionCall == nil || gotReq.Contents[1].Parts[0].FunctionCall.Name != "read_file" {
		t.Fatalf("assistant tool call mapping wrong: %#v", gotReq.Contents[1])
	}
	if gotReq.Contents[2].Role != "user" || gotReq.Contents[2].Parts[0].FunctionResponse == nil || gotReq.Contents[2].Parts[0].FunctionResponse.Name != "read_file" {
		t.Fatalf("tool response mapping wrong: %#v", gotReq.Contents[2])
	}
	if resp.FinishReason != "tool_use" || len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("response tool mapping wrong: %#v", resp)
	}
}
