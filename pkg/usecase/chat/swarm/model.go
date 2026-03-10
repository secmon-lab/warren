package swarm

// TaskPlan represents a single task in the parallel execution plan.
type TaskPlan struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	Tools              []string `json:"tools"`
	SubAgents          []string `json:"sub_agents"`
}

// PlanResult represents the LLM planning response.
type PlanResult struct {
	Message string     `json:"message"`
	Tasks   []TaskPlan `json:"tasks"`
}

// ReplanResult represents the LLM replan response.
type ReplanResult struct {
	Message  string     `json:"message,omitempty"`
	Tasks    []TaskPlan `json:"tasks"`
	Question string     `json:"question,omitempty"`
}

// TaskResult holds the outcome of a single task execution.
type TaskResult struct {
	TaskID         string
	Title          string
	Result         string
	Error          error
	BudgetExceeded bool // true if the task was terminated due to budget exhaustion
}
