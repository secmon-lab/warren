package session

import (
	"context"
	"reflect"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func (x *Service) handleCandidates(ctx context.Context, candidates []*genai.Candidate) ([]*action_model.Result, error) {
	logger := logging.From(ctx)

	var results []*action_model.Result

	for idx, candidate := range candidates {
		logger.Debug("candidate", "idx", idx, "contentes", candidate.Content)

		for _, part := range candidate.Content.Parts {
			switch v := part.(type) {
			case genai.Text:
				msg.Notify(ctx, "🐰 %s", string(v))

			case genai.FunctionCall:
				if v.Name == ctrlCommandExit {
					result := &action_model.Result{
						Name: string(v.Name),
						Data: v.Args,
					}
					results = append(results, result)
					continue
				}

				logger.Debug("function call", "name", v.Name, "args", v.Args)

				resp, err := x.action.Execute(ctx, string(v.Name), v.Args)
				if err != nil {
					return nil, goerr.Wrap(err, "failed to execute action", goerr.V("call", v))
				}
				results = append(results, resp)

			default:
				logging.From(ctx).Warn("unknown content type", "type", reflect.TypeOf(v))
			}
		}
	}

	return results, nil
}
