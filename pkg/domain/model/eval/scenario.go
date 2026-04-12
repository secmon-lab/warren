package eval

import (
	"fmt"
	"strings"
	"time"

	"github.com/secmon-lab/warren/pkg/domain/types"
)

// Scenario represents a loaded evaluation scenario (single scenario.yaml).
type Scenario struct {
	Name           string        `yaml:"name"`
	Description    string        `yaml:"description"`
	Alert          AlertConfig   `yaml:"alert"`
	World          WorldConfig   `yaml:"world"`
	InitialMessage string        `yaml:"initial_message"`
	Config         ScenarioConfig `yaml:"config"`
	Slack          *SlackConfig  `yaml:"slack"`
	Expectations   *Expectations `yaml:"expectations"`
}

// ScenarioConfig holds paths to config files relative to the scenario directory.
// All paths are resolved relative to the scenario folder at load time.
type ScenarioConfig struct {
	PolicyDir            string   `yaml:"policy_dir"`              // Rego policy directory (required for pipeline)
	BigQueryConfigs      []string `yaml:"bigquery_configs"`        // BigQuery dataset/table YAML configs
	BigQueryRunbooks     []string `yaml:"bigquery_runbooks"`       // BigQuery SQL runbook files/directories
	GitHubConfigs        []string `yaml:"github_configs"`          // GitHub repository config YAMLs
	UserSystemPrompt     string   `yaml:"user_system_prompt"`      // User system prompt file (markdown)
}

// AlertConfig defines the alert to inject into the pipeline.
type AlertConfig struct {
	Schema types.AlertSchema `yaml:"schema"`
	Data   any               `yaml:"data"`
}

// WorldConfig defines the mock world for dynamic response generation.
type WorldConfig struct {
	Description string            `yaml:"description"`
	ToolHints   map[string]string `yaml:"tool_hints"`
}

// SlackConfig holds Slack simulation settings.
type SlackConfig struct {
	Channel string `yaml:"channel"`
}

// Expectations defines the 3-layer evaluation criteria.
type Expectations struct {
	Outcome    *OutcomeExpectation    `yaml:"outcome"`
	Trajectory *TrajectoryExpectation `yaml:"trajectory"`
	Efficiency *EfficiencyExpectation `yaml:"efficiency"`
}

// OutcomeExpectation — Layer A: result evaluation.
type OutcomeExpectation struct {
	FindingMustContain []string `yaml:"finding_must_contain"`
	Severity           string   `yaml:"severity"`
	Criteria           []string `yaml:"criteria"`
}

// TrajectoryExpectation — Layer B: tool call sequence evaluation.
type TrajectoryExpectation struct {
	MustCall     []string `yaml:"must_call"`
	MustNotCall  []string `yaml:"must_not_call"`
	OrderedCalls []string `yaml:"ordered_calls"`
}

// EfficiencyExpectation — Layer C: resource usage evaluation.
type EfficiencyExpectation struct {
	MaxTotalCalls     int `yaml:"max_total_calls"`
	MaxDuplicateCalls int `yaml:"max_duplicate_calls"`
}

// Validate checks the Scenario for structural and logical consistency.
// Returns a list of validation errors (empty = valid).
func (s *Scenario) Validate() []string {
	var errors []string

	if s.Name == "" {
		errors = append(errors, "name is required")
	}
	if s.Alert.Schema == "" {
		errors = append(errors, "alert.schema is required")
	}
	if s.Alert.Data == nil {
		errors = append(errors, "alert.data is required")
	}
	if s.InitialMessage == "" {
		errors = append(errors, "initial_message is required")
	}
	if s.World.Description == "" {
		errors = append(errors, "world.description is required (mock agent needs scenario context)")
	}
	if s.Config.PolicyDir == "" {
		errors = append(errors, "config.policy_dir is required (pipeline needs Rego policies)")
	}

	if s.Expectations != nil {
		errors = append(errors, s.Expectations.Validate()...)
	}

	return errors
}

// Validate checks Expectations for logical consistency.
func (e *Expectations) Validate() []string {
	var errors []string

	if e.Trajectory != nil {
		mustCallSet := make(map[string]bool)
		for _, t := range e.Trajectory.MustCall {
			mustCallSet[t] = true
		}
		for _, t := range e.Trajectory.MustNotCall {
			if mustCallSet[t] {
				errors = append(errors, fmt.Sprintf("tool %q appears in both must_call and must_not_call", t))
			}
		}
		if len(e.Trajectory.MustCall) > 0 && len(e.Trajectory.OrderedCalls) > 0 {
			for _, t := range e.Trajectory.OrderedCalls {
				if !mustCallSet[t] {
					errors = append(errors, fmt.Sprintf("ordered_calls tool %q is not in must_call", t))
				}
			}
		}
	}

	if e.Efficiency != nil {
		if e.Efficiency.MaxTotalCalls < 0 {
			errors = append(errors, "efficiency.max_total_calls must be non-negative")
		}
		if e.Efficiency.MaxDuplicateCalls < 0 {
			errors = append(errors, "efficiency.max_duplicate_calls must be non-negative")
		}
	}

	return errors
}

