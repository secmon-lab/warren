package service_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
)

func TestConsole(t *testing.T) {
	buf := &bytes.Buffer{}
	svc := service.NewConsole(buf)

	// Test PostAlert
	thread, err := svc.PostAlert(context.Background(), model.Alert{
		Title:  "Test Alert",
		Schema: "test.alert.v1",
		Attributes: []model.Attribute{
			{
				Key:   "severity",
				Value: "high",
			},
			{
				Key:   "source",
				Value: "test",
				Link:  "https://example.com",
			},
		},
		Data: map[string]interface{}{
			"foo": "bar",
			"baz": 123,
		},
	})
	gt.NoError(t, err)
	gt.NotNil(t, thread)

	// Test UpdateAlert
	gt.NoError(t, thread.UpdateAlert(context.Background(), model.Alert{
		Title: "Updated Alert",
		Finding: &model.AlertFinding{
			Severity:       model.AlertSeverityHigh,
			Summary:        "Test Summary",
			Reason:         "Test Reason",
			Recommendation: "Test Recommendation",
		},
	}))

	// Test PostNextAction
	gt.NoError(t, thread.PostNextAction(context.Background(), prompt.ActionPromptResult{
		Action: "test_action",
		Args: map[string]any{
			"key": "value",
		},
	}))

	// Test AttachFile
	gt.NoError(t, thread.AttachFile(context.Background(),
		"Test file",
		"test.txt",
		[]byte("Hello, World!"),
	))

	// Test PostFinding
	gt.NoError(t, thread.PostFinding(context.Background(), model.AlertFinding{
		Severity:       model.AlertSeverityHigh,
		Summary:        "Test Finding",
		Reason:         "Test Reason",
		Recommendation: "Test Recommendation",
	}))

	gt.String(t, buf.String()).Contains(`
=== 🚨 New Alert ===
Title: Test Alert
Schema: test.alert.v1
`)

	gt.String(t, buf.String()).Contains(`
=== 📝 Alert Update ===
Title: Updated Alert
`)

	gt.String(t, buf.String()).Contains(`
=== ⚡ Next Action ===
Action: test_action
`)

	gt.String(t, buf.String()).Contains(`
=== 📎 File Attachment ===
Comment: Test file
Filename: test.txt
`)

}
