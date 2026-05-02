package llm

// File is the top-level structure of warren's LLM TOML config.
type File struct {
	Agent     *AgentConfig     `toml:"agent"`
	LLMs      []LLMConfig      `toml:"llm"`
	Embedding *EmbeddingConfig `toml:"embedding"`
}

// AgentConfig assigns roles to LLM ids defined in [[llm]].
type AgentConfig struct {
	Main string   `toml:"main"`
	Task []string `toml:"task"`
}

// LLMConfig is a single LLM definition.
type LLMConfig struct {
	ID          string `toml:"id"`
	Description string `toml:"description"`
	Provider    string `toml:"provider"`
	Model       string `toml:"model"`

	Claude *ClaudeOptions `toml:"claude"`
	Gemini *GeminiOptions `toml:"gemini"`
	OpenAI *OpenAIOptions `toml:"openai"`
}

// ClaudeOptions holds Claude-specific settings. Vertex and api_key modes are
// mutually exclusive.
type ClaudeOptions struct {
	ProjectID string `toml:"project_id"`
	Location  string `toml:"location"`
	APIKey    string `toml:"api_key"`
}

// GeminiOptions holds Gemini-specific settings. Currently Vertex AI mode only
// (the underlying gollem package does not expose API key direct mode).
type GeminiOptions struct {
	ProjectID      string `toml:"project_id"`
	Location       string `toml:"location"`
	APIKey         string `toml:"api_key"` // reserved; rejected by validation
	ThinkingBudget *int   `toml:"thinking_budget"`
}

// OpenAIOptions holds OpenAI-specific settings. API key only (gollem does not
// expose Azure OpenAI in this build).
type OpenAIOptions struct {
	APIKey string `toml:"api_key"`
}

// EmbeddingConfig is the embedding client config.
// Supports provider = "gemini" (Vertex), "openai", or "noop" (test-only).
type EmbeddingConfig struct {
	Provider  string `toml:"provider"`
	Model     string `toml:"model"`
	ProjectID string `toml:"project_id"`
	Location  string `toml:"location"`
	APIKey    string `toml:"api_key"`
}

// Provider names.
const (
	ProviderClaude = "claude"
	ProviderGemini = "gemini"
	ProviderOpenAI = "openai"
	// ProviderNoop is a test-only provider that returns canned responses and
	// zero-vector embeddings. Intended for E2E/UI tests where LLM is not the
	// system under test. Not for production use.
	ProviderNoop = "noop"
)
