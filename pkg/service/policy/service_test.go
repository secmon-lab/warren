package policy_test

import (
	"context"
	"errors"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	domainPolicy "github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/policy"
)

type mockPolicyClient struct {
	QueryFunc func(ctx context.Context, path string, input any, output any) error
}

func (m *mockPolicyClient) Query(ctx context.Context, path string, input any, output any, opts ...opaq.QueryOption) error {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, path, input, output)
	}
	return nil
}

func (m *mockPolicyClient) Sources() map[string]string {
	return make(map[string]string)
}

func TestService_EvaluateEnrichPolicy(t *testing.T) {
	t.Run("evaluates enrich policy successfully", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Title: "Test Alert",
		})

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				gt.Equal(t, path, "data.enrich")

				// Set enrich policy result
				result := output.(*domainPolicy.EnrichPolicyResult)
				result.Query = []domainPolicy.QueryTask{
					{EnrichTask: domainPolicy.EnrichTask{
						ID:     "task1",
						Inline: "Analyze this alert",
						Format: types.GenAIContentFormatText,
					}},
				}
				result.Agent = []domainPolicy.AgentTask{
					{EnrichTask: domainPolicy.EnrichTask{
						ID:     "task2",
						Inline: "Investigate this threat",
						Format: types.GenAIContentFormatJSON,
					}},
				}
				return nil
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateEnrichPolicy(ctx, &a)

		gt.NoError(t, err)
		gt.NotEqual(t, result, nil)
		gt.Equal(t, len(result.Query), 1)
		gt.Equal(t, len(result.Agent), 1)
		gt.Equal(t, result.Query[0].ID, "task1")
		gt.Equal(t, result.Agent[0].ID, "task2")
	})

	t.Run("returns empty result when no policy defined", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				return opaq.ErrNoEvalResult
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateEnrichPolicy(ctx, &a)

		gt.NoError(t, err)
		gt.NotEqual(t, result, nil)
		gt.Equal(t, len(result.Query), 0)
		gt.Equal(t, len(result.Agent), 0)
	})

	t.Run("ensures task IDs are generated", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				// Return tasks without IDs
				result := output.(*domainPolicy.EnrichPolicyResult)
				result.Query = []domainPolicy.QueryTask{
					{EnrichTask: domainPolicy.EnrichTask{
						Inline: "Analyze this",
						Format: types.GenAIContentFormatText,
					}},
				}
				return nil
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateEnrichPolicy(ctx, &a)

		gt.NoError(t, err)
		gt.NotEqual(t, result, nil)
		gt.Equal(t, len(result.Query), 1)
		// ID should be auto-generated
		gt.NotEqual(t, result.Query[0].ID, "")
	})

	t.Run("returns error on policy evaluation failure", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				return goerr.New("policy evaluation failed")
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateEnrichPolicy(ctx, &a)

		gt.Error(t, err)
		gt.Equal(t, result, nil)
	})
}

func TestService_EvaluateCommitPolicy(t *testing.T) {
	t.Run("evaluates commit policy successfully", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{
			Title: "Test Alert",
		})

		enrichResults := domainPolicy.EnrichResults{
			"task1": map[string]any{"severity": "high"},
		}

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				gt.Equal(t, path, "data.commit")

				// Verify input structure
				commitInput := input.(domainPolicy.CommitPolicyInput)
				gt.NotEqual(t, commitInput.Alert.ID, "")
				gt.Equal(t, len(commitInput.Enrich), 1)

				// Set commit policy result
				result := output.(*domainPolicy.CommitPolicyResult)
				result.Title = "Updated Title"
				result.Description = "Updated Description"
				result.Channel = "security-alerts"
				result.Publish = types.PublishTypeAlert
				return nil
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateCommitPolicy(ctx, &a, enrichResults)

		gt.NoError(t, err)
		gt.NotEqual(t, result, nil)
		gt.Equal(t, result.Title, "Updated Title")
		gt.Equal(t, result.Description, "Updated Description")
		gt.Equal(t, result.Channel, "security-alerts")
		gt.Equal(t, result.Publish, types.PublishTypeAlert)
	})

	t.Run("returns default behavior when no policy defined", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})
		enrichResults := domainPolicy.EnrichResults{}

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				return opaq.ErrNoEvalResult
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateCommitPolicy(ctx, &a, enrichResults)

		gt.NoError(t, err)
		gt.NotEqual(t, result, nil)
		gt.Equal(t, result.Publish, types.PublishTypeAlert) // Default behavior
		gt.Equal(t, result.Title, "")
		gt.Equal(t, result.Description, "")
	})

	t.Run("returns error on policy evaluation failure", func(t *testing.T) {
		ctx := context.Background()
		a := alert.New(ctx, "test-schema", nil, alert.Metadata{})
		enrichResults := domainPolicy.EnrichResults{}

		mockClient := &mockPolicyClient{
			QueryFunc: func(ctx context.Context, path string, input any, output any) error {
				return errors.New("policy evaluation failed")
			},
		}

		svc := policy.New(mockClient)
		result, err := svc.EvaluateCommitPolicy(ctx, &a, enrichResults)

		gt.Error(t, err)
		gt.Equal(t, result, nil)
	})
}
