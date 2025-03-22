package service

/*
func GenerateAlertListMeta(ctx context.Context, list alert.List, llmClient interfaces.LLMClient) (*prompt.MetaListPromptResult, error) {
	p, err := prompt.BuildMetaListPrompt(ctx, list)
	if err != nil {
		return nil, err
	}

	const (
		listMetaThreshold = 0.95
		maxRetryCount     = 3
	)

	if listMetaThreshold > CalcMaxSimilarity(list.Alerts) {
		thread.Reply(ctx, "🤖 Alert list is too similar to other alert lists. Skipping meta data generation ("+list.ID.String()+")")
		return nil, nil
	}

	var result *prompt.MetaListPromptResult
	for range maxRetryCount {
		thread.Reply(ctx, "🤖 Generating meta data of alert list... ("+list.ID.String()+")")
		resp, err := AskPrompt[prompt.MetaListPromptResult](ctx, llmClient, p)

		if err == nil {
			result = resp
			break
		}

		thread.Reply(ctx, "💥 Failed to generate meta data of alert list: "+err.Error())
		p = "Invalid result. Please retry: " + err.Error()
	}

	return result, nil
}
*/
