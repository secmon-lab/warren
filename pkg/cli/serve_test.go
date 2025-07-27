package cli_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli"
)

func TestServeCommand_StrictAlertValidation(t *testing.T) {
	// Test that --strict-alert without policy files returns an error
	ctx := context.Background()

	// Test with --strict-alert but no policy files
	// We need to provide minimal required flags to get to our validation
	err := cli.Run(ctx, []string{
		"warren", "serve",
		"--strict-alert",
		"--gemini-project-id", "test-project",
		"--firestore-project-id", "test-project",
	})
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("--strict-alert requires at least one policy file")

	// Test without --strict-alert and no policy files should not fail on this validation
	// (it may fail on other validations, but not on the strict-alert check)
}
