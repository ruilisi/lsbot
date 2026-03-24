package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	cronpkg "github.com/ruilisi/lsbot/internal/cron"
	"github.com/ruilisi/lsbot/internal/security"
	"github.com/ruilisi/lsbot/internal/tools"
)

const ServerName = "lsbot"

// ServerVersion is set via ldflags at build time
var ServerVersion = "2.1.2"

// ToolHandler is a function that handles tool calls
type ToolHandler func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)

// Server wraps the MCP server and adds cron scheduling capabilities
type Server struct {
	mcpServer     *server.MCPServer
	cronScheduler *cronpkg.Scheduler
	toolHandlers  map[string]ToolHandler
	pathChecker      *security.PathChecker
	disableFileTools bool
}

// SecurityOptions holds security settings for the MCP server.
type SecurityOptions struct {
	AllowedPaths     []string
	DisableFileTools bool
}

// NewServer creates a new MCP server with all tools registered
func NewServer(opts ...SecurityOptions) *Server {
	var opt SecurityOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	s := &Server{
		mcpServer: server.NewMCPServer(ServerName, ServerVersion,
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
		),
		toolHandlers:     make(map[string]ToolHandler),
		pathChecker:      security.NewPathChecker(opt.AllowedPaths),
		disableFileTools: opt.DisableFileTools,
	}

	// Register all tools
	registerFilesystemTools(s)
	registerShellTools(s)
	registerSystemTools(s)
	registerProcessTools(s)
	registerNetworkTools(s)
	registerCalendarTools(s)
	registerFileManagerTools(s)
	registerBrowserTools(s)

	// Initialize cron scheduler
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[CRON] Warning: Failed to get home directory: %v", err)
		homeDir = os.TempDir()
	}
	cronPath := filepath.Join(homeDir, ".lsbot.db")
	cronStore, err := cronpkg.NewStore(cronPath)
	if err != nil {
		log.Printf("[CRON] Warning: Failed to open cron store: %v", err)
		cronStore, _ = cronpkg.NewStore(filepath.Join(os.TempDir(), "lsbot.db"))
	}
	s.cronScheduler = cronpkg.NewScheduler(cronStore, s, nil, s)

	// Register cron tools
	registerCronTools(s)

	// Set global scheduler for cron tools
	SetCronScheduler(s.cronScheduler)

	// Start the scheduler
	if err := s.cronScheduler.Start(); err != nil {
		log.Printf("[CRON] Warning: Failed to start cron scheduler: %v", err)
	}

	return s
}

// GetMCPServer returns the underlying MCP server
func (s *Server) GetMCPServer() *server.MCPServer {
	return s.mcpServer
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	if s.cronScheduler != nil {
		return s.cronScheduler.Stop()
	}
	return nil
}

// ExecuteTool implements the ToolExecutor interface for the cron scheduler
func (s *Server) ExecuteTool(ctx context.Context, toolName string, arguments map[string]any) (any, error) {
	handler, exists := s.toolHandlers[toolName]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Create a CallToolRequest
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = arguments

	// Execute the tool
	result, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}

	// Return the result content
	if len(result.Content) > 0 {
		return result.Content[0], nil
	}

	return nil, nil
}

// NotifyChat implements the ChatNotifier interface for the cron scheduler
func (s *Server) NotifyChat(message string) error {
	// In MCP mode, we just log to console as there's no persistent chat session
	// The error will also be logged by the scheduler
	log.Printf("[CRON] Notification: %s", message)
	return nil
}

// NotifyChatUser implements the ChatNotifier interface for the cron scheduler
func (s *Server) NotifyChatUser(platform, channelID, userID, message string) error {
	// In MCP mode, there's no router to send messages through, so just log
	log.Printf("[CRON] Notification to %s/%s: %s", platform, channelID, message)
	return nil
}

// pathCheckedTools maps tool names to the argument key containing a file path.
var pathCheckedTools = map[string]string{
	"file_read":     "path",
	"file_write":    "path",
	"file_list":     "path",
	"file_search":   "path",
	"file_info":     "path",
	"file_trash":    "path",
	"file_list_old": "path",
}

