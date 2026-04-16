package termui

import (
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Skin holds all visual customisation for the CLI.
// Missing fields fall back to the active skin's defaults, which in turn
// fall back to the built-in "default" skin.
type Skin struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	// Spinner faces shown while waiting for the AI response.
	SpinnerFrames []string         `yaml:"spinner_frames"`
	// ThinkingVerbs are interpolated into the spinner line: "thinking…", "reasoning…"
	ThinkingVerbs []string         `yaml:"thinking_verbs"`
	// ToolPrefix is printed before each tool-result line, e.g. "┊ " or "▏ ".
	ToolPrefix    string           `yaml:"tool_prefix"`
	// Colors maps semantic names to ANSI escape codes.
	// Recognised keys: banner, response_border, tool_name, prompt_symbol.
	Colors        map[string]string `yaml:"colors"`
	// Branding overrides the agent name and prompt symbol shown in the terminal.
	Branding struct {
		AgentName     string `yaml:"agent_name"`
		PromptSymbol  string `yaml:"prompt_symbol"`
		WelcomeMsg    string `yaml:"welcome_msg"`
	} `yaml:"branding"`
}

// builtinSkins are the skins shipped with lsbot.
// Users can override any field via ~/.lsbot/skins/<name>.yaml.
var builtinSkins = map[string]*Skin{
	"default": {
		Name:          "default",
		Description:   "Classic lsbot — cyan accents, minimal chrome",
		SpinnerFrames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		ThinkingVerbs: []string{"thinking", "reasoning", "working", "processing"},
		ToolPrefix:    "┊ ",
		Colors: map[string]string{
			"banner":          Cyan,
			"response_border": BrightCyan,
			"tool_name":       Gray,
			"prompt_symbol":   Green,
		},
		Branding: struct {
			AgentName    string `yaml:"agent_name"`
			PromptSymbol string `yaml:"prompt_symbol"`
			WelcomeMsg   string `yaml:"welcome_msg"`
		}{
			AgentName:    "lsbot",
			PromptSymbol: "❯",
			WelcomeMsg:   "Hello! How can I help you today?",
		},
	},
	"mono": {
		Name:          "mono",
		Description:   "Monochrome — no colour, clean for light terminals",
		SpinnerFrames: []string{"|", "/", "-", "\\"},
		ThinkingVerbs: []string{"working"},
		ToolPrefix:    "| ",
		Colors: map[string]string{
			"banner":          Bold,
			"response_border": Bold,
			"tool_name":       Reset,
			"prompt_symbol":   Bold,
		},
		Branding: struct {
			AgentName    string `yaml:"agent_name"`
			PromptSymbol string `yaml:"prompt_symbol"`
			WelcomeMsg   string `yaml:"welcome_msg"`
		}{
			AgentName:    "lsbot",
			PromptSymbol: ">",
			WelcomeMsg:   "Ready.",
		},
	},
	"hacker": {
		Name:          "hacker",
		Description:   "Green-on-black matrix aesthetic",
		SpinnerFrames: []string{"▓", "▒", "░", "▒"},
		ThinkingVerbs: []string{"hacking", "decrypting", "compiling", "executing"},
		ToolPrefix:    "» ",
		Colors: map[string]string{
			"banner":          Green,
			"response_border": Green,
			"tool_name":       Green,
			"prompt_symbol":   Green,
		},
		Branding: struct {
			AgentName    string `yaml:"agent_name"`
			PromptSymbol string `yaml:"prompt_symbol"`
			WelcomeMsg   string `yaml:"welcome_msg"`
		}{
			AgentName:    "h4x0r",
			PromptSymbol: "$",
			WelcomeMsg:   "Access granted.",
		},
	},
	"warm": {
		Name:          "warm",
		Description:   "Yellow/amber — warm terminal feel",
		SpinnerFrames: []string{"◐", "◓", "◑", "◒"},
		ThinkingVerbs: []string{"pondering", "mulling", "considering"},
		ToolPrefix:    "• ",
		Colors: map[string]string{
			"banner":          Yellow,
			"response_border": Yellow,
			"tool_name":       Gray,
			"prompt_symbol":   Yellow,
		},
		Branding: struct {
			AgentName    string `yaml:"agent_name"`
			PromptSymbol string `yaml:"prompt_symbol"`
			WelcomeMsg   string `yaml:"welcome_msg"`
		}{
			AgentName:    "lsbot",
			PromptSymbol: "→",
			WelcomeMsg:   "What's on your mind?",
		},
	},
}

