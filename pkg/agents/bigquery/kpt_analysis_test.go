package bigquery_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/agents/bigquery"
	"github.com/secmon-lab/warren/pkg/service/memory"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestGenerateKPTAnalysis_Success(t *testing.T) {
	// Mock LLM client that returns valid JSON
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"successes": [
								"Found login errors using severity='ERROR' AND action='login'. User email is in user.email (STRING) field.",
								"Query used event_time (TIMESTAMP) field for time filtering, which is the partition field."
							],
							"problems": [],
							"improvements": []
						}`},
					}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	memSvc := memory.New(mockLLM, nil)
	agent := bigquery.NewAgent(&bigquery.Config{}, mockLLM, memSvc)

	resp := &gollem.ExecuteResponse{}
	successes, problems, improvements, err := agent.GenerateKPTAnalysis(
		ctx,
		"find login errors",
		resp,
		nil,
		0,
		nil, // session
	)

	gt.NoError(t, err)
	gt.A(t, successes).Length(2)
	gt.A(t, problems).Length(0)
	gt.A(t, improvements).Length(0)
	gt.True(t, len(successes[0]) > 0 && strings.Contains(successes[0], "severity='ERROR'"))
	gt.True(t, len(successes[0]) > 0 && strings.Contains(successes[0], "user.email"))
}

func TestGenerateKPTAnalysis_Failure(t *testing.T) {
	// Mock LLM client that returns KPT for failed execution
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"successes": [],
							"problems": [
								"Expected 'timestamp' field but actual field is 'event_time' (TIMESTAMP type)",
								"Query exceeded scan size limit due to missing partition filter"
							],
							"improvements": [
								"Always check table schema first using bigquery_schema tool",
								"Use event_time field with WHERE clause for time filtering"
							]
						}`},
					}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	memSvc := memory.New(mockLLM, nil)
	agent := bigquery.NewAgent(&bigquery.Config{}, mockLLM, memSvc)

	execErr := errors.New("query exceeded scan size limit")
	successes, problems, improvements, err := agent.GenerateKPTAnalysis(
		ctx,
		"find login errors",
		nil,
		execErr,
		0,
		nil, // session
	)

	gt.NoError(t, err)
	gt.A(t, successes).Length(0)
	gt.A(t, problems).Length(2)
	gt.A(t, improvements).Length(2)
	gt.True(t, strings.Contains(problems[0], "timestamp"))
	gt.True(t, strings.Contains(problems[0], "event_time"))
	gt.True(t, strings.Contains(improvements[0], "schema"))
}

func TestGenerateKPTAnalysis_LLMError(t *testing.T) {
	// Mock LLM client that returns error
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return nil, errors.New("llm session error")
		},
	}

	ctx := context.Background()
	memSvc := memory.New(mockLLM, nil)
	agent := bigquery.NewAgent(&bigquery.Config{}, mockLLM, memSvc)

	successes, problems, improvements, err := agent.GenerateKPTAnalysis(
		ctx,
		"find login errors",
		nil,
		nil,
		0,
		nil, // session
	)

	// Should return error with fallback tag and empty arrays
	gt.Error(t, err)
	gt.True(t, goerr.HasTag(err, goerr.NewTag("kpt_analysis_fallback")))
	gt.A(t, successes).Length(0)
	gt.A(t, problems).Length(0)
	gt.A(t, improvements).Length(0)
}

func TestGenerateKPTAnalysis_InvalidJSON(t *testing.T) {
	// Mock LLM client that returns invalid JSON
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"This is not valid JSON"},
					}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	memSvc := memory.New(mockLLM, nil)
	agent := bigquery.NewAgent(&bigquery.Config{}, mockLLM, memSvc)

	successes, problems, improvements, err := agent.GenerateKPTAnalysis(
		ctx,
		"find login errors",
		nil,
		nil,
		0,
		nil, // session
	)

	// Should return error with fallback tag and empty arrays
	gt.Error(t, err)
	gt.True(t, goerr.HasTag(err, goerr.NewTag("kpt_analysis_fallback")))
	gt.A(t, successes).Length(0)
	gt.A(t, problems).Length(0)
	gt.A(t, improvements).Length(0)
}

