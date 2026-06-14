package github_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	githubtool "github.com/secmon-lab/warren/pkg/tool/github"
	"github.com/secmon-lab/warren/pkg/utils/errutil"
	"github.com/urfave/cli/v3"
)

// bindFlags parses the action's flags so that env-var sources populate the
// flag-bound struct fields.
func bindFlags(t *testing.T, action *githubtool.Action) {
	t.Helper()
	cmd := &cli.Command{
		Name:   "github",
		Flags:  action.Flags(),
		Action: func(context.Context, *cli.Command) error { return nil },
	}
	gt.NoError(t, cmd.Run(context.Background(), []string{"github"}))
}

// testPrivateKey returns a valid PEM-encoded RSA private key. The external
// github toolset parses the key during construction, so a structurally valid
// key is required even for offline tests.
func testPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	gt.NoError(t, err)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	return string(pem.EncodeToMemory(block))
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	gt.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// configure builds and configures an Action via its CLI flags.
func configure(t *testing.T, appID, installationID int64, privateKey string, configFiles []string) (*githubtool.Action, error) {
	t.Helper()
	t.Setenv("WARREN_GITHUB_APP_ID", strconv.FormatInt(appID, 10))
	t.Setenv("WARREN_GITHUB_APP_INSTALLATION_ID", strconv.FormatInt(installationID, 10))
	t.Setenv("WARREN_GITHUB_APP_PRIVATE_KEY", privateKey)
	t.Setenv("WARREN_GITHUB_APP_CONFIG", strings.Join(configFiles, ","))

	action := &githubtool.Action{}
	bindFlags(t, action)
	return action, action.Configure(context.Background())
}

func TestGitHubConfigureValidation(t *testing.T) {
	pk := testPrivateKey(t)

	testCases := []struct {
		name              string
		appID             int64
		installationID    int64
		privateKey        string
		expectUnavailable bool
		expectErr         bool
	}{
		{name: "missing app ID", appID: 0, installationID: 12345, privateKey: pk, expectUnavailable: true, expectErr: true},
		{name: "missing installation ID", appID: 12345, installationID: 0, privateKey: pk, expectUnavailable: true, expectErr: true},
		{name: "missing private key", appID: 12345, installationID: 67890, privateKey: "", expectUnavailable: true, expectErr: true},
		{name: "invalid private key", appID: 12345, installationID: 67890, privateKey: "not-a-pem", expectUnavailable: false, expectErr: true},
		{name: "valid credentials", appID: 12345, installationID: 67890, privateKey: pk, expectUnavailable: false, expectErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := configure(t, tc.appID, tc.installationID, tc.privateKey, nil)
			if !tc.expectErr {
				gt.NoError(t, err)
				return
			}
			gt.Error(t, err)
			gt.Value(t, errors.Is(err, errutil.ErrActionUnavailable)).Equal(tc.expectUnavailable)
		})
	}
}

func TestGitHubLoadConfigAndPrompt(t *testing.T) {
	configContent := `repositories:
  - owner: "test-owner"
    repository: "test-repo"
    description: "Test repository"
    default_branch: "main"
  - owner: "another-owner"
    repository: "another-repo"
    description: "Another test repository"`

	path := writeConfig(t, configContent)
	action, err := configure(t, 12345, 67890, testPrivateKey(t), []string{path})
	gt.NoError(t, err)

	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.S(t, prompt).Contains("test-owner/test-repo")
	gt.S(t, prompt).Contains("Test repository")
	gt.S(t, prompt).Contains("default branch: main")
	gt.S(t, prompt).Contains("another-owner/another-repo")
}

func TestGitHubInvalidConfig(t *testing.T) {
	// owner missing -> validation error during Configure
	path := writeConfig(t, "repositories:\n  - repository: \"only-repo\"\n")
	_, err := configure(t, 12345, 67890, testPrivateKey(t), []string{path})
	gt.Error(t, err)
	gt.Value(t, errors.Is(err, errutil.ErrActionUnavailable)).Equal(false)
}

func TestGitHubPromptEmptyWithoutConfig(t *testing.T) {
	action, err := configure(t, 12345, 67890, testPrivateKey(t), nil)
	gt.NoError(t, err)
	prompt, err := action.Prompt(context.Background())
	gt.NoError(t, err)
	gt.S(t, prompt).Equal("")
}

func TestGitHubSpecsDelegation(t *testing.T) {
	action, err := configure(t, 12345, 67890, testPrivateKey(t), nil)
	gt.NoError(t, err)

	specs, err := action.Specs(context.Background())
	gt.NoError(t, err)
	gt.A(t, specs).Length(5)

	names := map[string]bool{}
	for _, s := range specs {
		names[s.Name] = true
	}
	for _, want := range []string{
		"github_code_search",
		"github_issue_search",
		"github_get_content",
		"github_list_commits",
		"github_get_blame",
	} {
		gt.Value(t, names[want]).Equal(true)
	}
}

func TestGitHubSpecsBeforeConfigure(t *testing.T) {
	action := &githubtool.Action{}
	_, err := action.Specs(context.Background())
	gt.Error(t, err)
}
