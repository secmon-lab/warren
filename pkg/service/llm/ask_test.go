package llm_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/llm/gemini"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/service/llm"
)

func TestAsk(t *testing.T) {
	projectID, ok := os.LookupEnv("TEST_GEMINI_PROJECT_ID")
	if !ok {
		t.Skip("GEMINI_PROJECT_ID is not set")
	}
	location, ok := os.LookupEnv("TEST_GEMINI_LOCATION")
	if !ok {
		t.Skip("GEMINI_LOCATION is not set")
	}

	ctx := t.Context()
	geminiClient, err := gemini.New(ctx, projectID, location,
		gemini.WithThinkingBudget(0), // Disable thinking feature
	)
	gt.NoError(t, err).Required()

	type resp struct {
		Message string `json:"message"`
	}

	prompt := `Reply for 'Hello, world!'. Format is following:
	{
		"message": "Hello, world!"
	}`

	result, err := llm.Ask[resp](ctx, geminiClient, prompt)
	gt.NoError(t, err)
	gt.NotNil(t, result)
	gt.S(t, result.Message).NotEqual("")
}

func TestAskWithValidation_RetryPreservesContext(t *testing.T) {
	type result struct {
		Count int `json:"count"`
	}

	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					callCount++
					switch callCount {
					case 1:
						// First call: valid JSON but fails validation (count < 5)
						return &gollem.Response{Texts: []string{`{"count": 2}`}}, nil
					default:
						// Second call: corrected response that passes validation
						return &gollem.Response{Texts: []string{`{"count": 10}`}}, nil
					}
				},
			}, nil
		},
	}

	ctx := context.Background()
	got, err := llm.Ask[result](ctx, mockLLM, "return a count >= 5",
		llm.WithValidate(func(r result) error {
			if r.Count < 5 {
				return goerr.New("count must be >= 5")
			}
			return nil
		}),
	)

	gt.NoError(t, err)
	gt.Value(t, got.Count).Equal(10)
	// Exactly 2 LLM calls: first fails validation, second succeeds.
	// No retry multiplication from nested gollem.Query loops.
	gt.Value(t, callCount).Equal(2)
}

func TestAskWithValidation_ExhaustsRetries(t *testing.T) {
	type result struct {
		Value string `json:"value"`
	}

	callCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					callCount++
					// Always return a value that fails validation
					return &gollem.Response{Texts: []string{`{"value": "bad"}`}}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	_, err := llm.Ask[result](ctx, mockLLM, "return something good",
		llm.WithMaxRetry[result](3),
		llm.WithValidate(func(r result) error {
			if r.Value != "good" {
				return goerr.New("expected good")
			}
			return nil
		}),
	)

	gt.Error(t, err)
	// Exactly maxRetry calls, no multiplication
	gt.Value(t, callCount).Equal(3)
}

func TestAskWithValidation_UsesSessionForConversationContext(t *testing.T) {
	type result struct {
		Items []string `json:"items"`
	}

	// Track that a single session is created and reused across retries
	sessionCreateCount := 0
	generateCount := 0
	mockLLM := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			sessionCreateCount++
			return &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					generateCount++
					if generateCount == 1 {
						// First: valid JSON but validation fails (too few items)
						return &gollem.Response{Texts: []string{`{"items": ["a"]}`}}, nil
					}
					// Second: corrected response
					return &gollem.Response{Texts: []string{`{"items": ["a", "b", "c"]}`}}, nil
				},
			}, nil
		},
	}

	ctx := context.Background()
	got, err := llm.Ask[result](ctx, mockLLM, "return at least 3 items",
		llm.WithValidate(func(r result) error {
			if len(r.Items) < 3 {
				return goerr.New("need at least 3 items")
			}
			return nil
		}),
	)

	gt.NoError(t, err)
	gt.Value(t, len(got.Items)).Equal(3)
	// Single session created, reused for retry (preserves conversation context)
	gt.Value(t, sessionCreateCount).Equal(1)
	gt.Value(t, generateCount).Equal(2)
}
