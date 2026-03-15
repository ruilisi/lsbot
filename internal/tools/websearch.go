package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// WebSearch performs a web search using DuckDuckGo
func WebSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, ok := req.Params.Arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	// Use DuckDuckGo HTML version for simple results
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", encodedQuery)

	client := &http.Client{Timeout: 15 * time.Second}
	req2, _ := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (compatible; lsbot/1.0)")

	resp, err := client.Do(req2)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// Extract search results (simplified parsing)
	results := parseSearchResults(string(body))
	if results == "" {
		return mcp.NewToolResultText("No results found"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Search results for '%s':\n\n%s", query, results)), nil
}

// parseSearchResults extracts results from DuckDuckGo HTML
func parseSearchResults(html string) string {
	var results []string
	maxResults := 5

	// Find result links (simplified extraction)
	// DuckDuckGo HTML uses class="result__a" for result links
	parts := strings.Split(html, `class="result__a"`)

	for i, part := range parts[1:] {
		if i >= maxResults {
			break
		}

		// Extract href
		hrefStart := strings.Index(part, `href="`)
		if hrefStart == -1 {
			continue
		}
		hrefStart += 6
		hrefEnd := strings.Index(part[hrefStart:], `"`)
		if hrefEnd == -1 {
			continue
		}
		href := part[hrefStart : hrefStart+hrefEnd]

		// Clean up DuckDuckGo redirect URL
		if strings.Contains(href, "duckduckgo.com") {
			if uddg := extractParam(href, "uddg"); uddg != "" {
				href = uddg
			}
		}

		// Extract title (text between > and </a>)
		titleStart := strings.Index(part, ">")
		if titleStart == -1 {
			continue
		}
		titleStart++
		titleEnd := strings.Index(part[titleStart:], "</a>")
		if titleEnd == -1 {
			continue
		}
		title := part[titleStart : titleStart+titleEnd]
		title = stripTags(title)
		title = strings.TrimSpace(title)

		if title != "" && href != "" {
			results = append(results, fmt.Sprintf("%d. %s\n   %s", i+1, title, href))
		}
	}

	return strings.Join(results, "\n\n")
}

// extractParam extracts a URL parameter value
func extractParam(urlStr, param string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return u.Query().Get(param)
}

// stripTags removes HTML tags from a string
func stripTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// WebFetch fetches content from a URL
func WebFetch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	urlStr, ok := req.Params.Arguments["url"].(string)
	if !ok || urlStr == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req2, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid URL: %v", err)), nil
	}
	req2.Header.Set("User-Agent", "Mozilla/5.0 (compatible; lsbot/1.0)")

	resp, err := client.Do(req2)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("fetch failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// Limit response size
	body, err := io.ReadAll(io.LimitReader(resp.Body, 100*1024)) // 100KB limit
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// For HTML, extract text content
	content := string(body)
	if strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		content = extractTextFromHTML(content)
	}

	// Truncate if too long
	if len(content) > 10000 {
		content = content[:10000] + "\n... (truncated)"
	}

	return mcp.NewToolResultText(content), nil
}

// extractTextFromHTML extracts readable text from HTML
func extractTextFromHTML(html string) string {
	// Remove script and style blocks
	for _, tag := range []string{"script", "style", "noscript"} {
		for {
			start := strings.Index(strings.ToLower(html), "<"+tag)
			if start == -1 {
				break
			}
			end := strings.Index(strings.ToLower(html[start:]), "</"+tag+">")
			if end == -1 {
				break
			}
			html = html[:start] + html[start+end+len("</"+tag+">"):]
		}
	}

	// Strip remaining tags
	text := stripTags(html)

	// Clean up whitespace
	lines := strings.Split(text, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}

	return strings.Join(cleaned, "\n")
}
