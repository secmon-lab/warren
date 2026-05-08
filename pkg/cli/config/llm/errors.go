package llm

import "github.com/m-mizutani/goerr/v2"

// Sentinel errors for LLM resolution.
var (
	// ErrEmptyLLMID is returned when an empty llm_id is passed to Resolve.
	ErrEmptyLLMID = goerr.New("llm_id is required")
	// ErrUnknownLLMID is returned when the requested id has no [[llm]] entry.
	ErrUnknownLLMID = goerr.New("llm_id does not match any [[llm]] entry")
	// ErrLLMNotInTaskList is returned when the id exists in [[llm]] but is not
	// listed in [agent].task — planner is restricted to the task allow-list.
	ErrLLMNotInTaskList = goerr.New("llm_id is not in [agent].task allow-list")
)
