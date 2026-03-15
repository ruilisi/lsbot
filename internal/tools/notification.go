package tools

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/mark3labs/mcp-go/mcp"
)

// NotificationSend sends a system notification
func NotificationSend(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title, ok := req.Params.Arguments["title"].(string)
	if !ok || title == "" {
		return mcp.NewToolResultError("title is required"), nil
	}

	message := ""
	if m, ok := req.Params.Arguments["message"].(string); ok {
		message = m
	}

	subtitle := ""
	if s, ok := req.Params.Arguments["subtitle"].(string); ok {
		subtitle = s
	}

	sound := true
	if s, ok := req.Params.Arguments["sound"].(bool); ok {
		sound = s
	}

	switch runtime.GOOS {
	case "darwin":
		return notifyMacOS(ctx, title, message, subtitle, sound)
	case "linux":
		return notifyLinux(ctx, title, message)
	case "windows":
		return notifyWindows(ctx, title, message)
	default:
		return mcp.NewToolResultError(fmt.Sprintf("notifications not supported on %s", runtime.GOOS)), nil
	}
}

func notifyMacOS(ctx context.Context, title, message, subtitle string, sound bool) (*mcp.CallToolResult, error) {
	script := fmt.Sprintf(`display notification "%s"`, escapeAppleScript(message))
	script += fmt.Sprintf(` with title "%s"`, escapeAppleScript(title))
	if subtitle != "" {
		script += fmt.Sprintf(` subtitle "%s"`, escapeAppleScript(subtitle))
	}
	if sound {
		script += ` sound name "default"`
	}

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send notification: %v - %s", err, output)), nil
	}

	return mcp.NewToolResultText("Notification sent"), nil
}

func notifyLinux(ctx context.Context, title, message string) (*mcp.CallToolResult, error) {
	cmd := exec.CommandContext(ctx, "notify-send", title, message)
	if err := cmd.Run(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to send notification: %v", err)), nil
	}
	return mcp.NewToolResultText("Notification sent"), nil
}

func notifyWindows(ctx context.Context, title, message string) (*mcp.CallToolResult, error) {
	// Use PowerShell to create a balloon notification
	script := fmt.Sprintf(`
		[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
		$template = [Windows.UI.Notifications.ToastTemplateType]::ToastText02
		$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent($template)
		$xml.GetElementsByTagName("text")[0].AppendChild($xml.CreateTextNode("%s")) | Out-Null
		$xml.GetElementsByTagName("text")[1].AppendChild($xml.CreateTextNode("%s")) | Out-Null
		$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
		[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("lsbot").Show($toast)
	`, title, message)

	cmd := exec.CommandContext(ctx, "powershell", "-command", script)
	if err := cmd.Run(); err != nil {
		// Fallback to msg command
		cmd = exec.CommandContext(ctx, "msg", "*", fmt.Sprintf("%s: %s", title, message))
		if err := cmd.Run(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to send notification: %v", err)), nil
		}
	}
	return mcp.NewToolResultText("Notification sent"), nil
}
