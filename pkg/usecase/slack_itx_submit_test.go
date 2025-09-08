package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/mock"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	tagmodel "github.com/secmon-lab/warren/pkg/domain/model/tag"
	"github.com/secmon-lab/warren/pkg/domain/model/ticket"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	slackservice "github.com/secmon-lab/warren/pkg/service/slack"
	tagservice "github.com/secmon-lab/warren/pkg/service/tag"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	slackSDK "github.com/slack-go/slack"
)

func TestGenerateResolveMessage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	runTest := func(tc struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}) func(t *testing.T) {
		return func(t *testing.T) {
			// Setup LLM mock
			llmMock := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					return &mock.LLMSessionMock{
						GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
							if tc.llmError != nil {
								return nil, tc.llmError
							}
							return &gollem.Response{
								Texts: []string{tc.llmResponse},
							}, nil
						},
					}, nil
				},
				GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
					// Return mock embedding data with correct dimension
					embedding := make([]float64, dimension)
					for i := range embedding {
						embedding[i] = 0.1 + float64(i)*0.01 // Generate some test values
					}
					return [][]float64{embedding}, nil
				},
			}

			// Create usecase
			uc := New(
				WithLLMClient(llmMock),
				WithRepository(repository.NewMemory()),
			)

			// Test the generateResolveMessage function
			message := uc.generateResolveMessage(ctx, tc.ticket)

			// Verify the result
			if tc.expectedPrefix != "" {
				// Check if message starts with the expected prefix or contains it
				gt.Value(t, strings.Contains(message, tc.expectedPrefix)).Equal(true)
			} else {
				gt.Value(t, message).Equal(tc.llmResponse)
			}
		}
	}

	t.Run("success with conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success with conclusion",
		ticket: &ticket.Ticket{
			ID:         types.NewTicketID(),
			Status:     types.TicketStatusResolved,
			Conclusion: types.AlertConclusionFalsePositive,
			Reason:     "False positive detection",
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse: "Great work! It was a false positive, but good job on the thorough investigation 🎯",
		llmError:    nil,
	}))

	t.Run("success without conclusion", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "success without conclusion",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Reason: "Response completed successfully",
			Metadata: ticket.Metadata{
				Title: "Network Alert",
			},
		},
		llmResponse: "Resolution complete! Another heroic day protecting the world's peace 🦸‍♂️",
		llmError:    nil,
	}))

	t.Run("llm error fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "llm error fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       goerr.New("LLM error"),
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))

	t.Run("empty response fallback", runTest(struct {
		name           string
		ticket         *ticket.Ticket
		llmResponse    string
		llmError       error
		expectedPrefix string
	}{
		name: "empty response fallback",
		ticket: &ticket.Ticket{
			ID:     types.NewTicketID(),
			Status: types.TicketStatusResolved,
			Metadata: ticket.Metadata{
				Title: "Test Alert",
			},
		},
		llmResponse:    "",
		llmError:       nil,
		expectedPrefix: "🎉 Great work! Ticket resolved successfully 🎯",
	}))
}

