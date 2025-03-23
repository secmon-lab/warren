package llm_test

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/vertexai/genai"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/service/llm"
)

type geminiClient struct {
	model *genai.GenerativeModel
}

func (c *geminiClient) StartChat() interfaces.LLMSession {
	return c.model.StartChat()
}

func (c *geminiClient) SendMessage(ctx context.Context, msg ...genai.Part) (*genai.GenerateContentResponse, error) {
	return c.model.GenerateContent(ctx, msg...)
}

func genGeminiClient(t *testing.T) *geminiClient {
	project, ok := os.LookupEnv("TEST_GEMINI_PROJECT")
	if !ok {
		t.Skip("TEST_GEMINI_PROJECT is not set")
	}
	location, ok := os.LookupEnv("TEST_GEMINI_LOCATION")
	if !ok {
		t.Skip("TEST_GEMINI_LOCATION is not set")
	}
	client, err := genai.NewClient(t.Context(), project, location)
	gt.NoError(t, err)
	geminiModel := client.GenerativeModel("gemini-2.0-flash-exp")
	geminiModel.GenerationConfig.ResponseMIMEType = "application/json"
	return &geminiClient{model: geminiModel}
}

func TestAsk(t *testing.T) {
	geminiClient := genGeminiClient(t)
	ssn := geminiClient.StartChat()

	type resp struct {
		Message string `json:"message"`
	}

	prompt := `Reply for 'Hello, world!'. Format is following:
	{
		"message": "Hello, world!"
	}`

	result, err := llm.Ask[resp](t.Context(), ssn.SendMessage, prompt)
	gt.NoError(t, err)
	gt.NotNil(t, result)
	gt.S(t, result.Message).NotEqual("")
}