// addTool is a helper to add a tool and track its handler
func (s *Server) addTool(tool mcp.Tool, handler ToolHandler) {
	wrappedHandler := handler
	if _, ok := pathCheckedTools[tool.Name]; ok {
		if s.disableFileTools {
			wrappedHandler = func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				return mcp.NewToolResultError("ACCESS DENIED: file operations are disabled by security policy. Do NOT retry. Inform the user that file access is disabled."), nil
			}
		} else {
			wrappedHandler = s.wrapPathCheck("path", handler)
		}
	} else if tool.Name == "shell_execute" {
		wrappedHandler = s.wrapPathCheck("working_directory", handler)
	}
	s.mcpServer.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return wrappedHandler(ctx, req)
	})
	s.toolHandlers[tool.Name] = handler
}

func (s *Server) wrapPathCheck(argKey string, handler ToolHandler) ToolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if s.pathChecker.HasRestrictions() {
			path := "."
			if p, ok := req.Params.Arguments[argKey].(string); ok && p != "" {
				path = p
			}
			if err := s.pathChecker.CheckPath(path); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
		}
		return handler(ctx, req)
	}
}

func registerFilesystemTools(s *Server) {
	// file_read
	s.addTool(mcp.NewTool("file_read",
		mcp.WithDescription("Read the contents of a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to read")),
	), tools.FileRead)

	// file_write
	s.addTool(mcp.NewTool("file_write",
		mcp.WithDescription("Write content to a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file to write")),
		mcp.WithString("content", mcp.Required(), mcp.Description("Content to write to the file")),
	), tools.FileWrite)

	// file_list
	s.addTool(mcp.NewTool("file_list",
		mcp.WithDescription("List contents of a directory"),
		mcp.WithString("path", mcp.Description("Path to the directory (default: current directory)")),
	), tools.FileList)

	// file_search
	s.addTool(mcp.NewTool("file_search",
		mcp.WithDescription("Search for files matching a pattern"),
		mcp.WithString("pattern", mcp.Required(), mcp.Description("Glob pattern to match (e.g., *.go, *.txt)")),
		mcp.WithString("path", mcp.Description("Directory to search in (default: current directory)")),
	), tools.FileSearch)

	// file_info
	s.addTool(mcp.NewTool("file_info",
		mcp.WithDescription("Get detailed information about a file"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Path to the file")),
	), tools.FileInfo)
}

