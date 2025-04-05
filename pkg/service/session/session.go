package session

import (
	"context"
	"reflect"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"

	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/service/action"
	"github.com/secmon-lab/warren/pkg/service/slack"
)

type Service struct {
	repo   interfaces.Repository
	slack  *slack.Service
	llm    interfaces.LLMClient
	action *action.Service
	ssn    *session.Session
}

func New(repository interfaces.Repository, llmClient interfaces.LLMClient, slackService *slack.Service, actionService *action.Service, ssn *session.Session) *Service {
	svc := &Service{
		repo:   repository,
		llm:    llmClient,
		slack:  slackService,
		action: actionService,
		ssn:    ssn,
	}

	return svc
}

const (
	ctrlCommandExit = "exit"
)

func (x *Service) buildTools(ctx context.Context) []*genai.FunctionDeclaration {
	tools := x.action.Specs()
	tools = append(tools, &genai.FunctionDeclaration{
		Name:        ctrlCommandExit,
		Description: "End the agent session and submit the final conclusion",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"conclusion": {
					Type:        genai.TypeString,
					Description: "The final conclusion in Slack markdown format and in " + lang.From(ctx).Name(),
				},
			},
			Required: []string{"conclusion"},
		},
	})

	return tools
}

func (x *Service) Chat(ctx context.Context, message string) error {
	// Restore history if exists
	histroy, err := x.repo.GetLatestHistory(ctx, x.ssn.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	llmSession := x.llm.StartChat(
		gemini.WithHistory(histroy),
		gemini.WithContentType("text/plain"),
		gemini.WithTools([]*genai.Tool{{
			FunctionDeclarations: x.buildTools(ctx),
		}}),
	)

	alerts, err := x.repo.BatchGetAlerts(ctx, x.ssn.AlertIDs)
	if err != nil {
		return goerr.Wrap(err, "failed to get alerts")
	}

	initPrompt, err := prompt.BuildSessionStartPrompt(ctx, message, alerts)
	if err != nil {
		return goerr.Wrap(err, "failed to build session start prompt")
	}

	parts := []genai.Part{
		genai.Text(initPrompt),
	}
	resp, err := llmSession.SendMessage(ctx, parts...)
	if err != nil {
		return goerr.Wrap(err, "failed to send message")
	}

	willExit := false
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			switch v := part.(type) {
			case genai.Text:
				ctx = msg.NewTrace(ctx, "🐇 %s", string(v))
			case genai.FunctionCall:
				if v.Name == ctrlCommandExit {
					willExit = true
				}
			}
		}
	}

	if willExit {
		return nil
	}

	const maxLoops = 32
	var exit *action_model.Exit
	var results []*action_model.Result

	for i := 0; i < maxLoops && exit == nil; i++ {
		nextPrompt, err := prompt.BuildSessionNextPrompt(ctx, results)
		if err != nil {
			return goerr.Wrap(err, "failed to build session next prompt")
		}

		resp, err := llmSession.SendMessage(ctx, genai.Text(nextPrompt))
		if err != nil {
			return goerr.Wrap(err, "failed to send message")
		}

		results = nil
		for _, candidate := range resp.Candidates {
			resultResp, exitResp, err := x.handleContent(ctx, candidate.Content)
			if err != nil {
				return goerr.Wrap(err, "failed to handle content")
			}

			results = append(results, resultResp...)
			if exitResp != nil {
				exit = exitResp
			}
		}
	}

	newHistory := session.NewHistory(ctx, llmSession.GetHistory())
	if err := x.repo.PutHistory(ctx, x.ssn.ID, newHistory); err != nil {
		return goerr.Wrap(err, "failed to put history")
	}

	if exit == nil {
		msg.Notify(ctx, "😫 Maximum action count exceeded")
		return nil
	}
	msg.Trace(ctx, "Finished session")

	msg.Notify(ctx, "%s", exit.Conclusion)

	return nil
}

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
