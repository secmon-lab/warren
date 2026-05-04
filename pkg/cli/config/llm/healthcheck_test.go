package llm_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/m-mizutani/gollem"
	gollem_mock "github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
)

func okSession() *gollem_mock.SessionMock {
	return &gollem_mock.SessionMock{
		GenerateFunc: func(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
			return &gollem.Response{Texts: []string{"pong"}}, nil
		},
	}
}

func okClient() *gollem_mock.LLMClientMock {
	s := okSession()
	return &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			return s, nil
		},
		GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
			return [][]float64{{0.1, 0.2}}, nil
		},
	}
}

func failingClient(msg string) *gollem_mock.LLMClientMock {
	return &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			return nil, errors.New(msg)
		},
		GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
			return [][]float64{{0.1}}, nil
		},
	}
}

func TestHealthCheck_AllPass(t *testing.T) {
	primary := llm.NewLLMEntryForTest("primary", "d", "claude", "m", okClient())
	fast := llm.NewLLMEntryForTest("fast", "d", "gemini", "m", okClient())
	emb := okClient()

	reg := llm.NewRegistryForTest("primary", []string{"primary", "fast"},
		map[string]*llm.LLMEntry{"primary": primary, "fast": fast},
		emb,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gt.NoError(t, reg.HealthCheck(ctx))
}

func TestHealthCheck_OneLLMFails(t *testing.T) {
	primary := llm.NewLLMEntryForTest("primary", "d", "claude", "m", okClient())
	fast := llm.NewLLMEntryForTest("fast", "d", "gemini", "m", failingClient("auth fail"))
	emb := okClient()

	reg := llm.NewRegistryForTest("primary", []string{"primary", "fast"},
		map[string]*llm.LLMEntry{"primary": primary, "fast": fast},
		emb,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := reg.HealthCheck(ctx)
	gt.Error(t, err)
}

func TestHealthCheck_AllFailAggregated(t *testing.T) {
	primary := llm.NewLLMEntryForTest("primary", "d", "claude", "m", failingClient("err-primary"))
	fast := llm.NewLLMEntryForTest("fast", "d", "gemini", "m", failingClient("err-fast"))
	emb := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			return nil, errors.New("emb-fail")
		},
		GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
			return nil, errors.New("emb-fail")
		},
	}

	reg := llm.NewRegistryForTest("primary", []string{"primary", "fast"},
		map[string]*llm.LLMEntry{"primary": primary, "fast": fast},
		emb,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := reg.HealthCheck(ctx)
	gt.Error(t, err)
}

func TestHealthCheck_DedupesMainInTask(t *testing.T) {
	// primary appears in both main and task; underlying client should be pinged once.
	calls := 0
	c := &gollem_mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, _ ...gollem.SessionOption) (gollem.Session, error) {
			calls++
			return okSession(), nil
		},
		GenerateEmbeddingFunc: func(_ context.Context, _ int, _ []string) ([][]float64, error) {
			return [][]float64{{0.1}}, nil
		},
	}
	primary := llm.NewLLMEntryForTest("primary", "d", "claude", "m", c)
	emb := okClient()

	reg := llm.NewRegistryForTest("primary", []string{"primary"},
		map[string]*llm.LLMEntry{"primary": primary},
		emb,
	)

	gt.NoError(t, reg.HealthCheck(context.Background()))
	gt.Equal(t, calls, 1)
}
