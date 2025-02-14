package prompt_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
)

func TestPrompt(t *testing.T) {
	alert := map[string]any{
		"id":      123,
		"message": "test",
	}
	d, err := prompt.BuildInitPrompt(alert, 3)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).
		Contains("# Alert").
		Contains(`"id": 123`).
		Contains(`"message": "test"`)
}

func TestActionPrompt(t *testing.T) {
	actions := []model.ActionSpec{
		{Name: "action1", Description: "action1 description", Args: []model.ArgumentSpec{{Name: "arg1", Description: "arg1 description", Type: model.ArgumentTypeString, Required: true}}},
	}
	d, err := prompt.BuildActionPrompt(actions)
	gt.NoError(t, err)

	t.Log(d)
	gt.S(t, d).Contains("## `action1`")
	gt.S(t, d).Contains("- `arg1` (string, required): arg1 description")
}