var (
	activeSkin     *Skin
	activeSkinOnce sync.Once
	activeSkinMu   sync.RWMutex
)

// skinsDir returns the user skins directory.
func skinsDir(hubDir string) string {
	return filepath.Join(hubDir, "skins")
}

// LoadSkin loads a skin by name: user skins (~/.lsbot/skins/<name>.yaml)
// take precedence over built-ins. Falls back to "default" if not found.
func LoadSkin(hubDir, name string) *Skin {
	if name == "" {
		name = "default"
	}

	// Try user skin file first
	userPath := filepath.Join(skinsDir(hubDir), name+".yaml")
	if data, err := os.ReadFile(userPath); err == nil {
		var s Skin
		if yaml.Unmarshal(data, &s) == nil {
			merged := mergeSkin(&s)
			return merged
		}
	}

	// Built-in skin
	if s, ok := builtinSkins[name]; ok {
		return s
	}

	// Unknown — return default
	return builtinSkins["default"]
}

// mergeSkin fills missing fields in a user skin from the default skin.
func mergeSkin(s *Skin) *Skin {
	d := builtinSkins["default"]
	out := *d // start with a copy of defaults

	if s.Name != "" {
		out.Name = s.Name
	}
	if s.Description != "" {
		out.Description = s.Description
	}
	if len(s.SpinnerFrames) > 0 {
		out.SpinnerFrames = s.SpinnerFrames
	}
	if len(s.ThinkingVerbs) > 0 {
		out.ThinkingVerbs = s.ThinkingVerbs
	}
	if s.ToolPrefix != "" {
		out.ToolPrefix = s.ToolPrefix
	}
	if s.Branding.AgentName != "" {
		out.Branding.AgentName = s.Branding.AgentName
	}
	if s.Branding.PromptSymbol != "" {
		out.Branding.PromptSymbol = s.Branding.PromptSymbol
	}
	if s.Branding.WelcomeMsg != "" {
		out.Branding.WelcomeMsg = s.Branding.WelcomeMsg
	}
	if len(s.Colors) > 0 {
		if out.Colors == nil {
			out.Colors = map[string]string{}
		}
		for k, v := range s.Colors {
			out.Colors[k] = v
		}
	}
	return &out
}

// SetActiveSkin switches the active skin at runtime.
func SetActiveSkin(hubDir, name string) {
	s := LoadSkin(hubDir, name)
	activeSkinMu.Lock()
	activeSkin = s
	activeSkinMu.Unlock()
}

// ActiveSkin returns the currently active skin, initialising to "default" if needed.
func ActiveSkin(hubDir string) *Skin {
	activeSkinOnce.Do(func() {
		skinName := os.Getenv("LSBOT_SKIN")
		if skinName == "" {
			skinName = "default"
		}
		activeSkin = LoadSkin(hubDir, skinName)
	})
	activeSkinMu.RLock()
	defer activeSkinMu.RUnlock()
	return activeSkin
}

// ListBuiltinSkins returns names and descriptions of all built-in skins.
func ListBuiltinSkins() []struct{ Name, Description string } {
	out := make([]struct{ Name, Description string }, 0, len(builtinSkins))
	for _, s := range builtinSkins {
		out = append(out, struct{ Name, Description string }{s.Name, s.Description})
	}
	return out
}
