package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/tools"
)

// dangerousPatterns is the comprehensive list of shell patterns that are
// blocked before execution. Each entry is (compiled-regex, human description).
// Ported and adapted from hermes-agent tools/approval.py.
var dangerousPatterns = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`(?i)\brm\s+(-[^\s]*\s+)*/`), "delete in root path"},
	{regexp.MustCompile(`(?i)\brm\s+-[^\s]*r`), "recursive delete"},
	{regexp.MustCompile(`(?i)\brm\s+--recursive\b`), "recursive delete (long flag)"},
	{regexp.MustCompile(`(?i)\bchmod\s+(-[^\s]*\s+)*(777|666|o\+[rwx]*w|a\+[rwx]*w)\b`), "world/other-writable permissions"},
	{regexp.MustCompile(`(?i)\bchown\s+(-[^\s]*)?R\s+root`), "recursive chown to root"},
	{regexp.MustCompile(`(?i)\bmkfs\b`), "format filesystem"},
	{regexp.MustCompile(`(?i)\bdd\s+.*if=`), "disk copy"},
	{regexp.MustCompile(`(?i)>\s*/dev/sd`), "write to block device"},
	{regexp.MustCompile(`(?i)\bDROP\s+(TABLE|DATABASE)\b`), "SQL DROP"},
	// Note: "DELETE FROM without WHERE" is handled programmatically in detectDangerousCommand
	// because Go's RE2 doesn't support negative lookaheads.
	{regexp.MustCompile(`(?i)\bTRUNCATE\s+(TABLE)?\s*\w`), "SQL TRUNCATE"},
	{regexp.MustCompile(`(?i)>\s*/etc/`), "overwrite system config"},
	{regexp.MustCompile(`(?i)\bsystemctl\s+(-[^\s]+\s+)*(stop|restart|disable|mask)\b`), "stop/restart system service"},
	{regexp.MustCompile(`(?i)\bkill\s+-9\s+-1\b`), "kill all processes"},
	{regexp.MustCompile(`(?i)\bpkill\s+-9\b`), "force kill processes"},
	{regexp.MustCompile(`(?i):\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`), "fork bomb"},
	{regexp.MustCompile(`(?i)\b(bash|sh|zsh|ksh)\s+-[^\s]*c(\s+|$)`), "shell command via -c/-lc flag"},
	{regexp.MustCompile(`(?i)\b(python[23]?|perl|ruby|node)\s+-[ec]\s+`), "script execution via -e/-c flag"},
	{regexp.MustCompile(`(?i)\b(curl|wget)\b.*\|\s*(ba)?sh\b`), "pipe remote content to shell"},
	{regexp.MustCompile(`(?i)\bxargs\s+.*\brm\b`), "xargs with rm"},
	{regexp.MustCompile(`(?i)\bfind\b.*-exec\s+(/\S*/)?rm\b`), "find -exec rm"},
	{regexp.MustCompile(`(?i)\bfind\b.*-delete\b`), "find -delete"},
	{regexp.MustCompile(`(?i)\bgit\s+reset\s+--hard\b`), "git reset --hard (destroys uncommitted changes)"},
	{regexp.MustCompile(`(?i)\bgit\s+push\b.*--force\b`), "git force push"},
	{regexp.MustCompile(`(?i)\bgit\s+push\b.*-f\b`), "git force push short flag"},
	{regexp.MustCompile(`(?i)\bgit\s+clean\s+-[^\s]*f`), "git clean with force (deletes untracked files)"},
	{regexp.MustCompile(`(?i)\bgit\s+branch\s+-D\b`), "git branch force delete"},
	{regexp.MustCompile(`(?i)\b(cp|mv|install)\b.*\s/etc/`), "copy/move file into /etc/"},
	{regexp.MustCompile(`(?i)\bsed\s+-[^\s]*i.*\s/etc/`), "in-place edit of system config"},
}

