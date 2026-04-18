package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/router"
)

const (
	// maxConcurrentChildren is the default cap on parallel subagents.
	maxConcurrentChildren = 3

	// delegateTimeout is the per-child wall-clock deadline.
	delegateTimeout = 5 * time.Minute
)

// DelegateTask spawns one or more child agents to run tasks in parallel.
// args shape:
//
//	{ "tasks": ["task1", "task2"], "context": "optional shared context" }
//	  — OR for a single task —
//	{ "task": "do something", "context": "optional" }
//
// Returns a JSON object with per-task results.
func (a *Agent) DelegateTask(ctx context.Context, args map[string]any) string {
	// Parse task list
	var tasks []string
	switch v := args["tasks"].(type) {
	case []any:
		for _, t := range v {
			if s, ok := t.(string); ok && strings.TrimSpace(s) != "" {
				tasks = append(tasks, s)
			}
		}
	}
	if len(tasks) == 0 {
		if single, ok := args["task"].(string); ok && strings.TrimSpace(single) != "" {
			tasks = []string{single}
		}
	}
	if len(tasks) == 0 {
		return `{"error": "provide 'task' (string) or 'tasks' (array of strings)"}`
	}

	sharedCtx, _ := args["context"].(string)

	// Cap concurrency
	concurrency := maxConcurrentChildren
	if len(tasks) < concurrency {
		concurrency = len(tasks)
	}

	type result struct {
		Task   string `json:"task"`
		Result string `json:"result"`
		Error  string `json:"error,omitempty"`
	}

	results := make([]result, len(tasks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			childCtx, cancel := context.WithTimeout(ctx, delegateTimeout)
			defer cancel()

			out, err := a.runChildAgent(childCtx, t, sharedCtx)
			results[idx] = result{Task: t}
			if err != nil {
				results[idx].Error = err.Error()
				logger.Warn("[Agent] Subagent task %d failed: %v", idx, err)
			} else {
				results[idx].Result = out
			}
		}(i, task)
	}

	wg.Wait()

	b, _ := json.Marshal(map[string]any{"results": results})
	return string(b)
}

// runChildAgent creates a fresh Agent with the same provider and runs a single
// task message. The child has no conversation history and cannot spawn further
// subagents (delegate_task is removed from its tool list).
func (a *Agent) runChildAgent(ctx context.Context, task, sharedCtx string) (string, error) {
	child := &Agent{
		provider:         a.provider,
		fallbackProviders: a.fallbackProviders,
		memory:           NewMemory(20, 30*time.Minute),
		sessions:         NewSessionStore(),
		autoApprove:      true, // children don't prompt
		maxToolRounds:    20,
		callTimeoutSecs:  a.callTimeoutSecs,
		mcpManager:       a.mcpManager,
		disableFileTools: a.disableFileTools,
		// pathChecker inherited
		pathChecker: a.pathChecker,
		// cronScheduler intentionally nil – children cannot schedule
	}

	prompt := task
	if sharedCtx != "" {
		prompt = fmt.Sprintf("Context: %s\n\nTask: %s", sharedCtx, task)
	}

	resp, err := child.HandleMessage(ctx, router.Message{
		Platform:  "delegate",
		ChannelID: "child",
		UserID:    "child",
		Username:  "child",
		Text:      prompt,
	})
	if err != nil {
		return "", err
	}
	return resp.Text, nil
}