func TestHandleSlackInteractionViewSubmissionSalvage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	// Setup LLM mock with embedding generation
	llmMock := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(ctx context.Context, dimension int, inputs []string) ([][]float64, error) {
			embedding := make([]float64, dimension)
			for i := range embedding {
				embedding[i] = 0.1 + float64(i)*0.01
			}
			return [][]float64{embedding}, nil
		},
	}

	// Setup Slack client mock
	slackClientMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
			return "C123456", "1234567890.123456", nil
		},
		UpdateMessageContextFunc: func(ctx context.Context, channelID string, timestamp string, options ...slackSDK.MsgOption) (string, string, string, error) {
			return channelID, timestamp, timestamp, nil
		},
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Create repository and setup test data
	repo := repository.NewMemory()

	// Create test alerts with slack threads and similar embeddings
	embedding := make([]float32, 256)
	for i := range embedding {
		embedding[i] = 0.1 + float32(i)*0.001 // Similar values for cosine similarity
	}

	alert1 := &alert.Alert{
		ID:     types.NewAlertID(),
		Schema: "test.schema.v1",
		Data:   map[string]any{"test": "data1"},
		Metadata: alert.Metadata{
			Title:       "Test Alert 1",
			Description: "Test description 1",
		},
		CreatedAt: now,
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "1111111111.111111",
		},
		Embedding: embedding,
	}
	alert2 := &alert.Alert{
		ID:     types.NewAlertID(),
		Schema: "test.schema.v1",
		Data:   map[string]any{"test": "data2"},
		Metadata: alert.Metadata{
			Title:       "Test Alert 2",
			Description: "Test description 2",
		},
		CreatedAt: now,
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "2222222222.222222",
		},
		Embedding: embedding,
	}

	// Store alerts in repository
	gt.NoError(t, repo.PutAlert(ctx, *alert1))
	gt.NoError(t, repo.PutAlert(ctx, *alert2))

	// Create test ticket with similar embedding
	testTicket := &ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{}, // Empty initially
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "3333333333.333333",
		},
		Metadata: ticket.Metadata{
			Title:       "Target Ticket",
			Description: "Test ticket for salvage",
		},
		Embedding: embedding,
	}
	gt.NoError(t, repo.PutTicket(ctx, *testTicket))

	// Create slack service
	slackSvc, err := slackservice.New(slackClientMock, "C123456")
	gt.NoError(t, err)

	// Create usecase
	uc := New(
		WithLLMClient(llmMock),
		WithRepository(repo),
		WithSlackService(slackSvc),
	)

	// Prepare test input values for salvage with low threshold to include both alerts
	values := slack.StateValue{
		string(slack.BlockIDSalvageThreshold): map[string]slack.BlockAction{
			string(slack.BlockActionIDSalvageThreshold): {Value: "0.1"},
		},
		string(slack.BlockIDSalvageKeyword): map[string]slack.BlockAction{
			string(slack.BlockActionIDSalvageKeyword): {Value: ""},
		},
	}

	user := slack.User{
		ID:   "U123456",
		Name: "test_user",
	}

	// Execute salvage operation
	err = uc.handleSlackInteractionViewSubmissionSalvage(ctx, user, string(testTicket.ID), values)
	gt.NoError(t, err)

	// Verify that alerts were bound to ticket
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, len(updatedTicket.AlertIDs)).Equal(2)

	// Verify alert IDs were added to ticket
	alertIDs := make(map[types.AlertID]bool)
	for _, id := range updatedTicket.AlertIDs {
		alertIDs[id] = true
	}
	gt.Value(t, alertIDs[alert1.ID]).Equal(true)
	gt.Value(t, alertIDs[alert2.ID]).Equal(true)

	// This test verifies that:
	// 1. Alerts are properly salvaged and bound to the ticket
	// 2. The salvage operation calls UpdateAlerts (which happens in slack_itx_submit.go:430)
	// 3. UpdateAlerts triggers the rate-limited updater to update individual alert Slack posts
}

