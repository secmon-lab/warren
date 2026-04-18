package testutil

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
)

// updateGolden controls whether running tests rewrite golden files with the
// current observed output rather than asserting against the existing file.
// Enable by running `go test ./... -update-slack-golden`.
var updateGolden = flag.Bool("update-slack-golden", false,
	"update golden files under testdata/slack_golden with current recorder output")

// AssertGolden compares got against the file at testdata/slack_golden/<name>.
//
// If the -update-slack-golden flag is set, or the file does not yet exist, the
// file is written with got's contents instead of failing the test.
//
// name must include the `.json` extension and may use slash-separated
// subdirectories (e.g. "chat_from_slack/mention_with_ticket.json").
func AssertGolden(t *testing.T, name string, got []byte) {
	t.Helper()

	path := filepath.Join("testdata", "slack_golden", name)

	if *updateGolden {
		writeGolden(t, path, got)
		return
	}

	want, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: seed the file so subsequent runs have something to
		// compare against. Fail once so the developer notices.
		writeGolden(t, path, got)
		t.Fatalf("golden file %s did not exist; wrote initial contents (review and rerun)", path)
	}
	gt.NoError(t, err).Required()

	if !bytes.Equal(bytes.TrimRight(want, "\n"), bytes.TrimRight(got, "\n")) {
		t.Errorf("slack golden mismatch: %s\n--- want\n%s\n--- got\n%s\n(rerun with -update-slack-golden to accept)",
			path, string(want), string(got))
	}
}

func writeGolden(t *testing.T, path string, data []byte) {
	t.Helper()
	dir := filepath.Dir(path)
	gt.NoError(t, os.MkdirAll(dir, 0o755)).Required()
	gt.NoError(t, os.WriteFile(path, data, 0o644)).Required()
}
