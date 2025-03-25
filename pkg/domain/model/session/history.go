package session

import (
	"context"
	"log/slog"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

type History struct {
	ID        types.HistoryID `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Role      string          `json:"role"`
	Parts     []Part          `json:"parts"`
}

type Histories []*History

type Part struct {
	Text string              `json:"text"`
	Blob []byte              `json:"blob"`
	Func *genai.FunctionCall `json:"func"`
}

func NewHistories(ctx context.Context, contents []*genai.Content) Histories {
	histories := make(Histories, len(contents))
	for i, content := range contents {
		for _, part := range content.Parts {
			h := &History{
				ID:        types.NewHistoryID(),
				CreatedAt: clock.Now(ctx),
				Role:      content.Role,
			}

			switch v := part.(type) {
			case genai.Text:
				h.Parts = append(h.Parts, Part{
					Text: string(v),
				})

			case genai.Blob:
				h.Parts = append(h.Parts, Part{
					Blob: v.Data,
				})

			case genai.FunctionCall:
				h.Parts = append(h.Parts, Part{
					Func: &v,
				})
			}

			histories[i] = h
		}
	}
	return histories
}

func (x Histories) ToContents() []*genai.Content {
	contents := make([]*genai.Content, len(x))
	for i, history := range x {
		parts := make([]genai.Part, len(history.Parts))
		for j, part := range history.Parts {
			switch {
			case part.Text != "":
				parts[j] = genai.Text(part.Text)
			case len(part.Blob) > 0:
				parts[j] = genai.Blob{Data: part.Blob}
			case part.Func != nil:
				parts[j] = part.Func
			}
		}
		contents[i] = &genai.Content{Role: history.Role, Parts: parts}
	}
	return contents
}

func (x *History) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.Any("created_at", x.CreatedAt),
		slog.Any("role", x.Role),
		slog.Any("count_parts", len(x.Parts)),
	)
}
