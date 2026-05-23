// Package testutil provides helpers for recording Slack API calls during tests.
//
// The central type is Recorder, which captures every Slack API invocation made
// through a SlackClientMock built via NewSlackClientMock. Calls are stored in
// chronological order across all methods so that tests (in particular the
// Phase 0 regression protection golden tests under chat-session-redesign spec)
// can assert against a stable JSON serialization of the entire interaction.
//
// Usage:
//
//	rec := testutil.NewRecorder()
//	client := testutil.NewSlackClientMock(rec)
//	// ... use client in production code paths ...
//	got := rec.CallsJSON()
//	golden.Assert(t, "fixture_name.json", got)
//
// Determinism: timestamps returned from PostMessageContext are synthesized by
// the recorder as "1700000000.{seq:06d}" so no wall-clock randomness leaks into
// the captured stream. MsgOption values are decoded by replaying the option
// chain through a real slack.Client whose HTTP transport is intercepted, so
// what we record matches the URL-encoded payload Slack itself would receive
// on the wire — including blocks and attachments, which since slack-go v0.23
// are only serialized at HTTP send time.
package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/secmon-lab/warren/pkg/domain/mock"
	slackSDK "github.com/slack-go/slack"
)

// Call captures a single Slack API invocation in a form suitable for
// deterministic JSON comparison.
type Call struct {
	Seq    int             `json:"seq"`
	Method string          `json:"method"`
	Args   json.RawMessage `json:"args"`
}

// Recorder stores Slack API calls in the order they were observed.
//
// A Recorder is safe for concurrent use.
type Recorder struct {
	mu        sync.Mutex
	calls     []Call
	tsCounter int
}

// NewRecorder constructs an empty Recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// record appends a call with deterministic sequence number.
func (r *Recorder) record(method string, args any) {
	data, err := json.Marshal(args)
	if err != nil {
		// Fallback to a string representation so the test still shows something
		// useful rather than panicking during capture.
		data = fmt.Appendf(nil, `{"_marshal_error": %q}`, err.Error())
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, Call{
		Seq:    len(r.calls),
		Method: method,
		Args:   data,
	})
}

// fakeTS returns a deterministic Slack timestamp string for post/update calls.
// The format mirrors Slack's "seconds.microseconds" convention but uses the
// recorder's internal counter to avoid wall-clock dependence.
func (r *Recorder) fakeTS() string {
	r.mu.Lock()
	r.tsCounter++
	n := r.tsCounter
	r.mu.Unlock()
	return fmt.Sprintf("1700000000.%06d", n)
}

// Calls returns a copy of the recorded calls in chronological order.
func (r *Recorder) Calls() []Call {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Call, len(r.calls))
	copy(out, r.calls)
	return out
}

// CallsJSON marshals the recorded calls to a pretty-printed JSON byte slice
// suitable for direct golden file comparison.
func (r *Recorder) CallsJSON() []byte {
	data, err := json.MarshalIndent(r.Calls(), "", "  ")
	if err != nil {
		return fmt.Appendf(nil, `{"_marshal_error": %q}`, err.Error())
	}
	return append(data, '\n')
}

// Reset clears all captured calls. The timestamp counter is also reset so that
// post/update calls after Reset start again at 1700000000.000001.
func (r *Recorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = nil
	r.tsCounter = 0
}

