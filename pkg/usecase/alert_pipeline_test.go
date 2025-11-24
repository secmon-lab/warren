package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/notice"
	slackService "github.com/secmon-lab/warren/pkg/service/slack"
	"github.com/secmon-lab/warren/pkg/usecase"
	"github.com/slack-go/slack"
)

func TestProcessAlertPipeline_Basic(t *testing.T) {
	t.Run("processes alert through pipeline successfully", func(t *testing.T) {
		ctx := context.Background()

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_no_tasks.rego",
				"testdata/triage_basic.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Use console notifier
		consoleNotifier := &mock.NotifierMock{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, consoleNotifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, results[0].Alert.Title, "Updated Title")
	})

	t.Run("processes alert with query task", func(t *testing.T) {
		ctx := context.Background()

		// Mock LLM client with embedding support
		mockLLM := &gollem_mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				return &gollem_mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{`{"severity": "high"}`},
						}, nil
					},
				}, nil
			},
			GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
				result := make([][]float64, len(input))
				for i := range input {
					result[i] = []float64{0.1, 0.2, 0.3}
				}
				return result, nil
			},
		}

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_query_task.rego",
				"testdata/triage_simple.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithLLMClient(mockLLM),
			usecase.WithPolicyClient(policyClient),
		)

		// Use console notifier
		consoleNotifier := &mock.NotifierMock{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, consoleNotifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 1)
		gt.NotEqual(t, results[0].EnrichResult["analyze"], nil)
	})

	t.Run("processes alert with inline prompt", func(t *testing.T) {
		ctx := context.Background()

		// Mock LLM client
		mockLLM := &gollem_mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				return &gollem_mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"analysis result"},
						}, nil
					},
				}, nil
			},
			GenerateEmbeddingFunc: func(ctx context.Context, dimension int, input []string) ([][]float64, error) {
				result := make([][]float64, len(input))
				for i := range input {
					result[i] = []float64{0.1, 0.2, 0.3}
				}
				return result, nil
			},
		}

		// Create real policy client with test policies
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_inline_prompt.rego",
				"testdata/triage_simple.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithLLMClient(mockLLM),
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &mock.NotifierMock{}

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
				"testdata/ingest_test.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &mock.NotifierMock{}

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
				"testdata/ingest_test.rego",
				"testdata/triage_basic.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &mock.NotifierMock{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 0) // No enrich tasks
		gt.Equal(t, results[0].Alert.Title, "Updated Title")
	})

	t.Run("works without commit policy", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with only alert and enrich policies (no commit)
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_no_tasks.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &mock.NotifierMock{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		// Commit policy should use default behavior (PublishTypeAlert)
		gt.Equal(t, results[0].TriageResult.Publish, "alert")
	})

	t.Run("works without enrich and commit policies", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with only alert policy
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		// Test notifier
		notifier := &mock.NotifierMock{}

		// Process pipeline
		results, err := uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		gt.NoError(t, err)
		gt.NotEqual(t, results, nil)
		gt.Equal(t, len(results), 1)
		gt.Equal(t, len(results[0].EnrichResult), 0)          // No enrich tasks
		gt.Equal(t, results[0].TriageResult.Publish, "alert") // Default publish type
	})

	t.Run("fails when LLM client is not configured but enrich tasks exist", func(t *testing.T) {
		ctx := context.Background()

		// Create policy client with enrich tasks
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_query_task.rego",
				"testdata/triage_simple.rego",
			),
		)
		gt.NoError(t, err)

		// Create use case WITHOUT LLM client
		uc := usecase.New(
			usecase.WithPolicyClient(policyClient),
		)

		notifier := &mock.NotifierMock{}

		// Process pipeline - should fail with appropriate error
		_, err = uc.ProcessAlertPipeline(ctx, "test_schema", map[string]any{"test": "data"}, notifier)

		// Should return error when LLM client is not configured
		gt.Error(t, err)
	})
}

