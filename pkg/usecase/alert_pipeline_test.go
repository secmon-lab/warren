package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/event"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/usecase"
)

type testNotifier struct {
	events *[]event.NotificationEvent
}

func (n *testNotifier) NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent) {
	*n.events = append(*n.events, ev)
}

func (n *testNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
	*n.events = append(*n.events, ev)
}

func (n *testNotifier) NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent) {
	*n.events = append(*n.events, ev)
}

func (n *testNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
	*n.events = append(*n.events, ev)
}

func (n *testNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
	*n.events = append(*n.events, ev)
}

func (n *testNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {
	*n.events = append(*n.events, ev)
}

var _ interfaces.Notifier = (*testNotifier)(nil)

type noopNotifier struct{}

func (n *noopNotifier) NotifyAlertPolicyResult(ctx context.Context, ev *event.AlertPolicyResultEvent) {
}

func (n *noopNotifier) NotifyEnrichPolicyResult(ctx context.Context, ev *event.EnrichPolicyResultEvent) {
}

func (n *noopNotifier) NotifyCommitPolicyResult(ctx context.Context, ev *event.CommitPolicyResultEvent) {
}

func (n *noopNotifier) NotifyEnrichTaskPrompt(ctx context.Context, ev *event.EnrichTaskPromptEvent) {
}

func (n *noopNotifier) NotifyEnrichTaskResponse(ctx context.Context, ev *event.EnrichTaskResponseEvent) {
}

func (n *noopNotifier) NotifyError(ctx context.Context, ev *event.ErrorEvent) {}

var _ interfaces.Notifier = (*noopNotifier)(nil)

type mockLLMClient struct {
	NewSessionFunc func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error)
}

func (m *mockLLMClient) NewSession(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
	if m.NewSessionFunc != nil {
		return m.NewSessionFunc(ctx, options...)
	}
	return nil, nil
}

func (m *mockLLMClient) GenerateEmbedding(ctx context.Context, dimension int, input []string) ([][]float64, error) {
	// Return a mock embedding for each input
	result := make([][]float64, len(input))
	for i := range input {
		result[i] = []float64{0.1, 0.2, 0.3} // Mock 3-dimensional embedding
	}
	return result, nil
}

type mockSession struct {
	GenerateContentFunc func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error)
}

func (m *mockSession) GenerateContent(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
	if m.GenerateContentFunc != nil {
		return m.GenerateContentFunc(ctx, input...)
	}
	return &gollem.Response{}, nil
}

func (m *mockSession) GenerateStream(ctx context.Context, input ...gollem.Input) (<-chan *gollem.Response, error) {
	return nil, nil
}

func (m *mockSession) History() (*gollem.History, error) {
	return nil, nil
}

func TestProcessAlertPipeline_Basic(t *testing.T) {
	t.Run("processes alert through pipeline successfully", func(t *testing.T) {
		ctx := context.Background()

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
				"testdata/enrich_no_tasks.rego",
				"testdata/commit_basic.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		events := []event.NotificationEvent{}
		notifier := &testNotifier{events: &events}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, results[0].Alert.Metadata.Title, "Updated Title")
		gt.True(t, len(events) > 0)
	})

	t.Run("processes alert with query task", func(t *testing.T) {
		ctx := context.Background()

		// Mock LLM client
		mockLLM := &mockLLMClient{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				return &mockSession{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{`{"severity": "high"}`},
						}, nil
					},
				}, nil
			},
		}

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
				"testdata/enrich_query_task.rego",
				"testdata/commit_simple.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithLLMClient(mockLLM),
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 1)
		gt.NotEqual(t, results[0].EnrichResult["analyze"], nil)
	})

	t.Run("processes alert with inline prompt", func(t *testing.T) {
		ctx := context.Background()

		// Mock LLM client
		mockLLM := &mockLLMClient{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				return &mockSession{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"analysis result"},
						}, nil
					},
				}, nil
			},
		}

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
				"testdata/enrich_inline_prompt.rego",
				"testdata/commit_simple.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithLLMClient(mockLLM),
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 1)
		gt.Equal(t, results[0].EnrichResult["task1"], "analysis result")
	})

	t.Run("returns empty results when no alerts", func(t *testing.T) {
		ctx := context.Background()

		// Create policy that returns no alerts (input doesn't match)
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline with data that doesn't match alert policy
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"no_match": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 0)
	})

	t.Run("works without enrich policy", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with only alert and commit policies (no enrich)
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
				"testdata/commit_basic.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 0) // No enrich tasks
		gt.Equal(t, results[0].Alert.Metadata.Title, "Updated Title")
	})

	t.Run("works without commit policy", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with only alert and enrich policies (no commit)
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
				"testdata/enrich_no_tasks.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		// Commit policy should use default behavior (PublishTypeAlert)
		gt.Equal(t, results[0].CommitResult.Publish, "alert")
	})

	t.Run("works without enrich and commit policies", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with only alert policy
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/alert_test.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &noopNotifier{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 0)          // No enrich tasks
		gt.Equal(t, results[0].CommitResult.Publish, "alert") // Default publish type
	})
}
