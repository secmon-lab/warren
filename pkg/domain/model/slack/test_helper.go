package slack

// Test helpers for creating slack.Message instances in tests

// NewTestMessage creates a Message for testing purposes.
// If threadID is empty, the message is treated as the first message in a thread (id becomes the threadID).
func NewTestMessage(channelID, threadID, teamID, messageID, userID, text string) Message {
	return Message{
		id:       messageID,
		channel:  channelID,
		threadID: threadID,
		teamID:   teamID,
		user: User{
			ID:   userID,
			Name: userID,
		},
		msg: text,
		ts:  messageID,
	}
}

// NewTestMessageInThread creates a Message that is part of an existing thread.
// This is a convenience function for the common case of creating a reply message.
func NewTestMessageInThread(thread Thread, userID, text string) Message {
	return Message{
		id:       "M" + text, // Simple ID generation for tests
		channel:  thread.ChannelID,
		threadID: thread.ThreadID,
		teamID:   thread.TeamID,
		user: User{
			ID:   userID,
			Name: userID,
		},
		msg: text,
		ts:  "M" + text,
	}
}
