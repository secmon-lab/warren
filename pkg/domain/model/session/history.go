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
	ID        types.HistoryID `firestore:"id" json:"id"`
	SessionID types.SessionID `firestore:"session_id" json:"session_id"`
	CreatedAt time.Time       `firestore:"created_at" json:"created_at"`

	// Contents is not stored in Firestore because it will be too large.
	Contents Contents `firestore:"-" json:"contents"`
}

type Content struct {
	Role  string
	Parts []Part
}

type Contents []*Content

type Part struct {
	Text     string                  `json:"text"`
	Blob     []byte                  `json:"blob"`
	FuncCall *genai.FunctionCall     `json:"func_call"`
	FuncResp *genai.FunctionResponse `json:"func_resp"`
}

func NewHistory(ctx context.Context, sessionID types.SessionID, contents []*genai.Content) *History {
	history := &History{
		ID:        types.NewHistoryID(),
		SessionID: sessionID,
		CreatedAt: clock.Now(ctx),
		Contents:  make(Contents, 0, len(contents)),
	}

	for _, content := range contents {
		var parts []Part

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

			case *genai.FunctionCall:
				parts = append(parts, Part{
					FuncCall: v,
				})

			case *genai.FunctionResponse:
				parts = append(parts, Part{
					FuncResp: v,
				})

			default:
				logging.From(ctx).Warn("unknown part type", "type", reflect.TypeOf(v).String())
			}
		}

		if len(parts) > 0 {
			history.Contents = append(history.Contents, &Content{
				Role:  content.Role,
				Parts: parts,
			})
		}
	}

	return history
}

func (x *History) ToContents() []*genai.Content {
	contents := make([]*genai.Content, 0, len(x.Contents))
	for _, content := range x.Contents {
		if content == nil {
			continue
		}

		parts := make([]genai.Part, len(content.Parts))
		for j, part := range content.Parts {
			switch {
			case part.Text != "":
				parts[j] = genai.Text(part.Text)
			case len(part.Blob) > 0:
				parts[j] = genai.Blob{Data: part.Blob}
			case part.FuncCall != nil:
				parts[j] = part.FuncCall
			case part.FuncResp != nil:
				parts[j] = part.FuncResp
			}
		}
		contents = append(contents, &genai.Content{Role: content.Role, Parts: parts})
	}
	return contents
}

func (x *History) LogValue() slog.Value {
	if x == nil {
		return slog.AnyValue(nil)
	}

	return slog.GroupValue(
		slog.String("id", x.ID.String()),
		slog.Any("created_at", x.CreatedAt),
		slog.Any("count_contents", len(x.Contents)),
	)
}
