package embedding_test

import (
	"context"
	_ "embed"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/embedding"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func genClient(t *testing.T) *embedding.Gemini {
	vars := test.NewEnvVars(t, "TEST_GEMINI_PROJECT_ID", "TEST_GEMINI_LOCATION")
	return embedding.NewGemini(
		vars.Get("TEST_GEMINI_PROJECT_ID"),
		embedding.WithLocation(vars.Get("TEST_GEMINI_LOCATION")),
		embedding.WithModelName("text-embedding-004"),
	)
}
func TestGemini(t *testing.T) {
	gemini := genClient(t)

	embeddings, err := gemini.Embeddings(context.Background(), []string{"Hello, world!"}, 5)
	gt.NoError(t, err)
	gt.A(t, embeddings).Length(1).At(0, func(t testing.TB, v []float32) {
		gt.A(t, v).Length(5)
	})
}

//go:embed testdata/guardduty.json
var guarddutyJSON []byte

func TestGemini_Embeddings_GuardDuty(t *testing.T) {
	gemini := genClient(t)

	embeddings, err := gemini.Embeddings(context.Background(), []string{string(guarddutyJSON)}, 256)
	gt.NoError(t, err)
	gt.A(t, embeddings).Length(1).At(0, func(t testing.TB, v []float32) {
		gt.A(t, v).Length(256)
	})
}