func registerShellTools(s *Server) {
	// shell_execute
	s.addTool(mcp.NewTool("shell_execute",
		mcp.WithDescription("Execute a shell command"),
		mcp.WithString("command", mcp.Required(), mcp.Description("The command to execute")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 30)")),
		mcp.WithString("working_directory", mcp.Description("Working directory for the command")),
	), tools.ShellExecute)

	// shell_which
	s.addTool(mcp.NewTool("shell_which",
		mcp.WithDescription("Find the path of an executable"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the executable to find")),
	), tools.ShellWhich)
}

func registerSystemTools(s *Server) {
	// system_info
	s.addTool(mcp.NewTool("system_info",
		mcp.WithDescription("Get system information (CPU, memory, OS)"),
	), tools.SystemInfo)

	// disk_usage
	s.addTool(mcp.NewTool("disk_usage",
		mcp.WithDescription("Get disk usage information"),
		mcp.WithString("path", mcp.Description("Path to check (default: /)")),
	), tools.DiskUsage)

	// env_get
	s.addTool(mcp.NewTool("env_get",
		mcp.WithDescription("Get an environment variable"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name of the environment variable")),
	), tools.EnvGet)

	// env_list
	s.addTool(mcp.NewTool("env_list",
		mcp.WithDescription("List all environment variables"),
	), tools.EnvList)
}

func registerProcessTools(s *Server) {
	// process_list
	s.addTool(mcp.NewTool("process_list",
		mcp.WithDescription("List running processes"),
		mcp.WithString("filter", mcp.Description("Filter processes by name (optional)")),
	), tools.ProcessList)

	// process_info
	s.addTool(mcp.NewTool("process_info",
		mcp.WithDescription("Get detailed information about a process"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID")),
	), tools.ProcessInfo)

	// process_kill
	s.addTool(mcp.NewTool("process_kill",
		mcp.WithDescription("Kill a process by PID"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("Process ID to kill")),
	), tools.ProcessKill)
}

func registerNetworkTools(s *Server) {
	// network_interfaces
	s.addTool(mcp.NewTool("network_interfaces",
		mcp.WithDescription("List network interfaces"),
	), tools.NetworkInterfaces)

	// network_connections
	s.addTool(mcp.NewTool("network_connections",
		mcp.WithDescription("List active network connections"),
		mcp.WithString("kind", mcp.Description("Connection type: tcp, udp, tcp4, tcp6, udp4, udp6, all (default: all)")),
	), tools.NetworkConnections)

	// network_ping
	s.addTool(mcp.NewTool("network_ping",
		mcp.WithDescription("Ping a host (TCP connect test)"),
		mcp.WithString("host", mcp.Required(), mcp.Description("Host to ping")),
		mcp.WithString("port", mcp.Description("Port to connect to (default: 80)")),
		mcp.WithNumber("timeout", mcp.Description("Timeout in seconds (default: 5)")),
	), tools.NetworkPing)

	// network_dns_lookup
	s.addTool(mcp.NewTool("network_dns_lookup",
		mcp.WithDescription("Perform DNS lookup for a hostname"),
		mcp.WithString("hostname", mcp.Required(), mcp.Description("Hostname to look up")),
	), tools.NetworkDNSLookup)
}

func registerCalendarTools(s *Server) {
	// calendar_list_events
	s.addTool(mcp.NewTool("calendar_list_events",
		mcp.WithDescription("List upcoming calendar events from macOS Calendar"),
		mcp.WithNumber("days", mcp.Description("Number of days to look ahead (default: 7)")),
	), tools.CalendarListEvents)

	// calendar_create_event
	s.addTool(mcp.NewTool("calendar_create_event",
		mcp.WithDescription("Create a new event in macOS Calendar"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Event title")),
		mcp.WithString("start_time", mcp.Required(), mcp.Description("Start time (format: 2024-01-15 14:00)")),
		mcp.WithNumber("duration", mcp.Description("Duration in minutes (default: 60)")),
		mcp.WithString("calendar", mcp.Description("Calendar name (default: Calendar)")),
		mcp.WithString("location", mcp.Description("Event location")),
		mcp.WithString("notes", mcp.Description("Event notes")),
	), tools.CalendarCreateEvent)

	// calendar_list_calendars
	s.addTool(mcp.NewTool("calendar_list_calendars",
		mcp.WithDescription("List available calendars"),
	), tools.CalendarListCalendars)

	// calendar_today
	s.addTool(mcp.NewTool("calendar_today",
		mcp.WithDescription("List calendar events/meetings scheduled for today. Only use when the user asks about their schedule, agenda, or appointments — NOT for asking the current date/time"),
	), tools.CalendarToday)

	// calendar_search
	s.addTool(mcp.NewTool("calendar_search",
		mcp.WithDescription("Search for events by keyword"),
		mcp.WithString("keyword", mcp.Required(), mcp.Description("Keyword to search for in event titles")),
		mcp.WithNumber("days", mcp.Description("Number of days to search ahead (default: 30)")),
	), tools.CalendarSearchEvents)

	// calendar_delete_event
	s.addTool(mcp.NewTool("calendar_delete_event",
		mcp.WithDescription("Delete a calendar event by title"),
		mcp.WithString("title", mcp.Required(), mcp.Description("Exact title of the event to delete")),
		mcp.WithString("calendar", mcp.Description("Calendar name to search in (optional)")),
		mcp.WithString("date", mcp.Description("Date of the event (format: 2024-01-15, optional)")),
	), tools.CalendarDeleteEvent)
}

func registerFileManagerTools(s *Server) {
	// file_list_old
	s.addTool(mcp.NewTool("file_list_old",
		mcp.WithDescription("List files that haven't been modified for a specified number of days"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to scan (e.g., ~/Desktop)")),
		mcp.WithNumber("days", mcp.Description("Minimum days since last modification (default: 30)")),
	), tools.FileListOld)

	// file_delete_old
	s.addTool(mcp.NewTool("file_delete_old",
		mcp.WithDescription("Delete files that haven't been modified for a specified number of days"),
		mcp.WithString("path", mcp.Required(), mcp.Description("Directory path to clean (e.g., ~/Desktop)")),
		mcp.WithNumber("days", mcp.Description("Minimum days since last modification (default: 30)")),
		mcp.WithBoolean("include_dirs", mcp.Description("Also delete old directories (default: false)")),
		mcp.WithBoolean("dry_run", mcp.Description("Only show what would be deleted without actually deleting (default: false)")),
	), tools.FileDeleteOld)

	// file_delete_list
	s.addTool(mcp.NewTool("file_delete_list",
		mcp.WithDescription("Delete specific files by their paths"),
		mcp.WithArray("files", mcp.Required(), mcp.Description("Array of file paths to delete")),
	), tools.FileDeleteList)

	// file_trash
	s.addTool(mcp.NewTool("file_trash",
		mcp.WithDescription("Move files to Trash instead of permanently deleting (macOS)"),
		mcp.WithArray("files", mcp.Required(), mcp.Description("Array of file paths to move to Trash")),
	), tools.FileMoveToTrash)
}

func registerBrowserTools(s *Server) {
	// browser_start
	s.addTool(mcp.NewTool("browser_start",
		mcp.WithDescription("Launch a browser for automation (snapshot-then-act pattern). Uses an isolated profile."),
		mcp.WithBoolean("headless", mcp.Description("Run in headless mode without visible window (default: true)")),
		mcp.WithString("url", mcp.Description("Initial URL to navigate to after launch")),
		mcp.WithString("executable_path", mcp.Description("Path to browser executable (auto-detected if omitted)")),
	), tools.BrowserStart)

	// browser_stop
	s.addTool(mcp.NewTool("browser_stop",
		mcp.WithDescription("Close the browser"),
	), tools.BrowserStop)

	// browser_status
	s.addTool(mcp.NewTool("browser_status",
		mcp.WithDescription("Check if the browser is running and get current state"),
	), tools.BrowserStatus)

	// browser_navigate
	s.addTool(mcp.NewTool("browser_navigate",
		mcp.WithDescription("Navigate to a URL. Auto-starts headless browser if not running."),
		mcp.WithString("url", mcp.Required(), mcp.Description("URL to navigate to")),
	), tools.BrowserNavigate)

	// browser_snapshot
	s.addTool(mcp.NewTool("browser_snapshot",
		mcp.WithDescription("Capture the page accessibility tree with numbered refs. Use these refs with browser_click/browser_type to interact with elements. Re-run after page changes."),
	), tools.BrowserSnapshot)

	// browser_screenshot
	s.addTool(mcp.NewTool("browser_screenshot",
		mcp.WithDescription("Take a screenshot of the current page"),
		mcp.WithString("path", mcp.Description("Output file path (default: ~/Desktop/browser_screenshot_<timestamp>.png)")),
		mcp.WithBoolean("full_page", mcp.Description("Capture the full scrollable page (default: false)")),
	), tools.BrowserScreenshot)

	// browser_click
	s.addTool(mcp.NewTool("browser_click",
		mcp.WithDescription("Click an element by its ref number from browser_snapshot"),
		mcp.WithNumber("ref", mcp.Required(), mcp.Description("Element ref number from browser_snapshot")),
	), tools.BrowserClick)

	// browser_type
	s.addTool(mcp.NewTool("browser_type",
		mcp.WithDescription("Type text into an element by its ref number from browser_snapshot"),
		mcp.WithNumber("ref", mcp.Required(), mcp.Description("Element ref number from browser_snapshot")),
		mcp.WithString("text", mcp.Required(), mcp.Description("Text to type")),
		mcp.WithBoolean("submit", mcp.Description("Press Enter after typing (default: false)")),
	), tools.BrowserType)

	// browser_press
	s.addTool(mcp.NewTool("browser_press",
		mcp.WithDescription("Press a keyboard key (Enter, Tab, Escape, Backspace, ArrowUp, ArrowDown, ArrowLeft, ArrowRight, Space, Delete, Home, End, PageUp, PageDown)"),
		mcp.WithString("key", mcp.Required(), mcp.Description("Key name to press")),
	), tools.BrowserPress)

	// browser_execute_js
	s.addTool(mcp.NewTool("browser_execute_js",
		mcp.WithDescription("Execute JavaScript on the current page. The script runs as a function body — use 'return expr' to get a value back. Statements like forEach/for loops work without return."),
		mcp.WithString("script", mcp.Required(), mcp.Description("JavaScript code to execute as function body (use 'return' to get values back)")),
	), tools.BrowserExecuteJS)

	// browser_tabs
	s.addTool(mcp.NewTool("browser_tabs",
		mcp.WithDescription("List all open browser tabs with their target IDs and URLs"),
	), tools.BrowserTabs)

	// browser_tab_open
	s.addTool(mcp.NewTool("browser_tab_open",
		mcp.WithDescription("Open a new browser tab"),
		mcp.WithString("url", mcp.Description("URL to open (default: about:blank)")),
	), tools.BrowserTabOpen)

	// browser_tab_close
	s.addTool(mcp.NewTool("browser_tab_close",
		mcp.WithDescription("Close a browser tab by target ID, or close the active tab if no ID given"),
		mcp.WithString("target_id", mcp.Description("Target ID of the tab to close (from browser_tabs)")),
	), tools.BrowserTabClose)
}