var deleteFromRe = regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`)
var whereRe = regexp.MustCompile(`(?i)\bWHERE\b`)

// detectDangerousCommand returns (true, description) if the command matches
// any dangerous pattern, (false, "") otherwise.
func detectDangerousCommand(cmd string) (bool, string) {
	for _, p := range dangerousPatterns {
		if p.re.MatchString(cmd) {
			return true, p.desc
		}
	}
	// DELETE FROM without WHERE — handled separately because Go's RE2 doesn't
	// support negative lookaheads.
	if deleteFromRe.MatchString(cmd) && !whereRe.MatchString(cmd) {
		return true, "SQL DELETE without WHERE"
	}
	return false, ""
}

// executeSystemInfo runs the system_info tool
func executeSystemInfo(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tools.SystemInfo(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeProcessList runs the process_list tool
func executeProcessList(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	if filter, ok := args["filter"].(string); ok {
		req.Params.Arguments["filter"] = filter
	}

	result, err := tools.ProcessList(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeCalendarToday runs the calendar_today tool
func executeCalendarToday(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}

	result, err := tools.CalendarToday(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeCalendarListEvents runs the calendar_list_events tool
func executeCalendarListEvents(ctx context.Context, days int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"days": float64(days),
	}

	result, err := tools.CalendarListEvents(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileSend validates a file path and returns a FileAttachment for sending to the user.
// It is handled specially in processToolCalls (not routed through callToolDirect).
func executeFileSend(input json.RawMessage) (string, *router.FileAttachment) {
	var args struct {
		Path      string `json:"path"`
		MediaType string `json:"media_type"`
	}
	if err := json.Unmarshal(input, &args); err != nil {
		return fmt.Sprintf("Error parsing arguments: %v", err), nil
	}
	if args.Path == "" {
		return "Error: path is required", nil
	}

	// Expand ~ to home directory
	path := args.Path
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}

	// Verify file exists and is not a directory
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Sprintf("Error: file not found: %s", path), nil
	}
	if info.IsDir() {
		return fmt.Sprintf("Error: %s is a directory, not a file", path), nil
	}

	mediaType := args.MediaType
	if mediaType == "" {
		mediaType = "file"
	}

	logger.Info("[Agent] file_send: queued %s (%s, %d bytes)", path, mediaType, info.Size())
	return fmt.Sprintf("File queued for sending: %s (%d bytes)", filepath.Base(path), info.Size()), &router.FileAttachment{
		Path:      path,
		Name:      filepath.Base(path),
		MediaType: mediaType,
	}
}

// executeFileList runs the file_list tool
func executeFileList(ctx context.Context, path string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path": path,
	}

	result, err := tools.FileList(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileListOld runs the file_list_old tool
func executeFileListOld(ctx context.Context, path string, days int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path": path,
		"days": float64(days),
	}

	result, err := tools.FileListOld(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileTrash moves files to trash
func executeFileTrash(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	result, err := tools.FileMoveToTrash(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileRead reads a file
// sensitiveFilePatterns contains file name patterns that should never be read by the AI agent.
var sensitiveFilePatterns = []string{
	".env", "credentials", ".pem", ".key",
	"id_rsa", "id_ed25519", ".htpasswd", ".netrc",
}

// isSensitiveFile checks if a file path matches sensitive patterns.
func isSensitiveFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	for _, p := range sensitiveFilePatterns {
		if strings.Contains(base, p) {
			return true
		}
	}
	return false
}

func executeFileRead(ctx context.Context, path string) string {
	if isSensitiveFile(path) {
		return "ACCESS DENIED: reading sensitive files (.env, credentials, keys) is blocked for security. Do NOT retry."
	}

	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path": path,
	}

	result, err := tools.FileRead(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeFileWrite writes content to a file
func executeFileWrite(ctx context.Context, path string, content string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{
		"path":    path,
		"content": content,
	}

	result, err := tools.FileWrite(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}

	return extractText(result)
}

// executeShell runs the shell_execute tool
func executeShell(ctx context.Context, command string) string {
	logger.Debug("[Shell] Executing: %s", command)

	// Safety check — enhanced dangerous command detection
	if dangerous, desc := detectDangerousCommand(command); dangerous {
		logger.Warn("[Shell] Command blocked (%s): %s", desc, command)
		return fmt.Sprintf("BLOCKED: Command matched dangerous pattern (%s). Do NOT retry this command.", desc)
	}

	// Safety check - block reading sensitive files via shell
	cmdLower := strings.ToLower(command)
	for _, pat := range sensitiveFilePatterns {
		if strings.Contains(cmdLower, pat) {
			logger.Warn("[Shell] Command blocked: references sensitive file pattern '%s'", pat)
			return "ACCESS DENIED: reading sensitive files (.env, credentials, keys) is blocked for security. Do NOT retry."
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var result strings.Builder
	if stdout.Len() > 0 {
		result.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		result.WriteString("\nstderr: " + stderr.String())
	}
	if err != nil {
		result.WriteString("\nerror: " + err.Error())
	}

	output := result.String()

	// Log result at verbose level (truncate if too long)
	if len(output) > 500 {
		logger.Debug("[Shell] Output: %s... (truncated)", output[:500])
	} else {
		logger.Debug("[Shell] Output: %s", output)
	}

	return output
}

// executeOpenURL opens a URL in the default browser
func executeOpenURL(ctx context.Context, url string) string {
	if url == "" {
		return "Error: URL is required"
	}

	// Validate URL has a scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", url)
	default: // linux and others
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	}

	err := cmd.Start()
	if err != nil {
		return "Error opening URL: " + err.Error()
	}

	return "Opened " + url + " in browser"
}

// extractText extracts text content from MCP result
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			return textContent.Text
		}
	}

	return ""
}

// === CALENDAR ===

func executeCalendarCreate(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.CalendarCreateEvent(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeCalendarSearch(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.CalendarSearchEvents(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeCalendarDelete(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.CalendarDeleteEvent(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === REMINDERS ===

func executeRemindersToday(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.RemindersToday(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeRemindersAdd(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.RemindersAdd(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeRemindersComplete(ctx context.Context, title string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"title": title}
	result, err := tools.RemindersComplete(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeRemindersDelete(ctx context.Context, title string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"title": title}
	result, err := tools.RemindersDelete(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === NOTES ===

func executeNotesList(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.NotesListNotes(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeNotesRead(ctx context.Context, title string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"title": title}
	result, err := tools.NotesRead(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeNotesCreate(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.NotesCreate(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeNotesSearch(ctx context.Context, keyword string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"keyword": keyword}
	result, err := tools.NotesSearch(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === WEATHER ===

func executeWeatherCurrent(ctx context.Context, location string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"location": location}
	result, err := tools.WeatherCurrent(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeWeatherForecast(ctx context.Context, location string, days int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"location": location, "days": float64(days)}
	result, err := tools.WeatherForecast(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === WEB ===

func executeWebSearch(ctx context.Context, query string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"query": query}
	result, err := tools.WebSearch(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeWebFetch(ctx context.Context, url string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"url": url}
	result, err := tools.WebFetch(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === BROWSER AUTOMATION ===

func executeBrowserStart(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserStart(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserNavigate(ctx context.Context, url string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"url": url}
	result, err := tools.BrowserNavigate(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserSnapshot(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.BrowserSnapshot(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserClick(ctx context.Context, ref int) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"ref": float64(ref)}
	result, err := tools.BrowserClick(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserType(ctx context.Context, ref int, text string, submit bool) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"ref": float64(ref), "text": text, "submit": submit}
	result, err := tools.BrowserType(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserPress(ctx context.Context, key string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"key": key}
	result, err := tools.BrowserPress(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserExecuteJS(ctx context.Context, script string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"script": script}
	result, err := tools.BrowserExecuteJS(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserCommentZhihu(ctx context.Context, comment, replyTo string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"comment": comment, "reply_to": replyTo}
	result, err := tools.BrowserCommentZhihu(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserCommentXiaohongshu(ctx context.Context, comment string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"comment": comment}
	result, err := tools.BrowserCommentXiaohongshu(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserVisited(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserVisited(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserClickAll(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserClickAll(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserScreenshot(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserScreenshot(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserTabs(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.BrowserTabs(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserTabOpen(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserTabOpen(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserTabClose(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.BrowserTabClose(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserStatus(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.BrowserStatus(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeBrowserStop(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.BrowserStop(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === CLIPBOARD ===

func executeClipboardRead(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.ClipboardRead(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeClipboardWrite(ctx context.Context, content string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"content": content}
	result, err := tools.ClipboardWrite(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === NOTIFICATION ===

func executeNotificationSend(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.NotificationSend(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === SCREENSHOT ===

func executeScreenshot(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.ScreenshotCapture(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === MUSIC ===

func executeMusicPlay(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.MusicPlay(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicPause(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.MusicPause(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicNext(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.MusicNext(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicPrevious(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.MusicPrevious(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicNowPlaying(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.MusicNowPlaying(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicVolume(ctx context.Context, volume float64) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"volume": volume}
	result, err := tools.MusicSetVolume(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeMusicSearch(ctx context.Context, query string) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{"query": query}
	result, err := tools.MusicSearch(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === GIT ===

func executeGitStatus(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.GitStatus(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitLog(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitLog(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitDiff(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitDiff(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitBranch(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.GitBranch(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === GITHUB ===

func executeGitHubPRList(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitHubPRList(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitHubPRView(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitHubPRView(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitHubIssueList(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitHubIssueList(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitHubIssueView(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitHubIssueView(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitHubIssueCreate(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.GitHubIssueCreate(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeGitHubRepoView(ctx context.Context) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]interface{}{}
	result, err := tools.GitHubRepoView(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

// === SQLITE ===

func executeSQLiteExec(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.SQLiteExec(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}

func executeSQLiteQuery(ctx context.Context, args map[string]any) string {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := tools.SQLiteQuery(ctx, req)
	if err != nil {
		return "Error: " + err.Error()
	}
	return extractText(result)
}
