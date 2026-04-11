package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/logger"
)

// TrajectoryEntry is one complete agent turn — a user message, all tool calls
// made during that turn, and the final assistant response.  A sequence of
// entries for a single conversation forms a trajectory suitable for RL /
// supervised fine-tuning.
type TrajectoryEntry struct {
	Timestamp   time.Time    `json:"timestamp"`
	ConvKey     string       `json:"conv_key"`
	Platform    string       `json:"platform"`
	Username    string       `json:"username"`
	UserMessage string       `json:"user_message"`
	ToolCalls   []ToolCallRecord `json:"tool_calls,omitempty"`
	Response    string       `json:"response"`
	RoundCount  int          `json:"round_count"`   // how many tool-call rounds were needed
	DurationMs  int64        `json:"duration_ms"`
}

// ToolCallRecord captures a single tool invocation for the trajectory log.
type ToolCallRecord struct {
	Name   string          `json:"name"`
	Input  json.RawMessage `json:"input"`
	Output string          `json:"output"`
	ErrMsg string          `json:"error,omitempty"`
}

// trajectoryWriter serialises TrajectoryEntry values to a JSONL file.
type trajectoryWriter struct {
	mu   sync.Mutex
	path string
}

var (
	globalTrajWriter     *trajectoryWriter
	globalTrajWriterOnce sync.Once
)

// trajectoryEnabled returns true when LSBOT_SAVE_TRAJECTORIES=1 (or "true").
func trajectoryEnabled() bool {
	v := os.Getenv("LSBOT_SAVE_TRAJECTORIES")
	return v == "1" || v == "true" || v == "yes"
}

// globalTrajectoryWriter returns a lazily-initialised writer.
// Returns nil if trajectory saving is disabled.
func globalTrajectoryWriter() *trajectoryWriter {
	if !trajectoryEnabled() {
		return nil
	}
	globalTrajWriterOnce.Do(func() {
		dir := filepath.Join(config.HubDir(), "trajectories")
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Warn("[Trajectory] Could not create directory: %v", err)
			return
		}
		// One JSONL file per day.
		fname := fmt.Sprintf("trajectories_%s.jsonl", time.Now().Format("2006-01-02"))
		globalTrajWriter = &trajectoryWriter{path: filepath.Join(dir, fname)}
	})
	return globalTrajWriter
}

// Write appends entry to the JSONL file atomically.
func (w *trajectoryWriter) Write(entry TrajectoryEntry) {
	b, err := json.Marshal(entry)
	if err != nil {
		logger.Warn("[Trajectory] Marshal error: %v", err)
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	f, err := os.OpenFile(w.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Warn("[Trajectory] Open file error: %v", err)
		return
	}
	defer f.Close()

	f.Write(b)     //nolint:errcheck
	f.Write([]byte("\n")) //nolint:errcheck
}

// saveTrajectory writes a completed turn to the trajectory log if enabled.
func saveTrajectory(entry TrajectoryEntry) {
	w := globalTrajectoryWriter()
	if w == nil {
		return
	}
	w.Write(entry)
}
