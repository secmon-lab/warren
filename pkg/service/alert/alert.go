package alert

import (
	"context"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

func GenerateAlertListMeta(ctx context.Context, list alert.List, llmClient interfaces.LLMClient) (*prompt.MetaListPromptResult, error) {
	p, err := prompt.BuildMetaListPrompt(ctx, list)
	if err != nil {
		return nil, err
	}

	const (
		listMetaThreshold = 0.95
		maxRetryCount     = 3
	)

	if listMetaThreshold > list.Alerts.MaxSimilarity() {
		msg.Trace(ctx, "🤖 Alert list is too similar to other alert lists. Skipping meta data generation (%s)", list.ID.String())
		return nil, nil
	}

	var result *prompt.MetaListPromptResult
	for range maxRetryCount {
		ctx = msg.Trace(ctx, "🤖 Generating meta data of alert list... (%s)", list.ID.String())
		resp, err := llm.Ask[prompt.MetaListPromptResult](ctx, llmClient, p)

		if err == nil {
			result = resp
			break
		}

		ctx = msg.Trace(ctx, "💥 Failed to generate meta data of alert list: %s", err.Error())
		p = "Invalid result. Please retry: " + err.Error()
	}

	return result, nil
}
