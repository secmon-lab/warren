package usecase

import "context"

type HandlePromptInput = handlePromptInput

func HandlePrompt(ctx context.Context, input HandlePromptInput) error {
	return handlePrompt(ctx, input)
}
