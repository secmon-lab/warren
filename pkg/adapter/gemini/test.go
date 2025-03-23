package gemini

import (
	"os"
	"testing"

	"github.com/m-mizutani/gt"
)

// NewTestClient creates a new Gemini client for testing.
// It looks up environment variables of TEST_GEMINI_PROJECT. If found, it returns project. If not, the test will skip.
func NewTestClient(t *testing.T, opts ...Option) *GeminiClient {
	project, ok := os.LookupEnv("TEST_GEMINI_PROJECT")
	if !ok {
		t.Skip("TEST_GEMINI_PROJECT is not set")
	}

	client, err := New(t.Context(), project, opts...)
	gt.NoError(t, err).Must()
	return client
}
