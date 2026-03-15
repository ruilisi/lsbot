// Package routing implements priority-based agent route resolution.
// It maps an incoming (platform, channelID, userID) triple to an agent ID
// by scanning Bindings from most-specific to least-specific match.
package routing

import "github.com/ruilisi/lsbot/internal/config"

// RouteResult holds the resolved agent ID and a description of which binding matched.
type RouteResult struct {
	AgentID   string // "" means no binding matched; caller falls back to default
	MatchedBy string // human-readable description for logging
}

// ResolveRoute finds the best matching binding for the given message attributes.
//
// Priority (highest to lowest):
//  1. platform + channelID + userID  — most specific
//  2. platform + userID
//  3. platform + channelID
//  4. platform only
//  5. no match → AgentID == ""
//
// Specificity is computed with power-of-two weights:
//
//	platform=1, channelID=2, userID=4
//
// so any combination with userID always outranks one without, etc.
func ResolveRoute(cfg *config.Config, platform, channelID, userID string) RouteResult {
	if len(cfg.Bindings) == 0 {
		return RouteResult{}
	}

	bestScore := -1
	bestAgentID := ""
	bestDesc := ""

	for _, b := range cfg.Bindings {
		m := b.Match
		// All non-empty fields must match.
		if m.Platform != "" && m.Platform != platform {
			continue
		}
		if m.ChannelID != "" && m.ChannelID != channelID {
			continue
		}
		if m.UserID != "" && m.UserID != userID {
			continue
		}

		// Compute specificity score.
		score := 0
		desc := ""
		if m.Platform != "" {
			score += 1
			desc += "platform=" + m.Platform
		}
		if m.ChannelID != "" {
			score += 2
			if desc != "" {
				desc += " "
			}
			desc += "channel=" + m.ChannelID
		}
		if m.UserID != "" {
			score += 4
			if desc != "" {
				desc += " "
			}
			desc += "user=" + m.UserID
		}

		if score > bestScore {
			bestScore = score
			bestAgentID = b.AgentID
			bestDesc = desc
		}
	}

	if bestScore < 0 {
		return RouteResult{}
	}
	return RouteResult{AgentID: bestAgentID, MatchedBy: bestDesc}
}