func TestHandleAlert_PublishTypes(t *testing.T) {
	t.Run("publish=alert posts to Slack with pipeline events", func(t *testing.T) {
		ctx := context.Background()

		// Create mock Slack client
		slackMock := &mock.SlackClientMock{
			PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return channelID, "1234567890.123456", nil
			},
			AuthTestFunc: func() (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID:       "U123",
					TeamID:       "T123",
					Team:         "test-team",
					EnterpriseID: "",
					BotID:        "B123",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack.TeamInfo, error) {
				return &slack.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}

		// Mock repository
		repoMock := &mock.RepositoryMock{
			PutAlertFunc: func(ctx context.Context, a alert.Alert) error {
				return nil
			},
		}

		// Create Slack service with mock client
		slackSvc, err := slackService.New(slackMock, "C1234567890")
		gt.NoError(t, err)

		// Create policy
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_no_tasks.rego",
				"testdata/triage_publish_alert.rego",
			),
		)
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithPolicyClient(policyClient),
			usecase.WithSlackService(slackSvc),
		)

		// Handle alert
		results, err := uc.HandleAlert(ctx, "test_schema", map[string]any{"test": "data"})
		gt.NoError(t, err)
		gt.Equal(t, len(results), 1)

		// Verify Slack posts
		postCalls := slackMock.PostMessageContextCalls()
		gt.True(t, len(postCalls) >= 3) // At least: alert + alert_policy_result + commit_policy_result

		// Verify all posts are to the correct channel
		for _, call := range postCalls {
			gt.Equal(t, call.ChannelID, "C1234567890")
			gt.NotEqual(t, call.Ctx, nil)
		}

		// First call should be the main alert post (has blocks but not context block)
		// Subsequent calls should be context blocks for pipeline events
		gt.True(t, len(postCalls[0].Options) > 0) // Alert has blocks

		// At least one call should have context blocks (pipeline events)
		hasContextBlock := false
		for i := 1; i < len(postCalls); i++ {
			if len(postCalls[i].Options) > 0 {
				hasContextBlock = true
			}
		}
		gt.True(t, hasContextBlock)

		// Verify repository save with alert details
		putCalls := repoMock.PutAlertCalls()
		gt.Array(t, putCalls).Length(1)
		savedAlert := putCalls[0].AlertMoqParam
		gt.Equal(t, savedAlert.Schema, "test_schema")
		gt.NotEqual(t, savedAlert.ID, "")
		gt.NotEqual(t, savedAlert.SlackThread, nil)
		gt.Equal(t, savedAlert.SlackThread.ChannelID, "C1234567890")
		gt.Equal(t, savedAlert.SlackThread.ThreadID, "1234567890.123456")
	})

	t.Run("publish=notice posts to Slack with pipeline events in thread", func(t *testing.T) {
		ctx := context.Background()

		// Create mock Slack client
		slackMock := &mock.SlackClientMock{
			PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return channelID, "1234567890.123456", nil
			},
			AuthTestFunc: func() (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID:       "U123",
					TeamID:       "T123",
					Team:         "test-team",
					EnterpriseID: "",
					BotID:        "B123",
				}, nil
			},
			GetTeamInfoFunc: func() (*slack.TeamInfo, error) {
				return &slack.TeamInfo{
					Domain: "test-workspace",
				}, nil
			},
		}

		// Mock repository
		repoMock := &mock.RepositoryMock{
			CreateNoticeFunc: func(ctx context.Context, n *notice.Notice) error {
				return nil
			},
			UpdateNoticeFunc: func(ctx context.Context, n *notice.Notice) error {
				return nil
			},
		}

		// Create Slack service with mock client
		slackSvc, err := slackService.New(slackMock, "C1234567890")
		gt.NoError(t, err)

		// Create policy with notice publish type
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_no_tasks.rego",
				"testdata/triage_publish_notice.rego",
			),
		)
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithPolicyClient(policyClient),
			usecase.WithSlackService(slackSvc),
		)

		// Handle alert - should create notice
		results, err := uc.HandleAlert(ctx, "test_schema", map[string]any{"test": "data"})
		gt.NoError(t, err)
		gt.Equal(t, len(results), 0) // Notices don't return as alerts

		// Verify Slack posts
		postCalls := slackMock.PostMessageContextCalls()
		gt.True(t, len(postCalls) >= 3) // notice + thread details + pipeline events

		// Verify all posts are to the correct channel
		for _, call := range postCalls {
			gt.Equal(t, call.ChannelID, "C1234567890")
			gt.NotEqual(t, call.Ctx, nil)
		}

		// First post should be the notice
		gt.True(t, len(postCalls[0].Options) > 0)

		// Subsequent posts should include thread details and pipeline events (context blocks)
		hasContextBlock := false
		for i := 1; i < len(postCalls); i++ {
			if len(postCalls[i].Options) > 0 {
				hasContextBlock = true
			}
		}
		gt.True(t, hasContextBlock)

		// Verify notice creation with details
		createCalls := repoMock.CreateNoticeCalls()
		gt.Array(t, createCalls).Length(1)
		createdNotice := createCalls[0].NoticeMoqParam
		gt.NotEqual(t, createdNotice.ID, "")
		gt.NotEqual(t, createdNotice.Alert.ID, "")
		gt.Equal(t, createdNotice.Alert.Schema, "test_schema")
		gt.Equal(t, createdNotice.Alert.Title, "Test Alert")
		gt.Equal(t, createdNotice.Alert.Description, "Test Description")
		gt.NotEqual(t, createdNotice.SlackTS, "")
		gt.Equal(t, createdNotice.Escalated, false)
	})

	t.Run("publish=discard does not post to Slack", func(t *testing.T) {
		ctx := context.Background()

		slackMock := &mock.SlackClientMock{}
		repoMock := &mock.RepositoryMock{}

		// Create policy with discard
		policyClient, err := opaq.New(
			opaq.Files(
				"testdata/ingest_test.rego",
				"testdata/enrich_no_tasks.rego",
				"testdata/triage_publish_discard.rego",
			),
		)
		gt.NoError(t, err)

		uc := usecase.New(
			usecase.WithRepository(repoMock),
			usecase.WithPolicyClient(policyClient),
		)

		// Handle alert
		results, err := uc.HandleAlert(ctx, "test_schema", map[string]any{"test": "data"})
		gt.NoError(t, err)
		gt.Equal(t, len(results), 0)

		// Verify NO Slack posts
		postCalls := slackMock.PostMessageContextCalls()
		gt.Array(t, postCalls).Length(0)

		// Verify NO repository saves
		putCalls := repoMock.PutAlertCalls()
		gt.Array(t, putCalls).Length(0)
	})
}
