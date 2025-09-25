package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestGenAIContentFormat_Validate(t *testing.T) {
	t.Run("valid text content format", func(t *testing.T) {
		contentFormat := types.GenAIContentFormatText
		err := contentFormat.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid json content format", func(t *testing.T) {
		contentFormat := types.GenAIContentFormatJSON
		err := contentFormat.Validate()
		gt.NoError(t, err)
	})

	t.Run("invalid content format", func(t *testing.T) {
		contentFormat := types.GenAIContentFormat("invalid")
		err := contentFormat.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content format")
		gt.S(t, err.Error()).Contains("invalid")
	})

	t.Run("empty content format", func(t *testing.T) {
		contentFormat := types.GenAIContentFormat("")
		err := contentFormat.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content format")
	})

	t.Run("case sensitive validation", func(t *testing.T) {
		contentFormat := types.GenAIContentFormat("TEXT")
		err := contentFormat.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content format")
	})
}
