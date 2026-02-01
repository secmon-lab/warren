package trace_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/gt"
	gollemtrace "github.com/m-mizutani/gollem/trace"
	traceAdapter "github.com/secmon-lab/warren/pkg/adapter/trace"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/safe"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func TestRepository_SaveAndRead(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_TRACE_BUCKET", "TEST_TRACE_PREFIX")

	ctx := context.Background()
	bucket := vars.Get("TEST_TRACE_BUCKET")
	prefix := fmt.Sprintf("%stest-%d/", vars.Get("TEST_TRACE_PREFIX"), time.Now().UnixNano())

	repo := gt.R1(traceAdapter.New(ctx, bucket)).NoError(t)
	repo.WithPrefix(prefix)
	defer safe.Close(ctx, repo)

	traceID := fmt.Sprintf("test-trace-%d", time.Now().UnixNano())
	now := time.Now().UTC().Truncate(time.Millisecond)

	trace := &gollemtrace.Trace{
		TraceID: traceID,
		RootSpan: &gollemtrace.Span{
			SpanID:    "span-1",
			Kind:      gollemtrace.SpanKindAgentExecute,
			Name:      "test-agent",
			StartedAt: now,
			EndedAt:   now.Add(5 * time.Second),
			Duration:  5 * time.Second,
			Status:    gollemtrace.SpanStatusOK,
			Children: []*gollemtrace.Span{
				{
					SpanID:    "span-2",
					ParentID:  "span-1",
					Kind:      gollemtrace.SpanKindLLMCall,
					Name:      "llm-call",
					StartedAt: now.Add(1 * time.Second),
					EndedAt:   now.Add(3 * time.Second),
					Duration:  2 * time.Second,
					Status:    gollemtrace.SpanStatusOK,
					LLMCall: &gollemtrace.LLMCallData{
						InputTokens:  100,
						OutputTokens: 50,
						Model:        "gemini-2.0-flash",
					},
				},
			},
		},
		Metadata: gollemtrace.TraceMetadata{
			Model:    "gemini-2.0-flash",
			Strategy: "plan-execute",
			Labels:   map[string]string{"env": "test"},
		},
		StartedAt: now,
		EndedAt:   now.Add(5 * time.Second),
	}

	// Save trace data
	gt.NoError(t, repo.Save(ctx, trace)).Required()

	// Read back and verify
	gcsClient := gt.R1(storage.NewClient(ctx)).NoError(t)
	defer safe.Close(ctx, gcsClient)

	objectPath := fmt.Sprintf("%s%s.json", prefix, traceID)
	reader := gt.R1(gcsClient.Bucket(bucket).Object(objectPath).NewReader(ctx)).NoError(t)
	defer safe.Close(ctx, reader)

	data := gt.R1(io.ReadAll(reader)).NoError(t)

	var saved gollemtrace.Trace
	gt.NoError(t, json.Unmarshal(data, &saved)).Required()

	gt.V(t, saved.TraceID).Equal(traceID)
	gt.V(t, saved.Metadata.Model).Equal("gemini-2.0-flash")
	gt.V(t, saved.Metadata.Strategy).Equal("plan-execute")
	gt.M(t, saved.Metadata.Labels).HasKey("env")
	gt.V(t, saved.Metadata.Labels["env"]).Equal("test")

	gt.V(t, saved.RootSpan).NotNil().Required()
	gt.V(t, saved.RootSpan.SpanID).Equal("span-1")
	gt.V(t, saved.RootSpan.Kind).Equal(gollemtrace.SpanKindAgentExecute)
	gt.V(t, saved.RootSpan.Name).Equal("test-agent")
	gt.V(t, saved.RootSpan.Status).Equal(gollemtrace.SpanStatusOK)

	gt.A(t, saved.RootSpan.Children).Length(1).Required()
	child := saved.RootSpan.Children[0]
	gt.V(t, child.SpanID).Equal("span-2")
	gt.V(t, child.Kind).Equal(gollemtrace.SpanKindLLMCall)
	gt.V(t, child.LLMCall).NotNil().Required()
	gt.N(t, child.LLMCall.InputTokens).Equal(100)
	gt.N(t, child.LLMCall.OutputTokens).Equal(50)
	gt.V(t, child.LLMCall.Model).Equal("gemini-2.0-flash")
}

func TestNewSafe_SuppressesErrors(t *testing.T) {
	logger := logging.Default()

	// Create a repository that always fails
	failRepo := &failingRepository{}
	safeRepo := traceAdapter.NewSafe(failRepo, logger)

	ctx := context.Background()
	trace := &gollemtrace.Trace{
		TraceID: "test-safe-trace",
	}

	// Save should NOT return an error even though inner repository fails
	err := safeRepo.Save(ctx, trace)
	gt.NoError(t, err)
}

type failingRepository struct{}

func (r *failingRepository) Save(_ context.Context, _ *gollemtrace.Trace) error {
	return fmt.Errorf("simulated GCS failure")
}