// Validate checks a ResponseFile for structural correctness.
// filename is the on-disk filename (e.g., "virustotal__a3f2b1c9.json") for prefix checking.
func (rf *ResponseFile) Validate(filename string) []string {
	var errors []string

	if rf.ToolName == "" {
		errors = append(errors, fmt.Sprintf("responses/%s: tool_name is required", filename))
	}
	if rf.Response == nil {
		errors = append(errors, fmt.Sprintf("responses/%s: response is required", filename))
	}
	if rf.ToolName != "" {
		expectedPrefix := rf.ToolName + "__"
		if !strings.HasPrefix(filename, expectedPrefix) {
			errors = append(errors, fmt.Sprintf("responses/%s: filename should start with %q (got tool_name=%q)", filename, expectedPrefix, rf.ToolName))
		}
	}

	return errors
}

// ResponseSource indicates where a tool response came from.
type ResponseSource string

const (
	ResponseSourcePreDefined ResponseSource = "pre_defined"
	ResponseSourceGenerated  ResponseSource = "generated"
)

// ToolCallRecord records a single tool invocation during evaluation.
type ToolCallRecord struct {
	Sequence   int            `json:"sequence"`
	ToolName   string         `json:"tool_name"`
	Args       map[string]any `json:"args"`
	Response   map[string]any `json:"response"`
	Source     ResponseSource `json:"source"`
	Duration   time.Duration  `json:"duration"`
	TokensUsed int64          `json:"tokens_used,omitempty"`
}

// PipelineTrace records pipeline stage results.
type PipelineTrace struct {
	IngestResult string `json:"ingest_result"`
	EnrichTasks  int    `json:"enrich_tasks"`
	TriageResult string `json:"triage_result"`
}

// Trace records the full execution trace of a scenario run.
type Trace struct {
	ScenarioName   string           `json:"scenario_name"`
	RunID          string           `json:"run_id"`
	StartTime      time.Time        `json:"start_time"`
	EndTime        time.Time        `json:"end_time"`
	ToolCalls      []ToolCallRecord `json:"tool_calls"`
	AgentOutput    string           `json:"agent_output"`
	PipelineResult *PipelineTrace   `json:"pipeline_result,omitempty"`
	TotalTokens    int64            `json:"total_tokens"`
}

// EvalResult represents the structured evaluation across 3 layers.
type EvalResult struct {
	Trace      *Trace            `json:"trace"`
	Outcome    *OutcomeResult    `json:"outcome"`
	Trajectory *TrajectoryResult `json:"trajectory"`
	Efficiency *EfficiencyResult `json:"efficiency"`
}

// OutcomeResult — deterministic checks + LLM judge.
type OutcomeResult struct {
	FindingKeywordsFound  []string          `json:"finding_keywords_found"`
	FindingKeywordsMissed []string          `json:"finding_keywords_missed"`
	SeverityMatch         bool              `json:"severity_match"`
	CriteriaResults       []CriterionResult `json:"criteria_results"`
}

// CriterionResult holds a single LLM judge evaluation.
type CriterionResult struct {
	Criterion string `json:"criterion"`
	Pass      bool   `json:"pass"`
	Reasoning string `json:"reasoning"`
}

// TrajectoryResult — tool call sequence evaluation.
type TrajectoryResult struct {
	MustCallResults    map[string]bool `json:"must_call_results"`
	MustNotCallResults map[string]bool `json:"must_not_call_results"`
	OrderedCallsPass   bool            `json:"ordered_calls_pass"`
	OrderedCallsDetail string          `json:"ordered_calls_detail"`
}

// EfficiencyResult — quantitative metrics.
type EfficiencyResult struct {
	TotalCalls       int           `json:"total_calls"`
	DuplicateCalls   int           `json:"duplicate_calls"`
	TotalTokens      int64         `json:"total_tokens"`
	Duration         time.Duration `json:"duration"`
	MaxCallsPass     bool          `json:"max_calls_pass"`
	MaxDuplicatePass bool          `json:"max_duplicate_pass"`
}

// ReportFormat defines the output format for evaluation reports.
type ReportFormat string

const (
	ReportFormatMarkdown ReportFormat = "markdown"
	ReportFormatJSON     ReportFormat = "json"
)

// Report represents the final evaluation report.
type Report struct {
	Format     ReportFormat `json:"format"`
	Content    string       `json:"content"`
	EvalResult *EvalResult  `json:"eval_result"`
}

// ResponseFile represents the on-disk format of a tool response file.
type ResponseFile struct {
	ToolName string         `json:"tool_name"`
	Args     map[string]any `json:"args"`
	Response map[string]any `json:"response"`
}
