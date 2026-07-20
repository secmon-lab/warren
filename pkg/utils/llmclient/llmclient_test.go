package llmclient_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/gollem-dev/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/utils/llmclient"
)

// recordingClient returns a mock LLMClient that captures the SessionOptions it
// receives, so tests can assert what the decorator forwarded.
func recordingClient(session gollem.Session, err error) (*mock.LLMClientMock, *[]gollem.SessionOption) {
	var got []gollem.SessionOption
	client := &mock.LLMClientMock{
		NewSessionFunc: func(_ context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			got = options
			return session, err
		},
	}
	return client, &got
}

func TestWithPromptCacheEnablesCaching(t *testing.T) {
	client, got := recordingClient(&mock.SessionMock{}, nil)

	wrapped := llmclient.WithPromptCache(client, true)
	_, err := wrapped.NewSession(context.Background(), gollem.WithSessionSystemPrompt("system"))
	gt.NoError(t, err)

	// The caller's own options must survive alongside the injected one.
	cfg := gollem.NewSessionConfig(*got...)
	gt.True(t, cfg.PromptCache())
	gt.Equal(t, cfg.SystemPrompt(), "system")
}

func TestWithPromptCacheDisabledReturnsClientUnchanged(t *testing.T) {
	client, got := recordingClient(&mock.SessionMock{}, nil)

	wrapped := llmclient.WithPromptCache(client, false)

	// Disabled must not wrap at all, so no option is injected.
	gt.Equal(t, wrapped, gollem.LLMClient(client))

	_, err := wrapped.NewSession(context.Background())
	gt.NoError(t, err)
	cfg := gollem.NewSessionConfig(*got...)
	gt.False(t, cfg.PromptCache())
}

func TestWithPromptCacheForwardsSessionAndError(t *testing.T) {
	t.Run("returns the wrapped client's session", func(t *testing.T) {
		want := &mock.SessionMock{}
		client, _ := recordingClient(want, nil)

		got, err := llmclient.WithPromptCache(client, true).NewSession(context.Background())
		gt.NoError(t, err)
		gt.Equal(t, got, gollem.Session(want))
	})

	t.Run("propagates the wrapped client's error", func(t *testing.T) {
		wantErr := errors.New("boom")
		client, _ := recordingClient(nil, wantErr)

		got, err := llmclient.WithPromptCache(client, true).NewSession(context.Background())
		gt.Nil(t, got)
		gt.True(t, errors.Is(err, wantErr))
	})
}

func TestWithPromptCacheDelegatesEmbedding(t *testing.T) {
	var gotDimension int
	var gotInput []string
	client := &mock.LLMClientMock{
		GenerateEmbeddingFunc: func(_ context.Context, dimension int, input []string) ([][]float64, error) {
			gotDimension = dimension
			gotInput = input
			return [][]float64{{0.1, 0.2}}, nil
		},
	}

	result, err := llmclient.WithPromptCache(client, true).GenerateEmbedding(context.Background(), 2, []string{"hello"})
	gt.NoError(t, err)
	gt.Equal(t, result, [][]float64{{0.1, 0.2}})
	gt.Equal(t, gotDimension, 2)
	gt.Equal(t, gotInput, []string{"hello"})
}

func TestWithPromptCacheNilClient(t *testing.T) {
	// Wrapping nil must stay nil rather than produce a decorator that panics on
	// first use.
	gt.Nil(t, llmclient.WithPromptCache(nil, true))
}

// TestWithPromptCacheReachesAgentSessions pins the property the whole design
// rests on: a gollem agent builds its session through the client it was given,
// so wrapping the client also covers agent-created sessions without touching any
// agent call site.
func TestWithPromptCacheReachesAgentSessions(t *testing.T) {
	session := &mock.SessionMock{
		GenerateFunc: func(_ context.Context, _ []gollem.Input, _ ...gollem.GenerateOption) (*gollem.Response, error) {
			return &gollem.Response{Texts: []string{"done"}}, nil
		},
	}
	client, got := recordingClient(session, nil)

	agent := gollem.New(llmclient.WithPromptCache(client, true))
	_, err := agent.Execute(context.Background(), gollem.Text("hello"))
	gt.NoError(t, err)

	cfg := gollem.NewSessionConfig(*got...)
	gt.True(t, cfg.PromptCache())
}
