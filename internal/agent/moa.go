package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/ruilisi/lsbot/internal/logger"
)

// MixtureOfAgents sends the same prompt to multiple providers concurrently,
// then synthesises their responses into a single answer using the primary provider.
//
// This is the mixture-of-agents (MoA) pattern: individual model responses are
// aggregated by a "proposer" round and then synthesised by a "finaliser" (the
// primary provider).  The result is typically higher quality than any individual
// model for complex or subjective questions.
//
// Tool: mixture_of_agents
//
//	args: { "prompt": "...", "providers": ["openai:gpt-4o", "deepseek", ...] }
//	      providers may be any provider name understood by createProvider.
//	      If omitted, the agent's configured fallback providers are used.
func (a *Agent) MixtureOfAgents(ctx context.Context, args map[string]any) string {
	prompt, _ := args["prompt"].(string)
	if strings.TrimSpace(prompt) == "" {
		return `{"error": "prompt is required"}`
	}

	// Resolve provider list
	var providers []Provider
	if rawList, ok := args["providers"].([]any); ok && len(rawList) > 0 {
		for _, v := range rawList {
			name, _ := v.(string)
			if name == "" {
				continue
			}
			// Parse "provider:model" shorthand
			providerName, model, _ := strings.Cut(name, ":")
			p, err := createProvider(Config{
				Provider: providerName,
				Model:    model,
				APIKey:   resolveProviderAPIKey(providerName),
			})
			if err != nil {
				logger.Warn("[MoA] Could not create provider %q: %v", name, err)
				continue
			}
			providers = append(providers, p)
		}
	}
	// Fall back to the agent's own fallback providers
	if len(providers) == 0 {
		providers = a.fallbackProviders
	}
	// Always include the primary provider
	providers = append([]Provider{a.provider}, providers...)
	if len(providers) < 2 {
		return `{"error": "mixture_of_agents requires at least 2 providers; configure fallbacks or pass a 'providers' list"}`
	}

	// --- Proposer round: ask all providers concurrently ---
	type draft struct {
		provider string
		response string
		err      error
	}
	drafts := make([]draft, len(providers))
	var wg sync.WaitGroup

	for i, p := range providers {
		wg.Add(1)
		go func(idx int, prov Provider) {
			defer wg.Done()
			resp, err := prov.Chat(ctx, ChatRequest{
				Messages:  []Message{{Role: "user", Content: prompt}},
				MaxTokens: 2048,
			})
			drafts[idx] = draft{provider: prov.Name(), response: resp.Content, err: err}
		}(i, p)
	}
	wg.Wait()

	// Collect successful drafts
	var sb strings.Builder
	successCount := 0
	for _, d := range drafts {
		if d.err != nil {
			logger.Warn("[MoA] Provider %s error: %v", d.provider, d.err)
			continue
		}
		sb.WriteString(fmt.Sprintf("=== %s ===\n%s\n\n", d.provider, d.response))
		successCount++
	}

	if successCount == 0 {
		return `{"error": "all providers failed to respond"}`
	}
	if successCount == 1 {
		// Only one response — no synthesis needed, return as-is
		for _, d := range drafts {
			if d.err == nil {
				out, _ := json.Marshal(map[string]string{"result": d.response, "note": "only one provider succeeded"})
				return string(out)
			}
		}
	}

	// --- Finaliser round: synthesise with primary provider ---
	synthesisPrompt := fmt.Sprintf(`You asked multiple AI models the following question:

<question>
%s
</question>

Here are their individual responses:

%s

Your task: synthesise these responses into a single, comprehensive, accurate answer.
- Merge complementary insights
- Resolve any contradictions by reasoning through them
- Discard hallucinations or obvious errors
- Do NOT mention the individual models or that this is a synthesis; just give the best answer.`, prompt, sb.String())

	final, err := a.provider.Chat(ctx, ChatRequest{
		Messages:  []Message{{Role: "user", Content: synthesisPrompt}},
		MaxTokens: 4096,
	})
	if err != nil {
		// Return the raw drafts if synthesis fails
		out, _ := json.Marshal(map[string]any{
			"drafts": sb.String(),
			"error":  "synthesis failed: " + err.Error(),
		})
		return string(out)
	}

	out, _ := json.Marshal(map[string]string{"result": final.Content})
	return string(out)
}
