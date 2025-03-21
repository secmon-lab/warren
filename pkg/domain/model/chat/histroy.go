package chat

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type History struct {
	ID        types.HistoryID `json:"id"`
	Thread    slack.Thread    `json:"thread"`
	CreatedBy slack.User      `json:"created_by"`
	CreatedAt time.Time       `json:"created_at"`
	Contents  []Content       `json:"contents"`
}

type Content struct {
	Role string   `json:"role"`
	Text []string `json:"text"`
}

func NewHistory(ctx context.Context, thread slack.Thread, createdBy slack.User, contents []*genai.Content) *History {
	contentData := make([]Content, len(contents))
	for i, content := range contents {
		var textSet []string
		for _, part := range content.Parts {
			if text, ok := part.(genai.Text); ok {
				textSet = append(textSet, string(text))
			}
		}
		contentData[i] = Content{
			Role: content.Role,
			Text: textSet,
		}
	}

	return &History{
		ID:        types.NewHistoryID(),
		Thread:    thread,
		CreatedAt: clock.Now(ctx),
		CreatedBy: createdBy,
		Contents:  contentData,
	}
}

func (x *History) ToContents() []*genai.Content {
	contents := make([]*genai.Content, len(x.Contents))
	for i, content := range x.Contents {
		parts := make([]genai.Part, len(content.Text))
		for j, text := range content.Text {
			parts[j] = genai.Text(text)
		}
		contents[i] = &genai.Content{Role: content.Role, Parts: parts}
	}
	return contents
}

func (x *History) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.Any("thread", x.Thread),
		slog.Any("created_by", x.CreatedBy),
		slog.Any("created_at", x.CreatedAt),
		slog.Any("contents", x.Contents),
	)
}
