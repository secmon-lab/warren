package session

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

const (
	ctrlActionResponse = "response"
)

func (x *Service) buildStartTools() []*genai.FunctionDeclaration {
	tools := x.action.Specs()
	tools = append(tools, &genai.FunctionDeclaration{
		Name:        ctrlActionResponse,
		Description: "Respond to the user's message",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"message": {
					Type:        genai.TypeString,
					Description: "The message to respond to the user",
				},
				"continue": {
					Type:        genai.TypeBoolean,
					Description: "Whether to continue the session to analyze the alerts. If you choose true, the session will continue and you can take actions.",
				},
			},
			Required: []string{"message", "continue"},
		},
	})

	return tools
}

func (x *Service) handleStart(ctx context.Context, resp *genai.GenerateContentResponse) (context.Context, bool, error) {
	var response string
	var contSsn bool

	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			switch v := part.(type) {
			case genai.Text:
				note := session.NewNote(x.ssn.ID, string(v))
				if err := x.repo.PutNote(ctx, note); err != nil {
					return ctx, contSsn, goerr.Wrap(err, "failed to put note")
				}

			case genai.FunctionCall:
				if v.Name == ctrlActionResponse {
					if nil != v.Args["message"] {
						response = v.Args["message"].(string)
					}
					if nil != v.Args["continue"] {
						contSsn = v.Args["continue"].(bool)
					}
				}
			}
		}
	}

	if contSsn {
		msg.Notify(ctx, "🐇 %s", response)
	} else {
		ctx = msg.NewTrace(ctx, "🐇 %s", response)
	}

	return ctx, contSsn, nil
}
