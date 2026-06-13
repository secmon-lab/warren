package webfetch_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gollem-dev/gollem"
	"github.com/gollem-dev/gollem/mock"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/webfetch"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
)

func TestParseLLMArgs_Empty(t *testing.T) {
	m, err := webfetch.ParseLLMArgs("")
	gt.NoError(t, err).Required()
	gt.Map(t, m).Length(0)
}

func TestParseLLMArgs_SinglePair(t *testing.T) {
	m, err := webfetch.ParseLLMArgs("project_id=my-proj")
	gt.NoError(t, err).Required()
	gt.Map(t, m).HasKeyValue("project_id", "my-proj")
}

func TestParseLLMArgs_MultiplePairs(t *testing.T) {
	m, err := webfetch.ParseLLMArgs("project_id=p,location=us-central1,temperature=0.2")
	gt.NoError(t, err).Required()
	gt.Map(t, m).Length(3)
	gt.Value(t, m["project_id"]).Equal("p")
	gt.Value(t, m["location"]).Equal("us-central1")
	gt.Value(t, m["temperature"]).Equal("0.2")
}

func TestParseLLMArgs_WhitespaceTolerated(t *testing.T) {
	m, err := webfetch.ParseLLMArgs(" project_id = p , location = us-central1 ")
	gt.NoError(t, err).Required()
	gt.Value(t, m["project_id"]).Equal("p")
	gt.Value(t, m["location"]).Equal("us-central1")
}

