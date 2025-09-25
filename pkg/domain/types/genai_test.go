package types_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

func TestGenAIContentType_Validate(t *testing.T) {
	t.Run("valid text content type", func(t *testing.T) {
		contentType := types.GenAIContentTypeText
		err := contentType.Validate()
		gt.NoError(t, err)
	})

	t.Run("valid json content type", func(t *testing.T) {
		contentType := types.GenAIContentTypeJSON
		err := contentType.Validate()
		gt.NoError(t, err)
	})

	t.Run("invalid content type", func(t *testing.T) {
		contentType := types.GenAIContentType("invalid")
		err := contentType.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content type")
		gt.S(t, err.Error()).Contains("invalid")
	})

	t.Run("empty content type", func(t *testing.T) {
		contentType := types.GenAIContentType("")
		err := contentType.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content type")
	})

	t.Run("case sensitive validation", func(t *testing.T) {
		contentType := types.GenAIContentType("TEXT")
		err := contentType.Validate()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid GenAI content type")
	})
}
