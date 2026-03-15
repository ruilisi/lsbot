package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	cronpkg "github.com/ruilisi/lsbot/internal/cron"
)

var cronScheduler *cronpkg.Scheduler

// SetCronScheduler sets the global cron scheduler instance
func SetCronScheduler(scheduler *cronpkg.Scheduler) {
	cronScheduler = scheduler
}

// CronCreate creates a new scheduled job
func CronCreate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cronScheduler == nil {
		return mcp.NewToolResultError("cron scheduler not initialized"), nil
	}

	// Extract parameters
	name, ok := req.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	schedule, ok := req.Params.Arguments["schedule"].(string)
	if !ok || schedule == "" {
		return mcp.NewToolResultError("schedule is required"), nil
	}

	tool, ok := req.Params.Arguments["tool"].(string)
	if !ok || tool == "" {
		return mcp.NewToolResultError("tool is required"), nil
	}

	// Arguments are optional
	arguments, _ := req.Params.Arguments["arguments"].(map[string]any)

	// Create job
	job, err := cronScheduler.AddJob(name, schedule, tool, arguments)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create job: %v", err)), nil
	}

	result := fmt.Sprintf("✓ Job created successfully\n\nID: %s\nName: %s\nSchedule: %s\nTool: %s\nStatus: enabled",
		job.ID, job.Name, job.Schedule, job.Tool)

	return mcp.NewToolResultText(result), nil
}

// CronList lists all scheduled jobs
func CronList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cronScheduler == nil {
		return mcp.NewToolResultError("cron scheduler not initialized"), nil
	}

	jobs := cronScheduler.ListJobs()

	if len(jobs) == 0 {
		return mcp.NewToolResultText("No scheduled jobs"), nil
	}

	result := fmt.Sprintf("Scheduled Jobs (%d total)\n\n", len(jobs))

	for i, job := range jobs {
		status := "enabled"
		if !job.Enabled {
			status = "paused"
		}

		lastRun := "never"
		if job.LastRun != nil {
			lastRun = job.LastRun.Format("2006-01-02 15:04:05")
		}

		lastError := "-"
		if job.LastError != "" {
			lastError = job.LastError
		}

		result += fmt.Sprintf("%d. %s (ID: %s)\n", i+1, job.Name, job.ID)
		result += fmt.Sprintf("   Schedule: %s\n", job.Schedule)
		result += fmt.Sprintf("   Tool: %s\n", job.Tool)
		result += fmt.Sprintf("   Status: %s\n", status)
		result += fmt.Sprintf("   Last Run: %s\n", lastRun)
		result += fmt.Sprintf("   Last Error: %s\n", lastError)
		result += "\n"
	}

	return mcp.NewToolResultText(result), nil
}

// CronDelete deletes a scheduled job
func CronDelete(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cronScheduler == nil {
		return mcp.NewToolResultError("cron scheduler not initialized"), nil
	}

	id, ok := req.Params.Arguments["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	if err := cronScheduler.RemoveJob(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete job: %v", err)), nil
	}

	result := fmt.Sprintf("✓ Job deleted: %s", id)
	return mcp.NewToolResultText(result), nil
}

// CronPause pauses a scheduled job
func CronPause(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cronScheduler == nil {
		return mcp.NewToolResultError("cron scheduler not initialized"), nil
	}

	id, ok := req.Params.Arguments["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	if err := cronScheduler.PauseJob(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to pause job: %v", err)), nil
	}

	result := fmt.Sprintf("✓ Job paused: %s", id)
	return mcp.NewToolResultText(result), nil
}

// CronResume resumes a paused job
func CronResume(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if cronScheduler == nil {
		return mcp.NewToolResultError("cron scheduler not initialized"), nil
	}

	id, ok := req.Params.Arguments["id"].(string)
	if !ok || id == "" {
		return mcp.NewToolResultError("id is required"), nil
	}

	if err := cronScheduler.ResumeJob(id); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to resume job: %v", err)), nil
	}

	result := fmt.Sprintf("✓ Job resumed: %s", id)
	return mcp.NewToolResultText(result), nil
}

func registerCronTools(s *Server) {
	// cron_create
	s.addTool(mcp.NewTool("cron_create",
		mcp.WithDescription("Create a scheduled job that runs periodically"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Human-readable name for the job")),
		mcp.WithString("schedule", mcp.Required(), mcp.Description("Cron expression (e.g., '0 * * * *' for every hour)")),
		mcp.WithString("tool", mcp.Required(), mcp.Description("MCP tool to execute")),
		mcp.WithObject("arguments", mcp.Description("Arguments to pass to the tool")),
	), CronCreate)

	// cron_list
	s.addTool(mcp.NewTool("cron_list",
		mcp.WithDescription("List all scheduled jobs"),
	), CronList)

	// cron_delete
	s.addTool(mcp.NewTool("cron_delete",
		mcp.WithDescription("Delete a scheduled job"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID to delete")),
	), CronDelete)

	// cron_pause
	s.addTool(mcp.NewTool("cron_pause",
		mcp.WithDescription("Pause a scheduled job"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID to pause")),
	), CronPause)

	// cron_resume
	s.addTool(mcp.NewTool("cron_resume",
		mcp.WithDescription("Resume a paused job"),
		mcp.WithString("id", mcp.Required(), mcp.Description("Job ID to resume")),
	), CronResume)
}
