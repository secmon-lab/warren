package testutil_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/service/slack/testutil"
	slackSDK "github.com/slack-go/slack"
)

func TestRecorder_PostMessage_CapturesTextAndThread(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, ts, err := client.PostMessageContext(context.Background(),
		"C123",
		slackSDK.MsgOptionText("hello world", false),
		slackSDK.MsgOptionTS("1234.5678"),
	)
	gt.NoError(t, err)
	gt.V(t, ts).Equal("1700000000.000001")

	calls := rec.Calls()
	gt.A(t, calls).Length(1)
	gt.V(t, calls[0].Method).Equal("PostMessageContext")

	var args map[string]any
	gt.NoError(t, json.Unmarshal(calls[0].Args, &args))
	gt.V(t, args["channel"]).Equal("C123")
	gt.V(t, args["text"]).Equal("hello world")
	gt.V(t, args["thread_ts"]).Equal("1234.5678")
}

func TestRecorder_PostMessage_DecodesBlocks(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	blocks := []slackSDK.Block{
		slackSDK.NewSectionBlock(
			slackSDK.NewTextBlockObject(slackSDK.MarkdownType, "*bold*", false, false),
			nil,
			nil,
		),
	}

	_, _, err := client.PostMessageContext(context.Background(),
		"C456",
		slackSDK.MsgOptionBlocks(blocks...),
	)
	gt.NoError(t, err)

	calls := rec.Calls()
	gt.A(t, calls).Length(1)

	var args map[string]any
	gt.NoError(t, json.Unmarshal(calls[0].Args, &args))

	decodedBlocks, ok := args["blocks"].([]any)
	gt.V(t, ok).Equal(true)
	gt.A(t, decodedBlocks).Length(1)
	block := decodedBlocks[0].(map[string]any)
	gt.V(t, block["type"]).Equal("section")
}

func TestRecorder_UpdateMessage_RecordsTimestamp(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, _, _, err := client.UpdateMessageContext(context.Background(),
		"C789", "1700000000.000001",
		slackSDK.MsgOptionText("updated", false),
	)
	gt.NoError(t, err)

	calls := rec.Calls()
	gt.A(t, calls).Length(1)
	gt.V(t, calls[0].Method).Equal("UpdateMessageContext")

	var args map[string]any
	gt.NoError(t, json.Unmarshal(calls[0].Args, &args))
	gt.V(t, args["channel"]).Equal("C789")
	gt.V(t, args["timestamp"]).Equal("1700000000.000001")
	gt.V(t, args["text"]).Equal("updated")
}

func TestRecorder_ChronologicalOrderingAcrossMethods(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	ctx := context.Background()
	_, _, _ = client.PostMessageContext(ctx, "C1", slackSDK.MsgOptionText("first", false))
	_, _, _, _ = client.UpdateMessageContext(ctx, "C1", "ts1", slackSDK.MsgOptionText("second", false))
	_, _, _ = client.PostMessageContext(ctx, "C1", slackSDK.MsgOptionText("third", false))
	_, _, _ = client.DeleteMessageContext(ctx, "C1", "ts2")

	calls := rec.Calls()
	gt.A(t, calls).Length(4)
	gt.V(t, calls[0].Method).Equal("PostMessageContext")
	gt.V(t, calls[1].Method).Equal("UpdateMessageContext")
	gt.V(t, calls[2].Method).Equal("PostMessageContext")
	gt.V(t, calls[3].Method).Equal("DeleteMessageContext")

	// Sequence numbers monotonically increase from 0
	for i, c := range calls {
		gt.V(t, c.Seq).Equal(i)
	}
}

func TestRecorder_FakeTimestampsAreDeterministic(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, ts1, _ := client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("a", false))
	_, ts2, _ := client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("b", false))
	_, ts3, _ := client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("c", false))

	gt.V(t, ts1).Equal("1700000000.000001")
	gt.V(t, ts2).Equal("1700000000.000002")
	gt.V(t, ts3).Equal("1700000000.000003")
}

func TestRecorder_Reset(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	_, _, _ = client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("a", false))
	gt.A(t, rec.Calls()).Length(1)

	rec.Reset()
	gt.A(t, rec.Calls()).Length(0)

	_, ts, _ := client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("b", false))
	gt.V(t, ts).Equal("1700000000.000001")
}

func TestRecorder_CallsJSON_Stable(t *testing.T) {
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	ctx := context.Background()
	_, _, _ = client.PostMessageContext(ctx, "C1",
		slackSDK.MsgOptionText("hello", false),
		slackSDK.MsgOptionTS("1.2"),
	)

	got := rec.CallsJSON()

	// Parse round-trip to confirm well-formed JSON
	var decoded []map[string]any
	gt.NoError(t, json.Unmarshal(got, &decoded))
	gt.A(t, decoded).Length(1)
	gt.V(t, decoded[0]["method"]).Equal("PostMessageContext")
}

func TestRecorder_OverridingFunc_SkipsBuiltinRecording(t *testing.T) {
	// When the caller overrides a Func field, it takes full responsibility
	// for recording (or not). Verify the override is respected.
	rec := testutil.NewRecorder()
	client := testutil.NewSlackClientMock(rec)

	client.PostMessageContextFunc = func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
		return channelID, "CUSTOM_TS", nil
	}

	_, ts, _ := client.PostMessageContext(context.Background(), "C1", slackSDK.MsgOptionText("x", false))
	gt.V(t, ts).Equal("CUSTOM_TS")

	// Recorder did not see the call because the override replaced the built-in.
	gt.A(t, rec.Calls()).Length(0)
}
