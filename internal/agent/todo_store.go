package agent

import (
	"encoding/json"
	"strings"
	"sync"
)

// validStatuses is the set of allowed todo item statuses.
var validStatuses = map[string]bool{
	"pending":   true,
	"in_progress": true,
	"completed": true,
	"cancelled": true,
}

// TodoItem is a single task list entry.
type TodoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

// TodoStore is an in-memory per-conversation task list.
type TodoStore struct {
	mu    sync.Mutex
	items []TodoItem
}

// Write replaces or merges todos. Returns the full list.
func (s *TodoStore) Write(todos []TodoItem, merge bool) []TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !merge {
		s.items = s.validate(s.dedupe(todos))
		return s.copy()
	}

	// Merge: update by id, append new ones.
	existing := make(map[string]*TodoItem, len(s.items))
	order := make([]string, 0, len(s.items))
	for i := range s.items {
		existing[s.items[i].ID] = &s.items[i]
		order = append(order, s.items[i].ID)
	}
	for _, t := range s.dedupe(todos) {
		t = s.validateOne(t)
		if ptr, ok := existing[t.ID]; ok {
			if t.Content != "" {
				ptr.Content = t.Content
			}
			if t.Status != "" {
				ptr.Status = t.Status
			}
		} else {
			cp := t
			existing[t.ID] = &cp
			order = append(order, t.ID)
		}
	}
	rebuilt := make([]TodoItem, 0, len(order))
	seen := map[string]bool{}
	for _, id := range order {
		if !seen[id] {
			rebuilt = append(rebuilt, *existing[id])
			seen[id] = true
		}
	}
	s.items = rebuilt
	return s.copy()
}

// Read returns a copy of the current list.
func (s *TodoStore) Read() []TodoItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.copy()
}

// FormatForInjection returns a human-readable summary for post-compression injection, or "".
func (s *TodoStore) FormatForInjection() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	markers := map[string]string{
		"pending":     "[ ]",
		"in_progress": "[>]",
		"completed":   "[x]",
		"cancelled":   "[~]",
	}
	var active []TodoItem
	for _, item := range s.items {
		if item.Status == "pending" || item.Status == "in_progress" {
			active = append(active, item)
		}
	}
	if len(active) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("[Your active task list was preserved across context compression]\n")
	for _, item := range active {
		m := markers[item.Status]
		sb.WriteString("- " + m + " " + item.ID + ". " + item.Content + " (" + item.Status + ")\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// --- helpers ----------------------------------------------------------------

func (s *TodoStore) copy() []TodoItem {
	out := make([]TodoItem, len(s.items))
	copy(out, s.items)
	return out
}

func (s *TodoStore) dedupe(items []TodoItem) []TodoItem {
	lastIdx := map[string]int{}
	for i, t := range items {
		if t.ID == "" {
			t.ID = "?"
		}
		lastIdx[t.ID] = i
	}
	// Preserve order of last occurrence.
	seen := map[string]bool{}
	out := make([]TodoItem, 0, len(items))
	for i := len(items) - 1; i >= 0; i-- {
		t := items[i]
		if t.ID == "" {
			t.ID = "?"
		}
		if lastIdx[t.ID] == i && !seen[t.ID] {
			out = append([]TodoItem{t}, out...)
			seen[t.ID] = true
		}
	}
	return out
}

func (s *TodoStore) validate(items []TodoItem) []TodoItem {
	out := make([]TodoItem, len(items))
	for i, t := range items {
		out[i] = s.validateOne(t)
	}
	return out
}

func (s *TodoStore) validateOne(t TodoItem) TodoItem {
	if t.ID == "" {
		t.ID = "?"
	}
	if t.Content == "" {
		t.Content = "(no description)"
	}
	t.Status = strings.ToLower(strings.TrimSpace(t.Status))
	if !validStatuses[t.Status] {
		t.Status = "pending"
	}
	return t
}

// ============================================================
// Per-conversation store registry
// ============================================================

// TodoRegistry maps convKey → *TodoStore.
type TodoRegistry struct {
	mu     sync.Mutex
	stores map[string]*TodoStore
}

func newTodoRegistry() *TodoRegistry {
	return &TodoRegistry{stores: make(map[string]*TodoStore)}
}

func (r *TodoRegistry) Get(convKey string) *TodoStore {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.stores[convKey]; ok {
		return s
	}
	s := &TodoStore{}
	r.stores[convKey] = s
	return s
}

// ============================================================
// Tool handler (called from agent.go dispatchTool)
// ============================================================

// HandleTodoTool processes a "todo" tool call.
func HandleTodoTool(store *TodoStore, args map[string]any) string {
	todosRaw, hasTodos := args["todos"]
	merge, _ := args["merge"].(bool)

	var items []TodoItem
	if hasTodos && todosRaw != nil {
		// todosRaw arrives as []interface{} from JSON unmarshal.
		b, err := json.Marshal(todosRaw)
		if err != nil {
			return `{"error":"failed to parse todos: ` + err.Error() + `"}`
		}
		if err := json.Unmarshal(b, &items); err != nil {
			return `{"error":"failed to parse todos: ` + err.Error() + `"}`
		}
		items = store.Write(items, merge)
	} else {
		items = store.Read()
	}

	// Build summary counts.
	counts := map[string]int{"pending": 0, "in_progress": 0, "completed": 0, "cancelled": 0}
	for _, item := range items {
		counts[item.Status]++
	}

	result := map[string]any{
		"todos": items,
		"summary": map[string]any{
			"total":       len(items),
			"pending":     counts["pending"],
			"in_progress": counts["in_progress"],
			"completed":   counts["completed"],
			"cancelled":   counts["cancelled"],
		},
	}
	b, _ := json.Marshal(result)
	return string(b)
}
