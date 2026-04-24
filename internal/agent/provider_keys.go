package agent

import "os"

// providerEnvVars maps canonical provider names to the environment variables
// that conventionally hold their API keys.  Used by MixtureOfAgents to
// instantiate providers on the fly without requiring the caller to supply keys.
var providerEnvVars = map[string][]string{
	"claude":      {"ANTHROPIC_API_KEY"},
	"anthropic":   {"ANTHROPIC_API_KEY"},
	"openai":      {"OPENAI_API_KEY"},
	"deepseek":    {"DEEPSEEK_API_KEY"},
	"gemini":      {"GEMINI_API_KEY", "GOOGLE_API_KEY"},
	"kimi":        {"KIMI_API_KEY", "MOONSHOT_API_KEY"},
	"moonshot":    {"MOONSHOT_API_KEY", "KIMI_API_KEY"},
	"qwen":        {"QWEN_API_KEY", "DASHSCOPE_API_KEY"},
	"qianwen":     {"QWEN_API_KEY", "DASHSCOPE_API_KEY"},
	"tongyi":      {"QWEN_API_KEY", "DASHSCOPE_API_KEY"},
	"zhipu":       {"ZHIPU_API_KEY"},
	"minimax":     {"MINIMAX_API_KEY"},
	"doubao":      {"DOUBAO_API_KEY"},
	"yi":          {"YI_API_KEY", "LINGYIWANWU_API_KEY"},
	"stepfun":     {"STEPFUN_API_KEY"},
	"siliconflow": {"SILICONFLOW_API_KEY"},
	"grok":        {"XAI_API_KEY", "GROK_API_KEY"},
	"baichuan":    {"BAICHUAN_API_KEY"},
	"spark":       {"SPARK_API_KEY"},
	"hunyuan":     {"HUNYUAN_API_KEY"},
}

// resolveProviderAPIKey returns the first non-empty value found in the
// environment variables associated with the given provider name.
// Returns "" if none are set (provider may still work for local endpoints).
func resolveProviderAPIKey(providerName string) string {
	vars, ok := providerEnvVars[providerName]
	if !ok {
		// Fall back to a generic pattern: <PROVIDER>_API_KEY
		v := os.Getenv(providerName + "_API_KEY")
		return v
	}
	for _, envVar := range vars {
		if v := os.Getenv(envVar); v != "" {
			return v
		}
	}
	return ""
}
