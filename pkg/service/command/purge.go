package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/service/command/core"
	slackSDK "github.com/slack-go/slack"
)

func purge(ctx context.Context, clients *core.Clients, msg *slack.Message, input string) (any, error) {
	thread := clients.Thread().Entity()

	// Get protected message IDs from DB
	protectedIDs := make(map[string]bool)

	// Get all alerts in this thread and protect their message IDs
	alerts, err := clients.Repo().GetAlertsByThread(ctx, *thread)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get alerts by thread")
	}
	for _, alert := range alerts {
		if alert.SlackMessageID != "" {
			protectedIDs[alert.SlackMessageID] = true
		}
	}

	// Get ticket if exists and add its message ID
	ticket, err := clients.Repo().GetTicketByThread(ctx, *thread)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get ticket by thread")
	}
	if ticket != nil && ticket.SlackMessageID != "" {
		protectedIDs[ticket.SlackMessageID] = true
	}

	// Get bot user ID to identify bot's own messages
	authResp, err := clients.SlackClient().AuthTest()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get bot info")
	}
	botUserID := authResp.UserID

	// Fetch all messages in thread using GetConversationReplies
	params := &slackSDK.GetConversationRepliesParameters{
		ChannelID: thread.ChannelID,
		Timestamp: thread.ThreadID,
		Limit:     1000,
	}

	messages, _, _, err := clients.SlackClient().GetConversationRepliesContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conversation replies")
	}

	// Collect messages to delete
	var messagesToDelete []string
	for _, message := range messages {
		// Skip if not a bot message
		if message.User != botUserID && message.BotID == "" {
			continue
		}

		// Skip if this is a protected message
		if protectedIDs[message.Timestamp] {
			continue
		}

		messagesToDelete = append(messagesToDelete, message.Timestamp)
	}

	if len(messagesToDelete) == 0 {
		return "削除対象のメッセージはありません", nil
	}

	// Delete messages concurrently using worker pool pattern
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	deletedCount := 0
	var errors []error

	for _, timestamp := range messagesToDelete {
		wg.Add(1)
		go func(ts string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			if _, _, err := clients.SlackClient().DeleteMessageContext(ctx, thread.ChannelID, ts); err != nil {
				mu.Lock()
				errors = append(errors, goerr.Wrap(err, "failed to delete message", goerr.V("timestamp", ts)))
				mu.Unlock()
			} else {
				mu.Lock()
				deletedCount++
				mu.Unlock()
			}
		}(timestamp)
	}

	wg.Wait()

	// Report results
	if len(errors) > 0 {
		return nil, goerr.Wrap(errors[0], "failed to delete some messages",
			goerr.V("deleted_count", deletedCount),
			goerr.V("error_count", len(errors)),
		)
	}

	return fmt.Sprintf("%d件のメッセージを削除しました", deletedCount), nil
}