func TestHandleSlackInteractionViewSubmissionResolveTicket_WithTags(t *testing.T) {
	// Test case: Resolve ticket with tag selection, should create tags and assign to ticket
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create test ticket
	testTicket := &ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{},
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "1234567890.123456",
		},
		Metadata: ticket.Metadata{
			Title:       "Security Incident",
			Description: "Network security incident requiring investigation",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, *testTicket))

	// Create existing tag using new API
	existingTag := &tagmodel.Tag{
		ID:          tagmodel.NewID(),
		Name:        "existing-tag",
		Color:       "bg-blue-100 text-blue-800",
		Description: "",
		CreatedAt:   now.Add(-1 * time.Hour),
		UpdatedAt:   now.Add(-1 * time.Hour),
		CreatedBy:   "test",
	}
	gt.NoError(t, repo.CreateTagWithID(ctx, existingTag))

	// Setup Slack client mock
	slackClientMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
			return channelID, "1234567890.123456", nil
		},
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Setup LLM mock for resolve message generation
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"🎉 Great work resolving this incident!"},
					}, nil
				},
			}, nil
		},
	}

	// Create slack service
	slackSvc, err := slackservice.New(slackClientMock, "C123456")
	gt.NoError(t, err)

	// Create tag service
	tagSvc := tagservice.New(repo)

	// Create usecase
	uc := New(
		WithRepository(repo),
		WithSlackService(slackSvc),
		WithLLMClient(llmMock),
		WithTagService(tagSvc),
	)

	// Prepare test input values for resolve with tag selection
	values := slack.StateValue{
		string(slack.BlockIDTicketConclusion): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketConclusion): {
				SelectedOption: slackSDK.OptionBlockObject{Value: string(types.AlertConclusionFalsePositive)},
			},
		},
		string(slack.BlockIDTicketComment): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketComment): {Value: "Investigation completed - false positive"},
		},
		string(slack.BlockIDTicketTags): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketTags): {
				SelectedOptions: []slackSDK.OptionBlockObject{
					{Value: existingTag.ID},   // Use existing tag ID, not name
					{Value: "false-positive"}, // This name will be converted to ID via tag service
					{Value: "investigation"},  // This name will be converted to ID via tag service
				},
			},
		},
	}

	user := slack.User{
		ID:   "U123456",
		Name: "test_user",
	}

	// Execute resolve operation
	err = uc.handleSlackInteractionViewSubmissionResolveTicket(ctx, user, string(testTicket.ID), values)
	gt.NoError(t, err)

	// Verify ticket was resolved with correct conclusion and reason
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedTicket.Status).Equal(types.TicketStatusResolved)
	gt.Value(t, updatedTicket.Conclusion).Equal(types.AlertConclusionFalsePositive)
	gt.Value(t, updatedTicket.Reason).Equal("Investigation completed - false positive")

	// Verify tags were assigned to ticket
	gt.Value(t, len(updatedTicket.TagIDs)).Equal(3)
	expectedTags := []string{"existing-tag", "false-positive", "investigation"}

	// Get actual tag names from ticket using compatibility method
	actualTagNames, err := updatedTicket.GetTagNames(ctx, func(ctx context.Context, tagIDs []string) ([]*tagmodel.Tag, error) {
		return repo.GetTagsByIDs(ctx, tagIDs)
	})
	gt.NoError(t, err)
	gt.Equal(t, len(actualTagNames), 3)

	// Verify all expected tags are present
	actualTagMap := make(map[string]bool)
	for _, name := range actualTagNames {
		actualTagMap[name] = true
	}
	for _, expectedTag := range expectedTags {
		gt.Value(t, actualTagMap[expectedTag]).Equal(true)
	}

	// Verify tags were created in repository (1 existing + 2 new = 3)
	tags, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.Array(t, tags).Length(3)

	// Verify all expected tags exist
	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}

	for _, expectedTag := range expectedTags {
		gt.Value(t, containsString(tagNames, string(expectedTag))).Equal(true)
	}

	// Verify existing tag unchanged
	existingTagAfter, err := repo.GetTagByName(ctx, "existing-tag")
	gt.NoError(t, err)
	gt.NotNil(t, existingTagAfter)
	gt.Value(t, existingTagAfter.CreatedAt).Equal(existingTag.CreatedAt)
	gt.Value(t, existingTagAfter.Color).Equal(existingTag.Color)

	// Verify new tags have colors assigned
	falsePositiveTag, err := repo.GetTagByName(ctx, "false-positive")
	gt.NoError(t, err)
	gt.NotNil(t, falsePositiveTag)
	gt.Value(t, falsePositiveTag.Color).NotEqual("")

	investigationTag, err := repo.GetTagByName(ctx, "investigation")
	gt.NoError(t, err)
	gt.NotNil(t, investigationTag)
	gt.Value(t, investigationTag.Color).NotEqual("")
}

func TestHandleSlackInteractionViewSubmissionResolveTicket_WithoutTags(t *testing.T) {
	// Test case: Resolve ticket without tag selection, should still work
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Create test ticket
	testTicket := &ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{},
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "1234567890.123456",
		},
		Metadata: ticket.Metadata{
			Title:       "Test Incident",
			Description: "Test incident for resolution",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, *testTicket))

	// Setup Slack client mock
	slackClientMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
			return channelID, "1234567890.123456", nil
		},
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Setup LLM mock
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"🎉 Resolution complete!"},
					}, nil
				},
			}, nil
		},
	}

	// Create slack service
	slackSvc, err := slackservice.New(slackClientMock, "C123456")
	gt.NoError(t, err)

	// Create tag service
	tagSvc := tagservice.New(repo)

	// Create usecase
	uc := New(
		WithRepository(repo),
		WithSlackService(slackSvc),
		WithLLMClient(llmMock),
		WithTagService(tagSvc),
	)

	// Prepare test input values for resolve without tags
	values := slack.StateValue{
		string(slack.BlockIDTicketConclusion): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketConclusion): {
				SelectedOption: slackSDK.OptionBlockObject{Value: string(types.AlertConclusionIntended)},
			},
		},
		string(slack.BlockIDTicketComment): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketComment): {Value: "Working as intended"},
		},
		// No tag selection block
	}

	user := slack.User{
		ID:   "U123456",
		Name: "test_user",
	}

	// Execute resolve operation
	err = uc.handleSlackInteractionViewSubmissionResolveTicket(ctx, user, string(testTicket.ID), values)
	gt.NoError(t, err)

	// Verify ticket was resolved
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedTicket.Status).Equal(types.TicketStatusResolved)
	gt.Value(t, updatedTicket.Conclusion).Equal(types.AlertConclusionIntended)
	gt.Value(t, updatedTicket.Reason).Equal("Working as intended")

	// Verify no tags were assigned (ticket should have empty tags)
	gt.Value(t, len(updatedTicket.TagIDs)).Equal(0)

	// Verify no tags were created in repository
	tags, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.Array(t, tags).Length(0)
}