// NewSlackClientMock constructs a SlackClientMock whose every method records
// invocations into the provided Recorder before returning safe default values.
//
// Tests may override any Func field on the returned mock to customize a
// specific method's behavior (for example, to simulate a failure or to return
// a particular conversation history payload). When a Func is overridden the
// override is fully responsible for recording the call if that is desired.
func NewSlackClientMock(r *Recorder) *mock.SlackClientMock {
	m := &mock.SlackClientMock{}

	m.AuthTestFunc = func() (*slackSDK.AuthTestResponse, error) {
		r.record("AuthTest", struct{}{})
		return &slackSDK.AuthTestResponse{
			UserID: "U_WARREN_BOT",
			User:   "warren",
			TeamID: "T_TEAM",
			Team:   "warren-test",
			URL:    "https://warren-test.slack.com/",
		}, nil
	}

	m.PostMessageContextFunc = func(ctx context.Context, channelID string, options ...slackSDK.MsgOption) (string, string, error) {
		r.record("PostMessageContext", buildMsgArgs(channelID, "", options))
		return channelID, r.fakeTS(), nil
	}

	m.UpdateMessageContextFunc = func(ctx context.Context, channelID string, timestamp string, options ...slackSDK.MsgOption) (string, string, string, error) {
		r.record("UpdateMessageContext", buildMsgArgs(channelID, timestamp, options))
		return channelID, timestamp, "h_" + timestamp, nil
	}

	m.DeleteMessageContextFunc = func(ctx context.Context, channelID string, timestamp string) (string, string, error) {
		r.record("DeleteMessageContext", map[string]any{
			"channel":   channelID,
			"timestamp": timestamp,
		})
		return channelID, timestamp, nil
	}

	m.GetBotInfoContextFunc = func(ctx context.Context, parameters slackSDK.GetBotInfoParameters) (*slackSDK.Bot, error) {
		r.record("GetBotInfoContext", map[string]any{
			"bot":  parameters.Bot,
			"team": parameters.TeamID,
		})
		return &slackSDK.Bot{
			ID:     "B_WARREN_BOT",
			AppID:  "A_WARREN_APP",
			Name:   "warren",
			UserID: "U_WARREN_BOT",
		}, nil
	}

	m.GetConversationHistoryContextFunc = func(ctx context.Context, params *slackSDK.GetConversationHistoryParameters) (*slackSDK.GetConversationHistoryResponse, error) {
		r.record("GetConversationHistoryContext", map[string]any{
			"channel":   params.ChannelID,
			"oldest":    params.Oldest,
			"latest":    params.Latest,
			"limit":     params.Limit,
			"inclusive": params.Inclusive,
			"cursor":    params.Cursor,
		})
		return &slackSDK.GetConversationHistoryResponse{
			SlackResponse: slackSDK.SlackResponse{Ok: true},
			Messages:      nil,
		}, nil
	}

	m.GetConversationInfoFunc = func(input *slackSDK.GetConversationInfoInput) (*slackSDK.Channel, error) {
		r.record("GetConversationInfo", map[string]any{
			"channel":             input.ChannelID,
			"include_locale":      input.IncludeLocale,
			"include_num_members": input.IncludeNumMembers,
		})
		ch := &slackSDK.Channel{}
		ch.ID = input.ChannelID
		ch.Name = "test-channel"
		return ch, nil
	}

	m.GetConversationRepliesContextFunc = func(ctx context.Context, params *slackSDK.GetConversationRepliesParameters) ([]slackSDK.Message, bool, string, error) {
		r.record("GetConversationRepliesContext", map[string]any{
			"channel":   params.ChannelID,
			"timestamp": params.Timestamp,
			"oldest":    params.Oldest,
			"latest":    params.Latest,
			"limit":     params.Limit,
			"inclusive": params.Inclusive,
			"cursor":    params.Cursor,
		})
		return nil, false, "", nil
	}

	m.GetTeamInfoFunc = func() (*slackSDK.TeamInfo, error) {
		r.record("GetTeamInfo", struct{}{})
		return &slackSDK.TeamInfo{ID: "T_TEAM", Name: "warren-test"}, nil
	}

	m.GetUserGroupsFunc = func(options ...slackSDK.GetUserGroupsOption) ([]slackSDK.UserGroup, error) {
		r.record("GetUserGroups", map[string]any{"num_options": len(options)})
		return nil, nil
	}

	m.GetUserInfoFunc = func(userID string) (*slackSDK.User, error) {
		r.record("GetUserInfo", map[string]any{"user_id": userID})
		u := &slackSDK.User{ID: userID, Name: "test-user-" + userID}
		return u, nil
	}

	m.GetUsersInfoFunc = func(users ...string) (*[]slackSDK.User, error) {
		sorted := append([]string(nil), users...)
		sort.Strings(sorted)
		r.record("GetUsersInfo", map[string]any{"user_ids": sorted})
		out := make([]slackSDK.User, 0, len(users))
		for _, id := range users {
			out = append(out, slackSDK.User{ID: id, Name: "test-user-" + id})
		}
		return &out, nil
	}

	m.OpenViewFunc = func(triggerID string, view slackSDK.ModalViewRequest) (*slackSDK.ViewResponse, error) {
		r.record("OpenView", map[string]any{
			"trigger_id":  triggerID,
			"callback_id": view.CallbackID,
			"title":       modalText(view.Title),
		})
		return &slackSDK.ViewResponse{}, nil
	}

	m.UpdateViewFunc = func(view slackSDK.ModalViewRequest, externalID string, hash string, viewID string) (*slackSDK.ViewResponse, error) {
		r.record("UpdateView", map[string]any{
			"external_id": externalID,
			"hash":        hash,
			"view_id":     viewID,
			"callback_id": view.CallbackID,
			"title":       modalText(view.Title),
		})
		return &slackSDK.ViewResponse{}, nil
	}

	m.SearchMessagesContextFunc = func(ctx context.Context, query string, params slackSDK.SearchParameters) (*slackSDK.SearchMessages, error) {
		r.record("SearchMessagesContext", map[string]any{
			"query": query,
			"count": params.Count,
			"page":  params.Page,
			"sort":  params.Sort,
		})
		return &slackSDK.SearchMessages{}, nil
	}

	m.UploadFileContextFunc = func(ctx context.Context, params slackSDK.UploadFileParameters) (*slackSDK.FileSummary, error) {
		r.record("UploadFileContext", map[string]any{
			"channel":         params.Channel,
			"filename":        params.Filename,
			"title":           params.Title,
			"initial_comment": params.InitialComment,
			"thread_ts":       params.ThreadTimestamp,
			"has_content":     len(params.Content) > 0,
			"has_reader":      params.Reader != nil,
		})
		return &slackSDK.FileSummary{ID: "F_FILE"}, nil
	}

	return m
}

