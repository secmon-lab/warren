package embedding

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/m-mizutani/goerr/v2"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/types/known/structpb"
)

type Gemini struct {
	projectID string
	location  string
	modelName string
}

type GeminiOption func(*Gemini)

func WithLocation(location string) GeminiOption {
	return func(g *Gemini) {
		g.location = location
	}
}

func WithModelName(modelName string) GeminiOption {
	return func(g *Gemini) {
		g.modelName = modelName
	}
}

func NewGemini(projectID string, opts ...GeminiOption) *Gemini {
	gemini := &Gemini{
		projectID: projectID,
		location:  "us-central1",
		modelName: "text-embedding-preview-0815",
	}
	for _, opt := range opts {
		opt(gemini)
	}
	return gemini
}

func (x *Gemini) Embeddings(ctx context.Context, texts []string, dimensionality int) ([][]float32, error) {
	apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", x.location)

	client, err := aiplatform.NewPredictionClient(ctx, option.WithEndpoint(apiEndpoint))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create aiplatform client")
	}
	defer client.Close()

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", x.projectID, x.location, x.modelName)
	instances := make([]*structpb.Value, len(texts))
	for i, text := range texts {
		instances[i] = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"content":   structpb.NewStringValue(text),
				"task_type": structpb.NewStringValue("QUESTION_ANSWERING"),
			},
		})
	}

	params := structpb.NewStructValue(&structpb.Struct{
		Fields: map[string]*structpb.Value{
			"outputDimensionality": structpb.NewNumberValue(float64(dimensionality)),
		},
	})

	req := &aiplatformpb.PredictRequest{
		Endpoint:   endpoint,
		Instances:  instances,
		Parameters: params,
	}
	resp, err := client.Predict(ctx, req)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to predict", goerr.V("endpoint", endpoint), goerr.V("dimensionality", dimensionality))
	}

	embeddings := make([][]float32, len(resp.Predictions))
	for i, prediction := range resp.Predictions {
		values := prediction.GetStructValue().Fields["embeddings"].GetStructValue().Fields["values"].GetListValue().Values
		embeddings[i] = make([]float32, len(values))
		for j, value := range values {
			embeddings[i][j] = float32(value.GetNumberValue())
		}
	}

	return embeddings, nil
}
