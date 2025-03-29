package usecase

/*
type throttleSession struct {
	ssn  interfaces.LLMSession
	last time.Time
	mu   sync.Mutex
}

func (x *throttleSession) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	x.mu.Lock()
	if time.Since(x.last) < 1*time.Second {
		time.Sleep(time.Until(x.last.Add(1 * time.Second)))
	}
	x.last = time.Now()
	defer x.mu.Unlock()

	return x.ssn.SendMessage(ctx, msg...)
}

func (uc *UseCases) RunWorkflow(ctx context.Context, alert alert.Alert) error {
	logger := logging.From(ctx)
	thread := uc.slackService.NewThread(*alert.SlackThread)
	msg.Trace(ctx, "Starting investigation...")

	ssn := &throttleSession{
		ssn:  uc.llmClient.StartChat(),
		last: time.Now(),
	}

	prePrompt, err := prompt.BuildInitPrompt(ctx, alert, uc.actionLimit)
	if err != nil {
		return goerr.Wrap(err, "failed to build init prompt")
	}

	for i := 0; i < uc.actionLimit; i++ {
		actionPrompt, err := planAction(ctx, ssn, prePrompt, uc.actionService)
		if err != nil {
			msg.Trace(ctx, "Failed to plan next action: "+err.Error())
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
			msg.Trace(ctx, msg)
			prePrompt = fmt.Sprintf("The action that you specified previously failed. Please try again. The action is: %s", actionPrompt.Action)
			continue
		}

		logger.Info("action executed", "action", actionResult)
		if err := thread.AttachFile(ctx, actionResult.Message, "result.json", []byte(actionResult.Data)); err != nil {
			msg.Trace(ctx, "Failed to attach file: "+err.Error())
			return goerr.Wrap(err, "failed to attach file")
		}

		prePrompt = fmt.Sprintf("Here is the result of the action.\n%s\n\n```json\n%s\n```", actionResult.Message, actionResult.Data)
	}

	for i := 0; i < uc.findingLimit && alert.Finding == nil; i++ {
		finding, err := uc.buildFinding(ctx, ssn)
		if err != nil {
			msg.Trace(ctx, "Failed to build finding: "+err.Error())
			return goerr.Wrap(err, "failed to build finding")
		}
		logger.Info("finding built", "finding", finding)

		if err := finding.Severity.Validate(); err != nil {
			msg.Trace(ctx, "Failed to validate severity. Retry...")
			continue
		}

		alert.Finding = &alert.Finding{
			Severity:       finding.Severity,
			Summary:        finding.Summary,
			Reason:         finding.Reason,
			Recommendation: finding.Recommendation,
		}
	}

	if alert.Finding == nil {
		msg.Trace(ctx, "Failed to build finding. Exiting...")
		return nil
	}

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return goerr.Wrap(err, "failed to put alert")
	}
	if err := thread.PostFinding(ctx, *alert.Finding); err != nil {
		return goerr.Wrap(err, "failed to post conclusion")
	}
	if err := thread.UpdateAlert(ctx, alert); err != nil {
		return goerr.Wrap(err, "failed to update alert")
	}
	return nil
}

func planAction(ctx context.Context, ssn interfaces.LLMSession, prePrompt string, actionSvc *service.ActionService) (*prompt.ActionPromptResult, error) {
	mainPrompt, err := prompt.BuildActionPrompt(ctx, actionSvc.Spec())
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

func (uc *UseCases) buildFinding(ctx context.Context, ssn interfaces.LLMSession) (*alert.Finding, error) {
	conclusionPrompt, err := prompt.BuildFindingPrompt(ctx)
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

	var result alert.Finding
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, eb.Wrap(err, "failed to unmarshal finding prompt result", goerr.V("text", text))
	}

	return &result, nil
}
*/
