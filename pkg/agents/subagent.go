package agents

import "github.com/m-mizutani/gollem"

// SubAgent wraps gollem.SubAgent with additional metadata for the parent agent.
// It provides a PromptHint that can be collected by the parent agent to include
// sub-agent specific information (e.g., available BigQuery tables) in its system prompt.
type SubAgent struct {
	inner      *gollem.SubAgent
	promptHint string
}

// NewSubAgent creates a new SubAgent wrapping a gollem.SubAgent with an optional prompt hint.
func NewSubAgent(inner *gollem.SubAgent, promptHint string) *SubAgent {
	return &SubAgent{inner: inner, promptHint: promptHint}
}

// Inner returns the underlying gollem.SubAgent for passing to gollem APIs.
func (s *SubAgent) Inner() *gollem.SubAgent {
	return s.inner
}

// PromptHint returns additional context to be included in the parent
// agent's system prompt. This helps the parent agent understand what
// this sub-agent can do and what resources it has access to.
func (s *SubAgent) PromptHint() string {
	return s.promptHint
}
