package service

import (
	"context"
	"math"

	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

func GenerateAlertListMeta(ctx context.Context, list model.AlertList, llmClient interfaces.LLMClient) (*prompt.MetaListPromptResult, error) {
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

func CalcMaxSimilarity(alerts []model.Alert) float64 {
	max := 0.0

	for i, a := range alerts {
		for j := i + 1; j < len(alerts); j++ {
			if CosineSimilarity(a.Embedding, alerts[j].Embedding) > max {
				max = CosineSimilarity(a.Embedding, alerts[j].Embedding)
			}
		}
	}

	return max
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var magnitudeA, magnitudeB float64

	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		magnitudeA += float64(a[i]) * float64(a[i])
		magnitudeB += float64(b[i]) * float64(b[i])
	}

	return dotProduct / (math.Sqrt(magnitudeA) * math.Sqrt(magnitudeB))
}
