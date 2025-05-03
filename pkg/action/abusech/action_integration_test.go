package abusech_test

import (
	"context"
	"os"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/action/abusech"
)

func TestActionIntegration(t *testing.T) {
	apiKey := os.Getenv("TEST_ABUSECH_API_KEY")
	if apiKey == "" {
		t.Skip("TEST_ABUSECH_API_KEY is not set")
	}

	action := abusech.New()
	action.SetAPIKey(apiKey)

	t.Run("success case - query known malware hash", func(t *testing.T) {
		// Using a known malware hash value
		// This is an example of an actual malware hash
		hash := "094fd325049b8a9cf6d3e5ef2a6d4cc6a567d7d49c35f8bb8dd9e3c6acf3d78d"

		result, err := action.Run(context.Background(), "abusech.bazaar.query", map[string]any{
			"hash": hash,
		})
		gt.NoError(t, err)

		// Verify response structure
		gt.Value(t, result["query_status"]).Equal("ok")
		data, ok := result["data"].([]any)
		gt.True(t, ok)
		gt.True(t, len(data) > 0)

		// Verify structure of the first entry
		entry, ok := data[0].(map[string]any)
		gt.True(t, ok)
		gt.Value(t, entry["sha256_hash"]).Equal(hash)
		gt.Value(t, entry["file_type"]).NotEqual("")
		gt.Value(t, entry["signature"]).NotEqual("")
	})

	t.Run("error case - query unknown hash", func(t *testing.T) {
		// Using a non-existent hash value
		hash := "0000000000000000000000000000000000000000000000000000000000000000"

		result, err := action.Run(context.Background(), "abusech.bazaar.query", map[string]any{
			"hash": hash,
		})
		gt.NoError(t, err)

		// Verify response structure
		gt.Value(t, result["query_status"]).Equal("ok")
		data, ok := result["data"].([]any)
		gt.True(t, ok)
		gt.True(t, len(data) == 0)
	})

	t.Run("error case - invalid hash format", func(t *testing.T) {
		// Using an invalid hash format
		hash := "invalid-hash"

		result, err := action.Run(context.Background(), "abusech.bazaar.query", map[string]any{
			"hash": hash,
		})
		gt.NoError(t, err)

		// Verify response structure
		gt.Value(t, result["query_status"]).Equal("ok")
		data, ok := result["data"].([]any)
		gt.True(t, ok)
		gt.True(t, len(data) == 0)
	})
}
