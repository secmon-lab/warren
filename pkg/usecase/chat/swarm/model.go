package swarm

// TaskPlan represents a single task in the parallel execution plan.
type TaskPlan struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
	Tools              []string `json:"tools"`
}

// PlanResult represents the LLM planning response.
type PlanResult struct {
	Message string     `json:"message"`
	Tasks   []TaskPlan `json:"tasks"`
}

// ReplanResult represents the LLM replan response.
// If Question is non-nil, it takes priority over Tasks (Tasks are ignored).
type ReplanResult struct {
	Message  string     `json:"message,omitempty"`
	Tasks    []TaskPlan `json:"tasks"`
	Question *Question  `json:"question,omitempty"`
}

// Question represents a question to ask the security operator.
type Question struct {
	Question string   `json:"question"` // The question text
	Options  []string `json:"options"`  // Answer choices (required)
	Reason   string   `json:"reason"`   // Why this question is needed
}

// TaskResult holds the outcome of a single task execution.
type TaskResult struct {
	TaskID         string
	Title          string
	Result         string
	Error          error
	BudgetExceeded bool // true if the task was terminated due to budget exhaustion
}
