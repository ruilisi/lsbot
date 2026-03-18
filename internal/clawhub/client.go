package clawhub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const DefaultRegistry = "https://clawhub.ai"

type Client struct {
	registry   string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		registry:   DefaultRegistry,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type SearchResult struct {
	Slug        string  `json:"slug"`
	DisplayName string  `json:"displayName"`
	Summary     string  `json:"summary"`
	Version     string  `json:"version"`
	Score       float64 `json:"score"`
}

type SkillInfo struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Summary     string `json:"summary"`
}

type VersionInfo struct {
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type OwnerInfo struct {
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

type SkillDetail struct {
	Skill         SkillInfo   `json:"skill"`
	LatestVersion VersionInfo `json:"latestVersion"`
	Owner         OwnerInfo   `json:"owner"`
}

type ResolveResult struct {
	Slug          string `json:"slug"`
	Match         bool   `json:"match"`
	LatestVersion string `json:"latestVersion"`
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("limit", fmt.Sprintf("%d", limit))
	params.Set("nonSuspiciousOnly", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.registry+"/api/v1/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search failed: HTTP %d", resp.StatusCode)
	}

	var wrapper struct {
		Results []SearchResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}
	return wrapper.Results, nil
}

func (c *Client) GetSkill(ctx context.Context, slug string) (*SkillDetail, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.registry+"/api/v1/skills/"+url.PathEscape(slug), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skill %q not found", slug)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get skill failed: HTTP %d", resp.StatusCode)
	}

	var detail SkillDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to decode skill response: %w", err)
	}
	return &detail, nil
}

func (c *Client) Download(ctx context.Context, slug, version string) (io.ReadCloser, error) {
	params := url.Values{}
	params.Set("slug", slug)
	if version != "" {
		params.Set("version", version)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.registry+"/api/v1/download?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *Client) Resolve(ctx context.Context, slug, hash string) (*ResolveResult, error) {
	params := url.Values{}
	params.Set("slug", slug)
	if hash != "" {
		params.Set("hash", hash)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.registry+"/api/v1/resolve?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resolve failed: HTTP %d", resp.StatusCode)
	}

	var result ResolveResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode resolve response: %w", err)
	}
	return &result, nil
}
