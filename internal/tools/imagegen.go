package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruilisi/lsbot/internal/config"
)

// ImageGenerate calls the OpenAI Images API (DALL-E 3 by default) and saves
// the result to a temp file, returning the local path so the agent can send
// it as a file attachment.
//
// Environment variables:
//
//	OPENAI_API_KEY  — required
//	IMAGE_GEN_MODEL — model name (default: dall-e-3)
//	IMAGE_GEN_URL   — base URL override (default: https://api.openai.com/v1)
func ImageGenerate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return mcp.NewToolResultError("OPENAI_API_KEY is not set"), nil
	}

	prompt, _ := req.Params.Arguments["prompt"].(string)
	if strings.TrimSpace(prompt) == "" {
		return mcp.NewToolResultError("prompt is required"), nil
	}

	model := os.Getenv("IMAGE_GEN_MODEL")
	if model == "" {
		model = "dall-e-3"
	}

	size, _ := req.Params.Arguments["size"].(string)
	if size == "" {
		size = "1024x1024"
	}

	quality, _ := req.Params.Arguments["quality"].(string)
	if quality == "" {
		quality = "standard"
	}

	baseURL := os.Getenv("IMAGE_GEN_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/images/generations"

	body, _ := json.Marshal(map[string]any{
		"model":           model,
		"prompt":          prompt,
		"n":               1,
		"size":            size,
		"quality":         quality,
		"response_format": "url",
	})

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("build request: %v", err)), nil
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("API request: %v", err)), nil
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return mcp.NewToolResultError(fmt.Sprintf("API error %d: %s", resp.StatusCode, raw)), nil
	}

	var result struct {
		Data []struct {
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Data) == 0 {
		return mcp.NewToolResultError("unexpected API response: " + string(raw)), nil
	}

	imageURL := result.Data[0].URL
	revisedPrompt := result.Data[0].RevisedPrompt

	// Download the image to a local temp file
	localPath, err := downloadImage(ctx, imageURL)
	if err != nil {
		// Return the URL so the user can still access it
		return mcp.NewToolResultText(fmt.Sprintf("Image generated (could not download locally): %s\nRevised prompt: %s", imageURL, revisedPrompt)), nil
	}

	msg := fmt.Sprintf("image_path:%s", localPath)
	if revisedPrompt != "" && revisedPrompt != prompt {
		msg += fmt.Sprintf("\nrevised_prompt:%s", revisedPrompt)
	}
	return mcp.NewToolResultText(msg), nil
}

func downloadImage(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Determine extension from Content-Type
	ext := ".png"
	if ct := resp.Header.Get("Content-Type"); strings.Contains(ct, "jpeg") || strings.Contains(ct, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(ct, "webp") {
		ext = ".webp"
	}

	outDir := filepath.Join(config.HubDir(), "generated_images")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}

	fname := fmt.Sprintf("img_%d%s", time.Now().UnixMilli(), ext)
	outPath := filepath.Join(outDir, fname)

	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", err
	}
	return outPath, nil
}
