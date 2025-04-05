package session

import (
	"context"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/gemini"
	"github.com/secmon-lab/warren/pkg/domain/model/lang"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
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

func (x *Service) buildActionTools(ctx context.Context) []*genai.FunctionDeclaration {
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

	// If history is empty, need to initialize the session
	llmSession := x.llm.StartChat(
		gemini.WithHistory(histroy),
		gemini.WithContentType("text/plain"),
		gemini.WithTools([]*genai.Tool{{
			FunctionDeclarations: x.buildStartTools(),
		}}),
	)

	defer func() {
		newHistory := session.NewHistory(ctx, llmSession.GetHistory())
		if err := x.repo.PutHistory(ctx, x.ssn.ID, newHistory); err != nil {
			errs.Handle(ctx, err)
			msg.Notify(ctx, "⚠️ Failed to save chat history")
		}
	}()

	var parts []genai.Part
	if histroy == nil {
		alerts, err := x.repo.BatchGetAlerts(ctx, x.ssn.AlertIDs)
		if err != nil {
			return goerr.Wrap(err, "failed to get alerts")
		}

		initPrompt, err := prompt.BuildSessionInitPrompt(ctx, alerts)
		if err != nil {
			return goerr.Wrap(err, "failed to build session start prompt")
		}

		parts = append(parts, genai.Text(initPrompt))
	}

	parts = append(parts, genai.Text(message))
	initResp, err := llmSession.SendMessage(ctx, parts...)
	if err != nil {
		return goerr.Wrap(err, "failed to send message")
	}

	ctx, contSsn, err := x.handleStart(ctx, initResp)
	if err != nil {
		return goerr.Wrap(err, "failed to handle start")
	}
	if !contSsn {
		return nil
	}

	// Start a new chat session with the current history
	newHist := session.NewHistory(ctx, llmSession.GetHistory())
	llmSession = x.llm.StartChat(
		gemini.WithHistory(newHist),
		gemini.WithContentType("text/plain"),
		gemini.WithTools([]*genai.Tool{{
			FunctionDeclarations: x.buildActionTools(ctx),
		}}),
	)

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
			return goerr.Wrap(err, "failed to send message", goerr.V("prompt", nextPrompt))
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

	if exit == nil {
		msg.Notify(ctx, "😫 Maximum action count exceeded")
		return nil
	}
	msg.Trace(ctx, "Finished")

	msg.Notify(ctx, "🐇 %s", exit.Conclusion)

	return nil
}
