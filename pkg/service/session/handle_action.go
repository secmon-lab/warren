package session

import (
	"context"
	"reflect"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (x *Service) handleContent(ctx context.Context, content *genai.Content) ([]*action_model.Result, *action_model.Exit, error) {
	var results []*action_model.Result
	var exit *action_model.Exit

	for _, part := range content.Parts {
		switch v := part.(type) {
		case genai.Text:
			note := session.NewNote(x.ssn.ID, string(v))
			if err := x.repo.PutNote(ctx, note); err != nil {
				return nil, exit, goerr.Wrap(err, "failed to put note")
			}

		case genai.FunctionCall:
			if v.Name == ctrlCommandExit {
				exit = &action_model.Exit{
					Conclusion: v.Args["conclusion"].(string),
				}
				continue
			}

			resp, err := x.action.Execute(ctx, string(v.Name), v.Args)
			if err != nil {
				return nil, exit, goerr.Wrap(err, "failed to execute action", goerr.V("call", v))
			}
			results = append(results, resp)

		default:
			logging.From(ctx).Warn("unknown content type", "type", reflect.TypeOf(v))
		}
	}

	return results, exit, nil
}