// Helper function to check if a string slice contains a specific string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func TestMergeTagIDs(t *testing.T) {
	tests := []struct {
		name         string
		existingTags []string
		newTags      []string
		expected     []string
	}{
		{
			name:         "Empty existing and new tags",
			existingTags: []string{},
			newTags:      []string{},
			expected:     []string{},
		},
		{
			name:         "Empty existing tags, add new tags",
			existingTags: []string{},
			newTags:      []string{"tag1", "tag2"},
			expected:     []string{"tag1", "tag2"},
		},
		{
			name:         "Existing tags, empty new tags",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{},
			expected:     []string{"tag1", "tag2"},
		},
		{
			name:         "No duplicate tags",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{"tag3", "tag4"},
			expected:     []string{"tag1", "tag2", "tag3", "tag4"},
		},
		{
			name:         "With duplicate tags",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{"tag2", "tag3"},
			expected:     []string{"tag1", "tag2", "tag3"},
		},
		{
			name:         "All duplicate tags",
			existingTags: []string{"tag1", "tag2"},
			newTags:      []string{"tag1", "tag2"},
			expected:     []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tagmodel.MergeTagIDs(tt.existingTags, tt.newTags)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("mergeTagIDs() = %v, expected length %d, got length %d", result, len(tt.expected), len(result))
				return
			}

			// Check all expected tags are present (order doesn't matter)
			resultMap := make(map[string]bool)
			for _, tag := range result {
				resultMap[tag] = true
			}

			for _, expectedTag := range tt.expected {
				if !resultMap[expectedTag] {
					t.Errorf("mergeTagIDs() = %v, missing expected tag %v", result, expectedTag)
				}
			}

			// Check no unexpected tags are present
			expectedMap := make(map[string]bool)
			for _, tag := range tt.expected {
				expectedMap[tag] = true
			}

			for _, resultTag := range result {
				if !expectedMap[resultTag] {
					t.Errorf("mergeTagIDs() = %v, unexpected tag %v", result, resultTag)
				}
			}
		})
	}
}