// buildMsgArgs decodes the MsgOption chain into a deterministic map suitable
// for JSON serialization. Starting with slack-go v0.23, blocks and attachments
// are only serialized into the request body at HTTP send time, so we capture
// the request via a synthetic HTTP transport to obtain the same wire payload
// Slack itself would receive.
func buildMsgArgs(channelID string, timestamp string, options []slackSDK.MsgOption) map[string]any {
	values := captureMsgValues(channelID, options)

	out := map[string]any{
		"channel": channelID,
	}
	if timestamp != "" {
		out["timestamp"] = timestamp
	}

	// Record the handful of fields that Warren actually uses. Additional
	// fields may be added here as coverage expands.
	copyField(values, out, "text")
	copyField(values, out, "thread_ts")
	copyField(values, out, "reply_broadcast")
	copyField(values, out, "unfurl_links")
	copyField(values, out, "unfurl_media")
	copyField(values, out, "icon_emoji")
	copyField(values, out, "icon_url")
	copyField(values, out, "username")
	copyField(values, out, "as_user")

	if raw := values.Get("blocks"); raw != "" {
		// blocks is JSON-encoded string; decode so assertions can compare
		// structure rather than exact serialization whitespace.
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			out["blocks"] = decoded
		} else {
			out["blocks_raw"] = raw
		}
	}
	if raw := values.Get("attachments"); raw != "" {
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err == nil {
			out["attachments"] = decoded
		} else {
			out["attachments_raw"] = raw
		}
	}

	return out
}

// captureMsgValues replays the given MsgOption chain through a real
// slack.Client whose HTTP transport is intercepted, returning the parsed
// form-encoded body that the SDK would have sent to Slack. This is the only
// reliable way to inspect blocks/attachments since slack-go v0.23 because
// those fields are no longer materialized in url.Values until HTTP send time.
func captureMsgValues(channelID string, options []slackSDK.MsgOption) url.Values {
	ct := &captureTransport{}
	api := slackSDK.New("xoxb-test", slackSDK.OptionHTTPClient(&http.Client{Transport: ct}))
	_, _, _ = api.PostMessageContext(context.Background(), channelID, options...)

	if !strings.Contains(ct.contentType, "application/x-www-form-urlencoded") {
		return url.Values{}
	}
	parsed, err := url.ParseQuery(string(ct.body))
	if err != nil {
		return url.Values{}
	}
	return parsed
}

// captureTransport is a minimal http.RoundTripper that records the outgoing
// request body and returns a canned successful chat.postMessage response so
// the slack.Client call completes without error.
type captureTransport struct {
	body        []byte
	contentType string
}

func (t *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err == nil {
			t.body = b
		}
	}
	t.contentType = req.Header.Get("Content-Type")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok":true,"channel":"X","ts":"1700000000.000000"}`))),
		Request:    req,
	}
	return resp, nil
}

func copyField(values url.Values, dst map[string]any, key string) {
	if v := values.Get(key); v != "" {
		dst[key] = v
	}
}

// modalText extracts a plain-text title from a Slack modal TextBlockObject.
func modalText(t *slackSDK.TextBlockObject) string {
	if t == nil {
		return ""
	}
	return t.Text
}
