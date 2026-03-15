package agent

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/ruilisi/lsbot/internal/config"
	"github.com/ruilisi/lsbot/internal/logger"
	"github.com/ruilisi/lsbot/internal/router"
	"github.com/ruilisi/lsbot/internal/routing"
)

// AgentPool manages multiple agents for per-platform/channel model overrides.
// When no overrides match, it uses the default agent.
type AgentPool struct {
	defaultAgent *Agent
	baseCfg      Config
	fullCfg      *config.Config
	agents       map[string]*Agent
	mu           sync.RWMutex
}

// NewAgentPool creates a pool with a default agent and config for overrides.
func NewAgentPool(defaultAgent *Agent, baseCfg Config, fullCfg *config.Config) *AgentPool {
	return &AgentPool{
		defaultAgent: defaultAgent,
		baseCfg:      baseCfg,
		fullCfg:      fullCfg,
		agents:       make(map[string]*Agent),
	}
}

// DefaultAgent returns the default agent (for setting cron scheduler, etc.)
func (p *AgentPool) DefaultAgent() *Agent {
	return p.defaultAgent
}

// HandleMessage resolves the right agent for the message and delegates.
func (p *AgentPool) HandleMessage(ctx context.Context, msg router.Message) (router.Response, error) {
	if p.fullCfg == nil {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	// --- NEW PATH: agents[] + bindings[] system ---
	if len(p.fullCfg.Agents) > 0 {
		return p.handleWithAgentRouting(ctx, msg)
	}

	// --- LEGACY PATH: named providers or ai.overrides ---
	if len(p.fullCfg.Providers) > 0 {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}
	if len(p.fullCfg.AI.Overrides) == 0 {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	platform := msg.Platform
	if ap, ok := msg.Metadata["actual_platform"]; ok && ap != "" {
		platform = ap
	}
	resolved := p.fullCfg.ResolveAI(platform, msg.ChannelID)
	if resolved.Provider == p.fullCfg.AI.Provider &&
		resolved.APIKey == p.fullCfg.AI.APIKey &&
		resolved.Model == p.fullCfg.AI.Model {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	a := p.getOrCreate(resolved)
	if a == nil {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}
	return a.HandleMessage(ctx, msg)
}

// handleWithAgentRouting uses the routing package to pick a named agent.
func (p *AgentPool) handleWithAgentRouting(ctx context.Context, msg router.Message) (router.Response, error) {
	platform := msg.Platform
	if ap, ok := msg.Metadata["actual_platform"]; ok && ap != "" {
		platform = ap
	}

	result := routing.ResolveRoute(p.fullCfg, platform, msg.ChannelID, msg.UserID)

	agentID := result.AgentID
	if agentID == "" {
		agentID = p.fullCfg.DefaultAgentID()
	}

	if agentID == "" {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	entry, found := p.fullCfg.FindAgent(agentID)
	if !found {
		logger.Warn("[AgentPool] Binding references unknown agent %q, using default", agentID)
		return p.defaultAgent.HandleMessage(ctx, msg)
	}

	if result.MatchedBy != "" {
		logger.Info("[AgentPool] Routing to agent %q (matched: %s)", agentID, result.MatchedBy)
	} else {
		logger.Info("[AgentPool] Routing to default agent %q", agentID)
	}

	a := p.getOrCreateByID(agentID, entry)
	if a == nil {
		return p.defaultAgent.HandleMessage(ctx, msg)
	}
	return a.HandleMessage(ctx, msg)
}

// getOrCreateByID looks up or lazily creates an agent by its named ID.
func (p *AgentPool) getOrCreateByID(id string, entry config.AgentEntry) *Agent {
	p.mu.RLock()
	if a, ok := p.agents[id]; ok {
		p.mu.RUnlock()
		return a
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if a, ok := p.agents[id]; ok {
		return a
	}

	aiCfg := p.fullCfg.ResolveAgentAI(entry)

	// Resolve instructions: treat as file path when it starts with / or .
	instructions := entry.Instructions
	if len(instructions) > 0 && (instructions[0] == '/' || instructions[0] == '.') {
		if data, err := os.ReadFile(instructions); err == nil {
			instructions = string(data)
		} else {
			logger.Warn("[AgentPool] Could not read instructions file %q for agent %q: %v", instructions, id, err)
		}
	}

	cfg := p.baseCfg
	cfg.Provider = aiCfg.Provider
	cfg.APIKey = aiCfg.APIKey
	cfg.BaseURL = aiCfg.BaseURL
	cfg.Model = aiCfg.Model
	if instructions != "" {
		cfg.CustomInstructions = instructions
	}
	if len(entry.AllowTools) > 0 {
		cfg.AllowTools = entry.AllowTools
	}
	if len(entry.DenyTools) > 0 {
		cfg.DenyTools = entry.DenyTools
	}
	if entry.Workspace != "" {
		cfg.Workspace = entry.Workspace
	}

	a, err := New(cfg)
	if err != nil {
		logger.Error("[AgentPool] Failed to create agent %q: %v", id, err)
		return nil
	}

	if p.defaultAgent.cronScheduler != nil {
		a.SetCronScheduler(p.defaultAgent.cronScheduler)
	}

	name := entry.Name
	if name == "" {
		name = id
	}
	logger.Info("[AgentPool] Created agent %q (provider=%s model=%s)", name, aiCfg.Provider, aiCfg.Model)
	p.agents[id] = a
	return a
}

// getOrCreate is the legacy path keyed by provider+key+model string.
func (p *AgentPool) getOrCreate(aiCfg config.AIConfig) *Agent {
	key := fmt.Sprintf("%s:%s:%s", aiCfg.Provider, aiCfg.APIKey, aiCfg.Model)

	p.mu.RLock()
	if a, ok := p.agents[key]; ok {
		p.mu.RUnlock()
		return a
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	if a, ok := p.agents[key]; ok {
		return a
	}

	cfg := p.baseCfg
	cfg.Provider = aiCfg.Provider
	cfg.APIKey = aiCfg.APIKey
	cfg.BaseURL = aiCfg.BaseURL
	cfg.Model = aiCfg.Model

	a, err := New(cfg)
	if err != nil {
		logger.Error("[AgentPool] Failed to create agent for %s/%s: %v", aiCfg.Provider, aiCfg.Model, err)
		return nil
	}

	if p.defaultAgent.cronScheduler != nil {
		a.SetCronScheduler(p.defaultAgent.cronScheduler)
	}

	logger.Info("[AgentPool] Created agent for provider=%s model=%s", aiCfg.Provider, aiCfg.Model)
	p.agents[key] = a
	return a
}