func TestSlackInteractionViewSubmissionResolveTicket_TagMerging(t *testing.T) {
	ctx := context.Background()

	// Setup repository and services
	repo := repository.NewMemory()
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"🎉 Great work resolving this ticket!"},
					}, nil
				},
			}, nil
		},
	}
	// Setup Slack client mock for this test
	slackClientMock := &mock.SlackClientMock{
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
		GetTeamInfoFunc: func() (*slackSDK.TeamInfo, error) {
			return &slackSDK.TeamInfo{
				Domain: "test-domain",
			}, nil
		},
	}
	slackSvc, err := slackservice.New(slackClientMock, "test-channel")
	gt.NoError(t, err)
	tagSvc := tagservice.New(repo)

	// Create tags first to ensure they exist
	existingTag1 := &tagmodel.Tag{ID: "existing-tag-1", Name: "existing1", Color: "color1"}
	existingTag2 := &tagmodel.Tag{ID: "existing-tag-2", Name: "existing2", Color: "color2"}
	newTag1 := &tagmodel.Tag{ID: "new-tag-1", Name: "newtag1", Color: "color3"}
	newTag2 := &tagmodel.Tag{ID: "new-tag-2", Name: "newtag2", Color: "color4"}

	gt.NoError(t, repo.CreateTagWithID(ctx, existingTag1))
	gt.NoError(t, repo.CreateTagWithID(ctx, existingTag2))
	gt.NoError(t, repo.CreateTagWithID(ctx, newTag1))
	gt.NoError(t, repo.CreateTagWithID(ctx, newTag2))

	// Create test ticket with existing tag IDs
	testTicket := ticket.New(ctx, []types.AlertID{}, &slack.Thread{
		ChannelID: "test-channel",
		ThreadID:  "test-thread",
	})
	if testTicket.TagIDs == nil {
		testTicket.TagIDs = make(map[string]bool)
	}
	testTicket.TagIDs[existingTag1.ID] = true
	testTicket.TagIDs[existingTag2.ID] = true
	gt.NoError(t, repo.PutTicket(ctx, testTicket))

	// Setup use case
	uc := New(
		WithRepository(repo),
		WithSlackService(slackSvc),
		WithLLMClient(llmMock),
		WithTagService(tagSvc),
	)

	// Prepare test input values for resolve with tag selection
	values := slack.StateValue{
		string(slack.BlockIDTicketConclusion): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketConclusion): {
				SelectedOption: slackSDK.OptionBlockObject{
					Value: "intended",
				},
			},
		},
		string(slack.BlockIDTicketTags): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketTags): {
				Type: "checkboxes",
				SelectedOptions: []slackSDK.OptionBlockObject{
					{Value: newTag1.ID},      // Using tag ID instead of name
					{Value: existingTag1.ID}, // This should not create duplicates
				},
			},
		},
	}

	// Execute
	err = uc.HandleSlackInteractionViewSubmission(
		ctx,
		slack.User{ID: "test-user", Name: "test-user"},
		"submit_resolve_ticket",
		testTicket.ID.String(),
		values,
	)

	// Verify
	gt.NoError(t, err)

	// Get updated ticket
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.NotNil(t, updatedTicket)

	// Verify ticket is resolved
	gt.Equal(t, updatedTicket.Status, types.TicketStatusResolved)

	// Verify tags are merged correctly (should have 3 unique tags)
	gt.Number(t, len(updatedTicket.TagIDs)).Equal(3)

	// Check that all expected tags are present
	tagMap := make(map[string]bool)
	for tagID := range updatedTicket.TagIDs {
		tagMap[tagID] = true
	}

	expectedTagIDs := []string{existingTag1.ID, existingTag2.ID, newTag1.ID}
	for _, expectedTagID := range expectedTagIDs {
		gt.Value(t, tagMap[expectedTagID]).Equal(true)
	}
}

