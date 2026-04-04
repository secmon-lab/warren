package webfetch_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/tool/webfetch"
)

func TestAction_Specs(t *testing.T) {
	action := &webfetch.Action{}
	specs, err := action.Specs(t.Context())
	gt.NoError(t, err).Required()
	gt.Array(t, specs).Length(1).Required()
	gt.Value(t, specs[0].Name).Equal("web_fetch")
	gt.Value(t, specs[0].Parameters["url"].Required).Equal(true)
}

func TestAction_Run_InvalidName(t *testing.T) {
	action := &webfetch.Action{}
	_, err := action.Run(t.Context(), "invalid_tool", map[string]any{"url": "https://example.com"})
	gt.Error(t, err)
}

func TestAction_Run_MissingURL(t *testing.T) {
	action := &webfetch.Action{}
	_, err := action.Run(t.Context(), "web_fetch", map[string]any{})
	gt.Error(t, err)
}

func TestAction_Name(t *testing.T) {
	action := &webfetch.Action{}
	gt.Value(t, action.ID()).Equal("webfetch")
}

func TestAction_Configure(t *testing.T) {
	action := &webfetch.Action{}
	gt.NoError(t, action.Configure(t.Context()))
}
