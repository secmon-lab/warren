package session

import (
	"context"
	"log/slog"
	"reflect"
	"time"

	"cloud.google.com/go/vertexai/genai"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

type History struct {
	ID        types.HistoryID `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Contents  Contents        `json:"contents"`
}

type Content struct {
	Role  string
	Parts []Part
}

type Contents []*Content

type Part struct {
	Text string              `json:"text"`
	Blob []byte              `json:"blob"`
	Func *genai.FunctionCall `json:"func"`
}

func NewHistory(ctx context.Context, contents []*genai.Content) *History {
	history := &History{
		ID:        types.NewHistoryID(),
		CreatedAt: clock.Now(ctx),
		Contents:  make(Contents, len(contents)),
	}

	for i, content := range contents {
		parts := make([]Part, 0, len(content.Parts))

		for _, part := range content.Parts {
			switch v := part.(type) {
			case genai.Text:
				parts = append(parts, Part{
					Text: string(v),
				})

			case genai.Blob:
				parts = append(parts, Part{
					Blob: v.Data,
				})

			case genai.FunctionCall:
				parts = append(parts, Part{
					Func: &v,
				})

			default:
				logging.From(ctx).Warn("unknown part type", "type", reflect.TypeOf(v))
			}
		}
		history.Contents[i] = &Content{
			Role:  content.Role,
			Parts: parts,
		}
	}
	return history
}

func (x *History) ToContents() []*genai.Content {
	contents := make([]*genai.Content, len(x.Contents))
	for i, content := range x.Contents {
		parts := make([]genai.Part, len(content.Parts))
		for j, part := range content.Parts {
			switch {
			case part.Text != "":
				parts[j] = genai.Text(part.Text)
			case len(part.Blob) > 0:
				parts[j] = genai.Blob{Data: part.Blob}
			case part.Func != nil:
				parts[j] = part.Func
			}
		}
		contents[i] = &genai.Content{Role: content.Role, Parts: parts}
	}
	return contents
}

func (x *History) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.Any("created_at", x.CreatedAt),
		slog.Any("count_contents", len(x.Contents)),
	)
}
