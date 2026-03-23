package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Skill represents a skill/plugin that can be executed
type Skill struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Author      string            `json:"author,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	Triggers    []Trigger         `json:"triggers,omitempty"`
	Actions     []Action          `json:"actions"`
	Config      map[string]string `json:"config,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// Trigger defines when a skill should be activated
type Trigger struct {
	Type    TriggerType `json:"type"`
	Pattern string      `json:"pattern,omitempty"` // Regex pattern for pattern trigger
	Command string      `json:"command,omitempty"` // Command name for command trigger
}

// TriggerType defines the type of trigger
type TriggerType string

const (
	TriggerCommand  TriggerType = "command"  // Triggered by /command
	TriggerPattern  TriggerType = "pattern"  // Triggered by regex pattern match
	TriggerKeyword  TriggerType = "keyword"  // Triggered by keyword in message
	TriggerSchedule TriggerType = "schedule" // Triggered by schedule (cron)
	TriggerEvent    TriggerType = "event"    // Triggered by system event
)

// Action defines what a skill does when triggered
type Action struct {
	ID          string         `json:"id"`
	Type        ActionType     `json:"type"`
	Description string         `json:"description,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

// ActionType defines the type of action
type ActionType string

const (
	ActionShell    ActionType = "shell"    // Execute shell command
	ActionHTTP     ActionType = "http"     // Make HTTP request
	ActionPrompt   ActionType = "prompt"   // Send prompt to AI
	ActionTool     ActionType = "tool"     // Execute a registered tool
	ActionWorkflow ActionType = "workflow" // Execute a multi-step workflow
)

// ExecutionContext provides context for skill execution
type ExecutionContext struct {
	Context   context.Context
	SessionID string
	UserID    string
	Platform  string
	Message   string
	Matches   []string          // Regex capture groups
	Variables map[string]string // Variables from previous actions
}

// ExecutionResult contains the result of skill execution
type ExecutionResult struct {
	Success  bool
	Output   string
	Error    error
	Continue bool // Whether to continue processing other skills
}

// SkillExecutor executes skill actions
type SkillExecutor interface {
	Execute(ctx ExecutionContext, action Action) ExecutionResult
}

// Registry manages skills
type Registry struct {
	skills    map[string]*Skill
	executors map[ActionType]SkillExecutor
	skillDir  string
	mu        sync.RWMutex
}

// NewRegistry creates a new skill registry
func NewRegistry(skillDir string) *Registry {
	if skillDir == "" {
		home, _ := os.UserHomeDir()
		skillDir = filepath.Join(home, ".lsbot", "skills")
	}

	return &Registry{
		skills:    make(map[string]*Skill),
		executors: make(map[ActionType]SkillExecutor),
		skillDir:  skillDir,
	}
}

// RegisterExecutor registers an action executor
func (r *Registry) RegisterExecutor(actionType ActionType, executor SkillExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[actionType] = executor
}

// Register adds a skill to the registry
func (r *Registry) Register(skill *Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if skill.ID == "" {
		return fmt.Errorf("skill ID is required")
	}

	if _, exists := r.skills[skill.ID]; exists {
		return fmt.Errorf("skill already registered: %s", skill.ID)
	}

	r.skills[skill.ID] = skill
	log.Printf("[Skills] Registered skill: %s (%s)", skill.Name, skill.ID)
	return nil
}

// Unregister removes a skill from the registry
func (r *Registry) Unregister(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skillID]; !exists {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	delete(r.skills, skillID)
	log.Printf("[Skills] Unregistered skill: %s", skillID)
	return nil
}

// Get returns a skill by ID
func (r *Registry) Get(skillID string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, ok := r.skills[skillID]
	return skill, ok
}

// List returns all registered skills
func (r *Registry) List() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		skills = append(skills, skill)
	}
	return skills
}

// ListEnabled returns all enabled skills
func (r *Registry) ListEnabled() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skills := make([]*Skill, 0)
	for _, skill := range r.skills {
		if skill.Enabled {
			skills = append(skills, skill)
		}
	}
	return skills
}

// Enable enables a skill
func (r *Registry) Enable(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, ok := r.skills[skillID]
	if !ok {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	skill.Enabled = true
	return nil
}

// Disable disables a skill
func (r *Registry) Disable(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, ok := r.skills[skillID]
	if !ok {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	skill.Enabled = false
	return nil
}

// FindByTrigger finds skills that match a trigger
func (r *Registry) FindByTrigger(triggerType TriggerType, value string) []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []*Skill
	for _, skill := range r.skills {
		if !skill.Enabled {
			continue
		}

		for _, trigger := range skill.Triggers {
			if trigger.Type != triggerType {
				continue
			}

			switch triggerType {
			case TriggerCommand:
				if trigger.Command == value {
					matches = append(matches, skill)
				}
			case TriggerKeyword:
				if trigger.Pattern == value {
					matches = append(matches, skill)
				}
			}
		}
	}
	return matches
}

// FindByCommand finds skills triggered by a command
func (r *Registry) FindByCommand(command string) []*Skill {
	return r.FindByTrigger(TriggerCommand, command)
}

// Execute runs a skill's actions
func (r *Registry) Execute(ctx ExecutionContext, skill *Skill) []ExecutionResult {
	r.mu.RLock()
	executors := r.executors
	r.mu.RUnlock()

	results := make([]ExecutionResult, 0, len(skill.Actions))

	for _, action := range skill.Actions {
		executor, ok := executors[action.Type]
		if !ok {
			results = append(results, ExecutionResult{
				Success: false,
				Error:   fmt.Errorf("no executor for action type: %s", action.Type),
			})
			continue
		}

		result := executor.Execute(ctx, action)
		results = append(results, result)

		// Store output in variables for next action
		if result.Success && result.Output != "" {
			if ctx.Variables == nil {
				ctx.Variables = make(map[string]string)
			}
			ctx.Variables[action.ID] = result.Output
		}

		// Stop if action failed and continue is false
		if !result.Success && !result.Continue {
			break
		}
	}

	return results
}

// LoadFromDirectory loads skills from JSON files in a directory
func (r *Registry) LoadFromDirectory(dir string) error {
	if dir == "" {
		dir = r.skillDir
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read skill directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		if err := r.LoadFromFile(path); err != nil {
			log.Printf("[Skills] Failed to load skill from %s: %v", path, err)
		}
	}

	return nil
}

// LoadFromFile loads a skill from a JSON file
func (r *Registry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read skill file: %w", err)
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return fmt.Errorf("failed to parse skill file: %w", err)
	}

	return r.Register(&skill)
}

// SaveToFile saves a skill to a JSON file
func (r *Registry) SaveToFile(skillID string) error {
	skill, ok := r.Get(skillID)
	if !ok {
		return fmt.Errorf("skill not found: %s", skillID)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(r.skillDir, 0755); err != nil {
		return fmt.Errorf("failed to create skill directory: %w", err)
	}

	data, err := json.MarshalIndent(skill, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill: %w", err)
	}

	path := filepath.Join(r.skillDir, skillID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write skill file: %w", err)
	}

	return nil
}

// ExportSkill exports a skill to JSON
func (r *Registry) ExportSkill(skillID string) ([]byte, error) {
	skill, ok := r.Get(skillID)
	if !ok {
		return nil, fmt.Errorf("skill not found: %s", skillID)
	}

	return json.MarshalIndent(skill, "", "  ")
}

// ImportSkill imports a skill from JSON
func (r *Registry) ImportSkill(data []byte) error {
	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return fmt.Errorf("failed to parse skill: %w", err)
	}

	return r.Register(&skill)
}
