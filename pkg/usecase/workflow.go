package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/utils/logging"
)

func (uc *UseCases) RunWorkflow(ctx context.Context, alert model.Alert) error {
	logger := logging.From(ctx)
	thread := uc.slackService.NewThread(alert)
	if err := thread.Reply(ctx, "Starting investigation..."); err != nil {
		return goerr.Wrap(err, "failed to reply to slack")
	}

	ssn := uc.geminiStartChat()

	prePrompt, err := prompt.BuildInitPrompt(alert)
	if err != nil {
		return goerr.Wrap(err, "failed to build init prompt")
	}

	for i := 0; i < uc.actionLimit; i++ {
		actionPrompt, err := planAction(ctx, ssn, prePrompt, uc.actionService)
		if err != nil {
			return goerr.Wrap(err, "failed to plan action")
		}
		logger.Info("action planned", "action", actionPrompt)

		if err := thread.PostNextAction(ctx, *actionPrompt); err != nil {
			return goerr.Wrap(err, "failed to post next action")
		}

		if actionPrompt.Action == "done" {
			break
		}

		actionResult, err := uc.actionService.Execute(ctx, thread, actionPrompt.Action, ssn, actionPrompt.Args)
		if err != nil {
			logger.Error("Action failed", "error", err, "action", actionPrompt.Action, "args", actionPrompt.Args)

			msg := fmt.Sprintf("Action failed: %s. Retry...", err.Error())
			if err := thread.Reply(ctx, msg); err != nil {
				return goerr.Wrap(err, "failed to reply to slack")
			}
			prePrompt = fmt.Sprintf("The action that you specified previously failed. Please try again. The action is: %s", actionPrompt.Action)
			continue
		}

		logger.Info("action executed", "action", actionResult)
		if err := thread.AttachFile(ctx, actionResult.Message, "result.json", []byte(actionResult.Data)); err != nil {
			return goerr.Wrap(err, "failed to attach file")
		}

		prePrompt = fmt.Sprintf("Here is the result of the action.\n%s\n\n```json\n%s\n```", actionResult.Message, actionResult.Data)
	}

	for i := 0; i < uc.findingLimit; i++ {
		finding, err := uc.buildFinding(ctx, ssn)
		if err != nil {
			return goerr.Wrap(err, "failed to build finding")
		}
		logger.Info("finding built", "finding", finding)

		if err := finding.Severity.Validate(); err != nil {
			if err := thread.Reply(ctx, "Failed to validate severity. Retry..."); err != nil {
				return goerr.Wrap(err, "failed to reply to slack")
			}
			continue
		}

		alert.Finding = &model.AlertFinding{
			Severity:       finding.Severity,
			Summary:        finding.Summary,
			Reason:         finding.Reason,
			Recommendation: finding.Recommendation,
		}
		break
	}

	if alert.Finding == nil {
		if err := thread.Reply(ctx, "Failed to build finding. Retry..."); err != nil {
			return goerr.Wrap(err, "failed to reply to slack")
		}
		return nil
	}

	if err := thread.PostFinding(ctx, *alert.Finding); err != nil {
		return goerr.Wrap(err, "failed to post conclusion")
	}

	return nil
}

func planAction(ctx context.Context, ssn interfaces.GenAIChatSession, prePrompt string, actionSvc *service.ActionService) (*prompt.ActionPromptResult, error) {
	mainPrompt, err := prompt.BuildActionPrompt(actionSvc.Spec())
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build action prompt")
	}

	resp, err := ssn.SendMessage(ctx, genai.Text(prePrompt), genai.Text(mainPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}
	eb := goerr.NewBuilder(goerr.V("prompt", mainPrompt), goerr.V("response", resp))

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, eb.New("no action prompt result")
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok || text == "" {
		return nil, eb.New("no action prompt result")
	}

	var result prompt.ActionPromptResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal action prompt result", goerr.V("text", text))
	}

	return &result, nil
}

func (uc *UseCases) buildFinding(ctx context.Context, ssn interfaces.GenAIChatSession) (*model.AlertFinding, error) {
	conclusionPrompt, err := prompt.BuildFindingPrompt()
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build conclusion prompt")
	}

	eb := goerr.NewBuilder(goerr.V("prompt", conclusionPrompt))

	resp, err := ssn.SendMessage(ctx, genai.Text(conclusionPrompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to send message")
	}
	eb = eb.With(goerr.V("response", resp))

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, eb.New("no conclusion prompt result")
	}

	text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text)
	if !ok || text == "" {
		return nil, eb.New("no conclusion prompt result")
	}

	var result model.AlertFinding
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal finding prompt result", goerr.V("text", text))
	}

	return &result, nil
}
