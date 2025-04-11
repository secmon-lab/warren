package session

import (
	"context"
	"fmt"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	action_model "github.com/secmon-lab/warren/pkg/domain/model/action"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
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

func New(repository interfaces.Repository, llmClient interfaces.LLMClient, actionService *action.Service, ssn *session.Session) *Service {
	svc := &Service{
		repo:   repository,
		llm:    llmClient,
		action: actionService,
		ssn:    ssn,
	}

	return svc
}

const (
	ctrlCommandExit = "exit"
)

func (x *Service) buildActionTools(ctx context.Context) []*genai.FunctionDeclaration {
	tools := x.action.Tools()
	tools = append(tools, &genai.FunctionDeclaration{
		Name:        ctrlCommandExit,
		Description: "Finish the agent session and submit the final conclusion",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"conclusion": {
					Type:        genai.TypeString,
					Description: fmt.Sprintf("If you need, you can leave a final conclusion in Slack markdown format and in %s", lang.From(ctx).Name()),
				},
			},
		},
	})

	return tools
}

func (x *Service) Chat(ctx context.Context, message string) error {
	logger := logging.From(ctx)

	// Restore history if exists
	histroy, err := x.repo.GetLatestHistory(ctx, x.ssn.ID)
	if err != nil {
		return goerr.Wrap(err, "failed to get latest history")
	}

	logger.Debug("got history", "history", histroy)

	// If history is empty, need to initialize the session
	llmSession := x.llm.StartChat(
		gemini.WithHistory(histroy),
		gemini.WithContentType("text/plain"),
		gemini.WithTools([]*genai.Tool{{
			FunctionDeclarations: x.buildActionTools(ctx),
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

	const maxLoops = 32
	var exit *action_model.Result

	for i := 0; i < maxLoops && exit == nil; i++ {
		resp, err := llmSession.SendMessage(ctx, parts...)
		if err != nil {
			return goerr.Wrap(err, "failed to send message")
		}

		actionResult, err := x.handleCandidates(ctx, resp.Candidates)
		if err != nil {
			return goerr.Wrap(err, "failed to handle content")
		}
		if len(actionResult) == 0 {
			return nil
		}

		parts = nil
		for _, result := range actionResult {
			parts = append(parts, genai.FunctionResponse{
				Name:     result.Name,
				Response: result.Data,
			})
		}

		for _, c := range resp.Candidates {
			if c.FinishReason != genai.FinishReasonStop {
				msg.Notify(ctx, "💥 %s", lookupFinishReasonDescription(c.FinishReason))
			}
		}

		for _, result := range actionResult {
			if result.Name == ctrlCommandExit {
				exit = result
				break
			}
		}
	}

	if exit == nil {
		msg.Notify(ctx, "😫 Maximum action count exceeded")
		return nil
	}

	if conclusion, ok := exit.Data["conclusion"]; ok {
		msg.Notify(ctx, "🐰 %s", conclusion)
	}

	return nil
}

var finishReasonDescriptions = map[genai.FinishReason]string{
	genai.FinishReasonUnspecified:           "The finish reason is unspecified.",
	genai.FinishReasonStop:                  "The model naturally stopped or reached a provided stop sequence.",
	genai.FinishReasonMaxTokens:             "The generation stopped because the maximum number of tokens was reached.",
	genai.FinishReasonSafety:                "The output was flagged for safety concerns and was stopped.",
	genai.FinishReasonRecitation:            "The output was stopped due to unauthorized citations.",
	genai.FinishReasonOther:                 "The generation was stopped for an unspecified other reason.",
	genai.FinishReasonBlocklist:             "The response was flagged for terms included in a blocklist.",
	genai.FinishReasonProhibitedContent:     "The response contained prohibited content and was stopped.",
	genai.FinishReasonSpii:                  "The response was flagged for containing sensitive personal information (SPII).",
	genai.FinishReasonMalformedFunctionCall: "The function call generated by the model was invalid or malformed.",
}

func lookupFinishReasonDescription(reason genai.FinishReason) string {
	if desc, ok := finishReasonDescriptions[reason]; ok {
		return desc
	}
	return "Unknown finish reason"
}
