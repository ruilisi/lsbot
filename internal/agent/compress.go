package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ruilisi/lsbot/internal/logger"
)

const (
	// compressTokenThreshold triggers compression when estimated tokens exceed this.
	// Roughly 80 k tokens at ~4 chars/token ≈ 320 000 chars.
	compressTokenThreshold = 80_000

	// compressKeepTail is the number of recent message pairs to keep verbatim
	// after compression. These stay in full so the AI has immediate context.
	compressKeepTail = 6

	// estimateDivisor converts character count to an approximate token count.
	estimateDivisor = 4
)

// estimateTokens returns a rough token count for a slice of messages.
func estimateTokens(msgs []Message) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Content)
	}
	return total / estimateDivisor
}

// compressMessages summarises the older portion of the conversation using the
// agent's own provider, then returns a condensed message list consisting of:
//   - one synthetic assistant message containing the summary
//   - the compressKeepTail most-recent messages unchanged
//
// If the provider call fails the original messages are returned unmodified so
// the agent can continue without interruption.
func (a *Agent) compressMessages(ctx context.Context, msgs []Message) []Message {
	if len(msgs) <= compressKeepTail*2 {
		return msgs
	}

	cutoff := len(msgs) - compressKeepTail
	older := msgs[:cutoff]
	recent := msgs[cutoff:]

	// Build a compact representation of the older messages for summarisation.
	var sb strings.Builder
	for _, m := range older {
		sb.WriteString(m.Role)
		sb.WriteString(": ")
		preview := m.Content
		if len(preview) > 800 {
			preview = preview[:800] + "…"
		}
		sb.WriteString(preview)
		sb.WriteString("\n\n")
	}

	summaryPrompt := fmt.Sprintf(
		"The following is the earlier portion of a conversation. Summarise it concisely in third-person, "+
			"preserving key facts, decisions, and user preferences. Output only the summary, no preamble.\n\n%s",
		sb.String())

	resp, err := a.chatWithFallback(ctx, ChatRequest{
		Messages:     []Message{{Role: "user", Content: summaryPrompt}},
		SystemPrompt: "You are a helpful assistant that summarises conversation history.",
		MaxTokens:    1024,
	})
	if err != nil {
		logger.Warn("[Agent] Context compression failed, keeping full history: %v", err)
		return msgs
	}

	summary := Message{
		Role:    "assistant",
		Content: "[Earlier conversation summary]\n" + resp.Content,
	}

	compressed := make([]Message, 0, 1+len(recent))
	compressed = append(compressed, summary)
	compressed = append(compressed, recent...)

	logger.Info("[Agent] Compressed %d messages → 1 summary + %d recent", len(older), len(recent))
	return compressed
}