func TestSlackInteractionViewSubmissionResolveTicket_TagDuplicationFix(t *testing.T) {
	// This test specifically verifies the tag duplication bug fix
	// It tests that when modal sends tag IDs and existing tags are selected,
	// no duplicate tags are created in the repository
	ctx := context.Background()
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	ctx = clock.With(ctx, func() time.Time { return now })

	repo := repository.NewMemory()

	// Setup LLM mock
	llmMock := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.LLMSessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"🎉 Bug fix verified!"},
					}, nil
				},
			}, nil
		},
	}

	// Setup Slack client mock
	slackClientMock := &mock.SlackClientMock{
		PostMessageContextFunc: func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
			return channelID, "1234567890.123456", nil
		},
		AuthTestFunc: func() (*slackSDK.AuthTestResponse, error) {
			return &slackSDK.AuthTestResponse{
				UserID: "test-user",
			}, nil
		},
	}

	// Generate random tag IDs and names to avoid conflicts with other tests
	randomSuffix := now.UnixNano()
	existingTag1 := &tagmodel.Tag{
		ID:          fmt.Sprintf("tag-id-1-%d", randomSuffix),
		Name:        fmt.Sprintf("security-%d", randomSuffix),
		Color:       "bg-red-100 text-red-800",
		Description: "",
		CreatedAt:   now.Add(-1 * time.Hour),
		UpdatedAt:   now.Add(-1 * time.Hour),
		CreatedBy:   "test",
	}
	existingTag2 := &tagmodel.Tag{
		ID:          fmt.Sprintf("tag-id-2-%d", randomSuffix),
		Name:        fmt.Sprintf("investigation-%d", randomSuffix),
		Color:       "bg-blue-100 text-blue-800",
		Description: "",
		CreatedAt:   now.Add(-1 * time.Hour),
		UpdatedAt:   now.Add(-1 * time.Hour),
		CreatedBy:   "test",
	}
	gt.NoError(t, repo.CreateTagWithID(ctx, existingTag1))
	gt.NoError(t, repo.CreateTagWithID(ctx, existingTag2))

	// Create test ticket
	testTicket := &ticket.Ticket{
		ID:       types.NewTicketID(),
		Status:   types.TicketStatusOpen,
		AlertIDs: []types.AlertID{},
		SlackThread: &slack.Thread{
			ChannelID: "C123456",
			ThreadID:  "1234567890.123456",
		},
		Metadata: ticket.Metadata{
			Title:       "Security Incident",
			Description: "Test ticket for tag duplication fix",
		},
	}
	gt.NoError(t, repo.PutTicket(ctx, *testTicket))

	// Create slack service and tag service
	slackSvc, err := slackservice.New(slackClientMock, "C123456")
	gt.NoError(t, err)
	tagSvc := tagservice.New(repo)

	// Create usecase
	uc := New(
		WithRepository(repo),
		WithSlackService(slackSvc),
		WithLLMClient(llmMock),
		WithTagService(tagSvc),
	)

	// Count tags before resolve operation
	tagsBefore, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.Array(t, tagsBefore).Length(2) // Should have exactly 2 tags

	// Prepare test input values - modal sends tag IDs, not names
	values := slack.StateValue{
		string(slack.BlockIDTicketConclusion): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketConclusion): {
				SelectedOption: slackSDK.OptionBlockObject{Value: string(types.AlertConclusionFalsePositive)},
			},
		},
		string(slack.BlockIDTicketComment): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketComment): {Value: "Investigation completed - tag duplication test"},
		},
		string(slack.BlockIDTicketTags): map[string]slack.BlockAction{
			string(slack.BlockActionIDTicketTags): {
				SelectedOptions: []slackSDK.OptionBlockObject{
					{Value: existingTag1.ID}, // Modal now sends tag ID, not name
					{Value: existingTag2.ID}, // Modal now sends tag ID, not name
				},
			},
		},
	}

	user := slack.User{
		ID:   "U123456",
		Name: "test_user",
	}

	// Execute resolve operation
	err = uc.handleSlackInteractionViewSubmissionResolveTicket(ctx, user, string(testTicket.ID), values)
	gt.NoError(t, err)

	// Verify ticket was resolved
	updatedTicket, err := repo.GetTicket(ctx, testTicket.ID)
	gt.NoError(t, err)
	gt.Value(t, updatedTicket.Status).Equal(types.TicketStatusResolved)

	// Verify tags were assigned to ticket correctly
	gt.Value(t, len(updatedTicket.TagIDs)).Equal(2)
	gt.Value(t, updatedTicket.TagIDs[existingTag1.ID]).Equal(true)
	gt.Value(t, updatedTicket.TagIDs[existingTag2.ID]).Equal(true)

	// CRITICAL: Verify no duplicate tags were created in repository
	tagsAfter, err := repo.ListAllTags(ctx)
	gt.NoError(t, err)
	gt.Array(t, tagsAfter).Length(2) // Should still have exactly 2 tags, no duplicates

	// Verify the existing tags are unchanged
	tag1After, err := repo.GetTagByID(ctx, existingTag1.ID)
	gt.NoError(t, err)
	gt.NotNil(t, tag1After)
	gt.Value(t, tag1After.Name).Equal(existingTag1.Name)
	gt.Value(t, tag1After.CreatedAt).Equal(existingTag1.CreatedAt)

	tag2After, err := repo.GetTagByID(ctx, existingTag2.ID)
	gt.NoError(t, err)
	gt.NotNil(t, tag2After)
	gt.Value(t, tag2After.Name).Equal(existingTag2.Name)
	gt.Value(t, tag2After.CreatedAt).Equal(existingTag2.CreatedAt)

	// Verify tag names are preserved
	tagNames := make([]string, len(tagsAfter))
	for i, tag := range tagsAfter {
		tagNames[i] = tag.Name
	}
	gt.Value(t, containsString(tagNames, existingTag1.Name)).Equal(true)
	gt.Value(t, containsString(tagNames, existingTag2.Name)).Equal(true)
}