func TestParseLLMArgs_TokenWithoutEquals(t *testing.T) {
	_, err := webfetch.ParseLLMArgs("project_id")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestParseLLMArgs_EmptyKey(t *testing.T) {
	_, err := webfetch.ParseLLMArgs("=value")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestParseLLMArgs_DuplicateKey(t *testing.T) {
	_, err := webfetch.ParseLLMArgs("project_id=a,project_id=b")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestParseLLMArgs_EmptyToken(t *testing.T) {
	_, err := webfetch.ParseLLMArgs("project_id=a,,location=b")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_ModelMissing(t *testing.T) {
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "", map[string]string{}, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_UnknownProvider(t *testing.T) {
	_, err := webfetch.BuildLLMClient(t.Context(), "mistral", "model", map[string]string{}, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_GeminiMissingArgs(t *testing.T) {
	_, err := webfetch.BuildLLMClient(t.Context(), "gemini", "gemini-2.5-flash", map[string]string{}, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_GeminiUnknownKey(t *testing.T) {
	args := map[string]string{
		"project_id": "p",
		"location":   "us-central1",
		"foo":        "bar",
	}
	_, err := webfetch.BuildLLMClient(t.Context(), "gemini", "gemini-2.5-flash", args, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_GeminiRejectsAPIKey(t *testing.T) {
	args := map[string]string{"project_id": "p", "location": "us-central1"}
	_, err := webfetch.BuildLLMClient(t.Context(), "gemini", "gemini-2.5-flash", args, "should-not-be-set")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAIMissingAPIKey(t *testing.T) {
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", map[string]string{}, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAIUnknownKey(t *testing.T) {
	args := map[string]string{"project_id": "p"}
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAIInvalidTemperature(t *testing.T) {
	args := map[string]string{"temperature": "not-a-float"}
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAITemperatureTooHigh(t *testing.T) {
	args := map[string]string{"temperature": "2.5"}
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAITemperatureNegative(t *testing.T) {
	args := map[string]string{"temperature": "-0.1"}
	_, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_OpenAITemperatureBoundaryAccepted(t *testing.T) {
	// 0.0 and 2.0 are both accepted (inclusive bounds).
	for _, raw := range []string{"0", "0.0", "2", "2.0"} {
		args := map[string]string{"temperature": raw}
		client, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "fake-key")
		gt.NoError(t, err).Required()
		gt.NotNil(t, client)
	}
}

// TestBuildLLMClient_OpenAIHappy verifies the dispatch reaches the OpenAI
// constructor when given valid inputs. The OpenAI gollem constructor does not
// require network access at construction time, so this happy-path test is
// safe to run unconditionally.
func TestBuildLLMClient_OpenAIHappy(t *testing.T) {
	args := map[string]string{"temperature": "0.2"}
	client, err := webfetch.BuildLLMClient(t.Context(), "openai", "gpt-4o", args, "fake-key")
	gt.NoError(t, err).Required()
	gt.NotNil(t, client)
}

func TestBuildLLMClient_ClaudeRoute_VertexHappy_RequiresGCP(t *testing.T) {
	if testing.Short() {
		t.Skip("Claude Vertex constructor requires GCP authentication; skipped in -short mode")
	}
	// We don't run real construction in unit tests since it requires GCP creds.
	// Validation-path coverage below verifies all error branches.
}

func TestBuildLLMClient_ClaudeRoute_Unspecified(t *testing.T) {
	_, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4-5-20250929", map[string]string{}, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_ClaudeRoute_Ambiguous(t *testing.T) {
	args := map[string]string{"project_id": "p", "location": "us-east5"}
	_, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4@20250514", args, "also-api-key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_ClaudeRoute_PartialVertexProjectOnly(t *testing.T) {
	args := map[string]string{"project_id": "p"}
	_, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4@20250514", args, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_ClaudeRoute_PartialVertexLocationOnly(t *testing.T) {
	args := map[string]string{"location": "us-east5"}
	_, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4@20250514", args, "")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

func TestBuildLLMClient_ClaudeRoute_UnknownKey(t *testing.T) {
	args := map[string]string{"foo": "bar"}
	_, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4-5-20250929", args, "key")
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagValidation))
}

// TestBuildLLMClient_ClaudeRoute_DirectHappy verifies the Anthropic-direct
// dispatch path. The claude direct constructor does not perform network calls
// at construction time, so this works without real credentials.
func TestBuildLLMClient_ClaudeRoute_DirectHappy(t *testing.T) {
	client, err := webfetch.BuildLLMClient(t.Context(), "claude", "claude-sonnet-4-5-20250929", map[string]string{}, "fake-key")
	gt.NoError(t, err).Required()
	gt.NotNil(t, client)
}

func TestPingLLMClient_NilClient(t *testing.T) {
	err := webfetch.PingLLMClient(t.Context(), nil)
	gt.Error(t, err).Required()
	gt.True(t, goerr.HasTag(err, errutil.TagInternal))
}

func TestPingLLMClient_Success(t *testing.T) {
	var sessionCalls int
	var generateCalls int
	var capturedOpts []gollem.GenerateOption

	client := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			sessionCalls++
			return &mock.SessionMock{
				GenerateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					generateCalls++
					capturedOpts = opts
					return &gollem.Response{Texts: []string{"ok"}}, nil
				},
			}, nil
		},
	}

	gt.NoError(t, webfetch.PingLLMClient(t.Context(), client))
	gt.Value(t, sessionCalls).Equal(1)
	gt.Value(t, generateCalls).Equal(1)
	// Verify the ping uses a max-tokens cap to keep the call cheap.
	cfg := gollem.NewGenerateConfig(capturedOpts...)
	maxTokens := cfg.MaxTokens()
	gt.NotNil(t, maxTokens)
	gt.Value(t, *maxTokens).Equal(1)
}

func TestPingLLMClient_GenerateError(t *testing.T) {
	upstream := errors.New("upstream failure")
	client := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return &mock.SessionMock{
				GenerateFunc: func(ctx context.Context, input []gollem.Input, opts ...gollem.GenerateOption) (*gollem.Response, error) {
					return nil, upstream
				},
			}, nil
		},
	}

	err := webfetch.PingLLMClient(t.Context(), client)
	gt.Error(t, err).Required()
	gt.True(t, errors.Is(err, upstream))
	gt.True(t, goerr.HasTag(err, errutil.TagLLMError))
}

func TestPingLLMClient_SessionError(t *testing.T) {
	upstream := errors.New("session init failure")
	client := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			return nil, upstream
		},
	}
	err := webfetch.PingLLMClient(t.Context(), client)
	gt.Error(t, err).Required()
	gt.True(t, errors.Is(err, upstream))
	gt.True(t, goerr.HasTag(err, errutil.TagLLMError))
}
