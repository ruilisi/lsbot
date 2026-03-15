// Package mcpclient manages connections to external MCP servers and exposes
// their tools to the lsbot agent.
package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruilisi/lsbot/internal/logger"
)

// ServerConfig describes one external MCP server.
type ServerConfig struct {
	// Name is a human-readable label used as a namespace prefix for tool names.
	// E.g. "chrome" → tools become "mcp_chrome_take_snapshot", etc.
	Name string `yaml:"name"`

	// Command + Args launch a stdio MCP server subprocess.
	// Either Command or URL must be set.
	Command string   `yaml:"command,omitempty"`
	Args    []string `yaml:"args,omitempty"`
	Env     []string `yaml:"env,omitempty"`

	// URL connects to an SSE-based MCP server.
	URL string `yaml:"url,omitempty"`
}

// Tool is a discovered tool from an external MCP server, ready to be exposed
// to the AI agent.
type Tool struct {
	// FullName is the namespaced tool name exposed to the AI: "mcp_<name>_<tool>".
	FullName    string
	Description string
	InputSchema json.RawMessage

	server *serverConn // back-reference for calling
	// original tool name on the MCP server
	remoteName string
}

// serverConn holds a live connection to one MCP server.
type serverConn struct {
	cfg    ServerConfig
	client *mcpgo.Client
	tools  []Tool
	mu     sync.Mutex
}

// Manager owns all external MCP server connections.
type Manager struct {
	servers []*serverConn
}

// New creates a Manager and connects to all configured servers.
// Servers that fail to connect are logged and skipped (non-fatal).
func New(cfgs []ServerConfig) *Manager {
	m := &Manager{}
	for _, cfg := range cfgs {
		conn := &serverConn{cfg: cfg}
		if err := conn.connect(); err != nil {
			logger.Warn("[MCP] failed to connect to server %q: %v", cfg.Name, err)
			continue
		}
		if err := conn.discoverTools(); err != nil {
			logger.Warn("[MCP] failed to list tools from server %q: %v", cfg.Name, err)
			continue
		}
		logger.Info("[MCP] connected to %q, %d tools available", cfg.Name, len(conn.tools))
		m.servers = append(m.servers, conn)
	}
	return m
}

// AllTools returns all discovered tools across all connected servers.
func (m *Manager) AllTools() []Tool {
	var out []Tool
	for _, s := range m.servers {
		out = append(out, s.tools...)
	}
	return out
}

// Call invokes a tool by its full namespaced name and returns the result as a string.
func (m *Manager) Call(ctx context.Context, fullName string, args map[string]any) (string, error) {
	for _, s := range m.servers {
		for _, t := range s.tools {
			if t.FullName == fullName {
				return s.call(ctx, t.remoteName, args)
			}
		}
	}
	return "", fmt.Errorf("MCP tool %q not found", fullName)
}

// Close shuts down all server connections.
func (m *Manager) Close() {
	for _, s := range m.servers {
		if s.client != nil {
			_ = s.client.Close()
		}
	}
}

// IsMCPTool returns true if the tool name belongs to an external MCP server.
func IsMCPTool(name string) bool {
	return strings.HasPrefix(name, "mcp_")
}

// --- serverConn internals ---

func (s *serverConn) connect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var c *mcpgo.Client
	var err error

	if s.cfg.Command != "" {
		c, err = mcpgo.NewStdioMCPClient(s.cfg.Command, s.cfg.Env, s.cfg.Args...)
		if err != nil {
			return fmt.Errorf("stdio connect: %w", err)
		}
	} else if s.cfg.URL != "" {
		c, err = mcpgo.NewSSEMCPClient(s.cfg.URL)
		if err != nil {
			return fmt.Errorf("SSE connect: %w", err)
		}
		if err := c.Start(ctx); err != nil {
			return fmt.Errorf("SSE start: %w", err)
		}
	} else {
		return fmt.Errorf("server %q: either command or url must be set", s.cfg.Name)
	}

	// MCP handshake
	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "lsbot", Version: "1.0"}
	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		_ = c.Close()
		return fmt.Errorf("initialize: %w", err)
	}

	s.client = c
	return nil
}

func (s *serverConn) discoverTools() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := s.client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return err
	}

	prefix := "mcp_" + sanitizeName(s.cfg.Name) + "_"
	for _, t := range result.Tools {
		schema, err := toolSchema(t)
		if err != nil {
			logger.Warn("[MCP] skipping tool %q from %q: %v", t.Name, s.cfg.Name, err)
			continue
		}
		s.tools = append(s.tools, Tool{
			FullName:    prefix + sanitizeName(t.Name),
			Description: t.Description,
			InputSchema: schema,
			server:      s,
			remoteName:  t.Name,
		})
	}
	return nil
}

func (s *serverConn) call(ctx context.Context, toolName string, args map[string]any) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	if args == nil {
		args = map[string]any{}
	}
	req.Params.Arguments = args

	result, err := s.client.CallTool(ctx, req)
	if err != nil {
		return "", fmt.Errorf("call %q: %w", toolName, err)
	}

	return extractResult(result), nil
}

// extractResult converts an MCP tool result to a plain string.
func extractResult(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, c := range result.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			parts = append(parts, v.Text)
		case mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[image: %s]", v.MIMEType))
		default:
			if b, err := json.Marshal(v); err == nil {
				parts = append(parts, string(b))
			}
		}
	}
	if result.IsError {
		return "Error: " + strings.Join(parts, "\n")
	}
	return strings.Join(parts, "\n")
}

// toolSchema marshals the MCP tool's input schema to raw JSON for the agent.
func toolSchema(t mcp.Tool) (json.RawMessage, error) {
	b, err := json.Marshal(t.InputSchema)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// sanitizeName converts a name like "chrome-devtools" or "take_snapshot" to
// a snake_case identifier safe for use in tool names.
func sanitizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}