func TestGenerateKPTAnalysis_WithMarkdownCodeBlocks(t *testing.T) {
	// Mock LLM that returns JSON wrapped in markdown code blocks
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"```json\n" + `{
							"successes": ["Test success"],
							"problems": [],
							"improvements": []
						}` + "\n```"},
					}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	memSvc := memory.New(mockLLM, nil)
	agent := bigquery.NewAgent(&bigquery.Config{}, mockLLM, memSvc)

	successes, _, _, err := agent.GenerateKPTAnalysis(
		ctx,
		"test query",
		nil,
		nil,
		0,
		nil, // session
	)

	gt.NoError(t, err)
	gt.A(t, successes).Length(1)
	gt.Equal(t, successes[0], "Test success")
}

// TestGenerateKPTAnalysis_RealLLM tests with actual Gemini LLM
// This test only runs when TEST_GEMINI_PROJECT_ID is set
func TestGenerateKPTAnalysis_RealLLM(t *testing.T) {
	if os.Getenv("TEST_GEMINI_PROJECT_ID") == "" {
		t.Skip("TEST_GEMINI_PROJECT_ID not set, skipping real LLM test")
	}

	ctx := context.Background()

	// Create real Gemini client
	llmClient := test.NewGeminiClient(t)

	memSvc := memory.New(llmClient, nil)

	// Create agent with empty config for testing
	config := &bigquery.Config{
		Tables: []bigquery.TableConfig{
			{
				ProjectID:   "test-project",
				DatasetID:   "security_logs",
				TableID:     "events",
				Description: "Security event logs",
			},
		},
	}
	agent := bigquery.NewAgent(config, llmClient, memSvc)

	t.Run("success case", func(t *testing.T) {
		// Simulate successful execution with results
		resp := gollem.NewExecuteResponse(
			"Found 42 authentication failures",
			"Query executed successfully using event_type='AUTHENTICATION_FAILURE' AND event_timestamp > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 24 HOUR)",
		)
		successes, problems, improvements, err := agent.GenerateKPTAnalysis(
			ctx,
			"find authentication failures in the last 24 hours",
			resp,
			nil,
			0,
			nil, // session
		)

		gt.NoError(t, err)
		t.Logf("Successes: %v", successes)
		t.Logf("Problems: %v", problems)
		t.Logf("Improvements: %v", improvements)

		// Verify response structure
		gt.True(t, len(successes) >= 1 && len(successes) <= 4)

		// Verify content quality (should contain domain knowledge)
		if len(successes) > 0 {
			hasFieldName := false
			for _, s := range successes {
				gt.True(t, len(s) >= 50)
				gt.True(t, len(s) <= 600)
				if len(s) >= 50 && len(s) <= 600 {
					// Check for field names or data types
					if containsAny(s, []string{"field", "Field", "column", "STRING", "INT64", "TIMESTAMP"}) {
						hasFieldName = true
					}
				}
			}
			gt.True(t, hasFieldName)
		}
	})

	t.Run("failure case", func(t *testing.T) {
		// Simulate failed execution
		execErr := errors.New("query exceeded scan size limit: 15TB scanned, limit is 10GB")
		successes, problems, improvements, err := agent.GenerateKPTAnalysis(
			ctx,
			"find all errors in the table",
			nil,
			execErr,
			0,
			nil, // session
		)

		gt.NoError(t, err)
		t.Logf("Successes: %v", successes)
		t.Logf("Problems: %v", problems)
		t.Logf("Improvements: %v", improvements)

		// Verify response structure
		gt.A(t, successes).Length(0)
		gt.True(t, len(problems) >= 1 && len(problems) <= 3)
		gt.True(t, len(improvements) >= 1 && len(improvements) <= 4)

		// Verify content quality
		if len(problems) > 0 {
			for _, p := range problems {
				gt.True(t, len(p) >= 50)
				gt.True(t, len(p) <= 600)
			}
		}

		if len(improvements) > 0 {
			for _, imp := range improvements {
				gt.True(t, len(imp) >= 50)
				gt.True(t, len(imp) <= 600)
			}
		}
	})
}

// containsAny checks if string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
