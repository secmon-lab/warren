package usecase

/*
func (uc *UseCases) HandleAlertWithAuth(ctx context.Context, schema string, alertData any) ([]*alert.Alert, error) {
	authCtx := auth.BuildContext(ctx)

	policyClient, err := uc.policyService.NewClient(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create policy client")
	}

	var result model.PolicyAuth
	if err := policyClient.Query(ctx, "data.auth", authCtx, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("auth", authCtx))
	}

	if !result.Allow {
		return nil, goerr.New("unauthorized", goerr.V("auth", authCtx))
	}

	return uc.HandleAlert(ctx, schema, alertData, policyClient)
}

func (uc *UseCases) HandleAlert(ctx context.Context, schema string, alertData any, policyClient interfaces.PolicyClient) ([]*alert.Alert, error) {
	logger := logging.From(ctx)

	var result model.PolicyResult
	if err := policyClient.Query(ctx, "data.alert."+schema, alertData, &result); err != nil {
		return nil, goerr.Wrap(err, "failed to query policy", goerr.V("schema", schema), goerr.V("alert", alertData))
	}

	logger.Info("policy query result", "input", alertData, "output", result)

	var results []*alert.Alert
	for _, a := range result.Alert {
		alert := model.NewAlert(ctx, schema, a)
		if alert.Data == nil {
			alert.Data = alertData
		}

		newAlert, err := uc.handleAlert(ctx, alert)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to handle alert", goerr.V("alert", a))
		}
		results = append(results, newAlert)
	}

	return results, nil
}

func (uc *UseCases) handleAlert(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	logger := logging.From(ctx)

	newAlert, err := uc.generateAlertMetadata(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate alert metadata")
	}
	alert = *newAlert

	if uc.embeddingClient != nil {
		rawData, err := json.Marshal(alert.Data)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to marshal alert data")
		}
		embedding, err := uc.embeddingClient.Embeddings(ctx, []string{string(rawData)}, 256)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to embed alert data")
		}
		if len(embedding) == 0 {
			return nil, goerr.New("failed to embed alert data")
		}
		logger.Info("alert embedding", "embedding", embedding[0], "alert", alert.ID)
		alert.Embedding = embedding[0]
	}

	// Check if the alert is similar to any existing alerts
	// NOTE: Disable similarity merger for now


	// Post new alert to Slack and save the alert with Slack channel and message ID
	thread, err := uc.slackService.PostAlert(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to post alert", goerr.V("alert", alert))
	}
	alert.SlackThread = &slack.SlackThread{
		ChannelID: thread.ChannelID(),
		ThreadID:  thread.ThreadID(),
	}

	if err := uc.repository.PutAlert(ctx, alert); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert", goerr.V("alert", alert))
	}
	logger.Info("alert created", "alert", alert)

	return &alert, nil
}

func (uc *UseCases) generateAlertMetadata(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	logger := logging.From(ctx)
	p, err := prompt.BuildMetaPrompt(ctx, alert)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build meta prompt")
	}

	ssn := uc.llmClient.StartChat()

	var result *prompt.MetaPromptResult
	for i := 0; i < 3 && result == nil; i++ {
		result, err = service.AskChat[prompt.MetaPromptResult](ctx, ssn, p)
		if err != nil {
			if goerr.HasTag(err, errs.TagInvalidLLMResponse) {
				logger.Warn("invalid LLM response, retry to generate alert metadata", "error", err)
				p = fmt.Sprintf("invalid format, please try again: %s", err.Error())
				continue
			}
			return nil, goerr.Wrap(err, "failed to ask chat")
		}
	}
	if result == nil {
		return nil, goerr.New("failed to generate alert metadata")
	}

	if alert.Title == "" {
		alert.Title = result.Title
	}

	if alert.Description == "" {
		alert.Description = result.Description
	}

	for _, resAttr := range result.Attrs {
		found := false
		for _, aAttr := range alert.Attributes {
			if aAttr.Value == resAttr.Value {
				found = true
				break
			}
		}
		if !found {
			resAttr.Auto = true
			alert.Attributes = append(alert.Attributes, resAttr)
		}
	}

	return &alert, nil
}

func (uc *UseCases) findSimilarAlert(ctx context.Context, alert alert.Alert) (*alert.Alert, error) {
	oldest := alert.CreatedAt.Add(-24 * time.Hour)
	alerts, err := uc.repository.GetLatestAlerts(ctx, oldest, 100)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to fetch latest alerts")
	}
	unresolvedAlerts := make([]alert.Alert, 0, len(alerts))
	for _, a := range alerts {
		if a.Status != alert.StatusResolved {
			unresolvedAlerts = append(unresolvedAlerts, a)
		}
	}

	p, err := prompt.BuildAggregatePrompt(ctx, alert, unresolvedAlerts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to build aggregate prompt")
	}

	ssn := uc.llmClient.StartChat()
	result, err := service.AskChat[prompt.AggregatePromptResult](ctx, ssn, p)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to ask chat")
	}

	if result == nil || result.AlertID == "" {
		return nil, nil
	}

	alertID := alert.AlertID(result.AlertID)

	for _, candidate := range alerts {
		if candidate.ID == alertID {
			return &candidate, nil
		}
	}

	return nil, nil
}
*/
