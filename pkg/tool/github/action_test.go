package github_test

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/gt"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	githubtool "github.com/secmon-lab/warren/pkg/tool/github"
)

func TestGitHubCodeSearch(t *testing.T) {
	// Create mock GitHub client
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetSearchCode,
			github.CodeSearchResult{
				Total: github.Ptr(2),
				CodeResults: []*github.CodeResult{
					{
						Name:    github.Ptr("main.go"),
						Path:    github.Ptr("src/main.go"),
						SHA:     github.Ptr("abc123"),
						HTMLURL: github.Ptr("https://github.com/test/repo/blob/main/src/main.go"),
						Repository: &github.Repository{
							FullName: github.Ptr("test/repo"),
						},
						TextMatches: []*github.TextMatch{
							{
								Fragment: github.Ptr("func main() {\n    fmt.Println(\"test\")\n}"),
							},
						},
					},
					{
						Name:    github.Ptr("utils.go"),
						Path:    github.Ptr("pkg/utils.go"),
						SHA:     github.Ptr("def456"),
						HTMLURL: github.Ptr("https://github.com/test/repo/blob/main/pkg/utils.go"),
						Repository: &github.Repository{
							FullName: github.Ptr("test/repo"),
						},
					},
				},
			},
		),
	)

	client := github.NewClient(mockedHTTPClient)

	// Create action and set mock client
	action := &githubtool.Action{}
	action.SetGitHubClient(client)
	action.SetConfigs([]*githubtool.RepositoryConfig{
		{
			Owner:       "test",
			Repository:  "repo",
			Description: "Test repository",
		},
	})

	// Execute search
	ctx := context.Background()
	args := map[string]any{
		"query":    "fmt.Println",
		"language": "Go",
	}

	result, err := action.Run(ctx, "github_code_search", args)
	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Validate results
	total, ok := result["total"].(int)
	gt.True(t, ok)
	gt.Number(t, total).Equal(2)

	results, ok := result["results"].([]githubtool.CodeSearchResult)
	gt.True(t, ok)
	gt.A(t, results).Length(2)

	// Check first result
	gt.S(t, results[0].Repository).Equal("test/repo")
	gt.S(t, results[0].Path).Equal("src/main.go")
	gt.A(t, results[0].Matches).Length(1)
}

func TestGitHubIssueSearch(t *testing.T) {
	// Create mock GitHub client
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetSearchIssues,
			github.IssuesSearchResult{
				Total: github.Ptr(2),
				Issues: []*github.Issue{
					{
						Number:  github.Ptr(123),
						Title:   github.Ptr("Test Issue"),
						State:   github.Ptr("open"),
						HTMLURL: github.Ptr("https://github.com/test/repo/issues/123"),
						User: &github.User{
							Login: github.Ptr("testuser"),
						},
						Body: github.Ptr("This is a test issue"),
						Labels: []*github.Label{
							{Name: github.Ptr("bug")},
							{Name: github.Ptr("help wanted")},
						},
						RepositoryURL: github.Ptr("https://api.github.com/repos/test/repo"),
					},
					{
						Number:  github.Ptr(456),
						Title:   github.Ptr("Test PR"),
						State:   github.Ptr("open"),
						HTMLURL: github.Ptr("https://github.com/test/repo/pull/456"),
						User: &github.User{
							Login: github.Ptr("anotheruser"),
						},
						PullRequestLinks: &github.PullRequestLinks{
							URL: github.Ptr("https://api.github.com/repos/test/repo/pulls/456"),
						},
						RepositoryURL: github.Ptr("https://api.github.com/repos/test/repo"),
					},
				},
			},
		),
	)

	client := github.NewClient(mockedHTTPClient)

	// Create action and set mock client
	action := &githubtool.Action{}
	action.SetGitHubClient(client)
	action.SetConfigs([]*githubtool.RepositoryConfig{
		{
			Owner:       "test",
			Repository:  "repo",
			Description: "Test repository",
		},
	})

	// Execute search
	ctx := context.Background()
	args := map[string]any{
		"query":  "test",
		"state":  "open",
		"labels": "bug,help wanted",
	}

	result, err := action.Run(ctx, "github_issue_search", args)
	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Validate results
	total, ok := result["total"].(int)
	gt.True(t, ok)
	gt.Number(t, total).Equal(2)

	results, ok := result["results"].([]githubtool.IssueSearchResult)
	gt.True(t, ok)
	gt.A(t, results).Length(2)

	// Check first result (issue)
	gt.Number(t, results[0].Number).Equal(123)
	gt.S(t, results[0].Title).Equal("Test Issue")
	gt.S(t, results[0].State).Equal("open")
	gt.S(t, results[0].User).Equal("testuser")
	gt.B(t, results[0].IsPR).False()
	gt.A(t, results[0].Labels).Length(2)
	gt.B(t, results[0].Labels[0] == "bug" || results[0].Labels[1] == "bug").True()
	gt.B(t, results[0].Labels[0] == "help wanted" || results[0].Labels[1] == "help wanted").True()

	// Check second result (PR)
	gt.Number(t, results[1].Number).Equal(456)
	gt.B(t, results[1].IsPR).True()
}

func TestGitHubGetContent(t *testing.T) {
	// Create mock GitHub client
	content := "package main\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n"
	encodedContent := "cGFja2FnZSBtYWluCgpmdW5jIG1haW4oKSB7CiAgICBmbXQuUHJpbnRsbigiSGVsbG8sIFdvcmxkISIpCn0K"

	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposContentsByOwnerByRepoByPath,
			github.RepositoryContent{
				Name:     github.Ptr("main.go"),
				Path:     github.Ptr("main.go"),
				SHA:      github.Ptr("abc123def456"),
				Size:     github.Ptr(len(content)),
				Content:  github.Ptr(encodedContent),
				HTMLURL:  github.Ptr("https://github.com/test/repo/blob/main/main.go"),
				Type:     github.Ptr("file"),
				Encoding: github.Ptr("base64"),
			},
		),
	)

	client := github.NewClient(mockedHTTPClient)

	// Create action and set mock client
	action := &githubtool.Action{}
	action.SetGitHubClient(client)
	action.SetConfigs([]*githubtool.RepositoryConfig{
		{
			Owner:       "test",
			Repository:  "repo",
			Description: "Test repository",
		},
	})

	// Execute get content
	ctx := context.Background()
	args := map[string]any{
		"owner": "test",
		"repo":  "repo",
		"path":  "main.go",
		"ref":   "main",
	}

	result, err := action.Run(ctx, "github_get_content", args)
	gt.NoError(t, err)
	gt.NotNil(t, result)

	// Validate results
	gt.S(t, result["repository"].(string)).Equal("test/repo")
	gt.S(t, result["path"].(string)).Equal("main.go")
	gt.S(t, result["content"].(string)).Equal(content)
	gt.S(t, result["sha"].(string)).Equal("abc123def456")
	gt.Number(t, result["size"].(int)).Equal(len(content))
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `repositories:
  - owner: "test-owner"
    repository: "test-repo"
    description: "Test repository"
    default_branch: "main"
  - owner: "another-owner"
    repository: "another-repo"
    description: "Another test repository"`

	tmpFile, err := os.CreateTemp("", "github-app-config-*.yaml")
	gt.NoError(t, err)
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	_, err = tmpFile.WriteString(configContent)
	gt.NoError(t, err)
	_ = tmpFile.Close()

	// Create action and configure
	action := &githubtool.Action{}

	// Set required fields for Configure to work
	action.SetTestData(12345, 67890, "dummy-private-key", []string{tmpFile.Name()})

	err = action.Configure(context.Background())
	// Will fail due to invalid private key format
	gt.Error(t, err)
}

func TestConfigureValidation(t *testing.T) {
	testCases := []struct {
		name           string
		appID          int64
		installationID int64
		privateKeyPath string
		configFiles    []string
		expectError    bool
	}{
		{
			name:           "missing app ID",
			appID:          0,
			installationID: 12345,
			privateKeyPath: "key.pem",
			configFiles:    []string{"config.yaml"},
			expectError:    true,
		},
		{
			name:           "missing installation ID",
			appID:          12345,
			installationID: 0,
			privateKeyPath: "key.pem",
			configFiles:    []string{"config.yaml"},
			expectError:    true,
		},
		{
			name:           "missing private key",
			appID:          12345,
			installationID: 67890,
			privateKeyPath: "",
			configFiles:    []string{"config.yaml"},
			expectError:    true,
		},
		{
			name:           "missing config files",
			appID:          12345,
			installationID: 67890,
			privateKeyPath: "key.pem",
			configFiles:    []string{},
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			action := &githubtool.Action{}
			action.SetTestData(tc.appID, tc.installationID, tc.privateKeyPath, tc.configFiles)

			err := action.Configure(context.Background())
			if tc.expectError {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}

func TestGitHubIntegration(t *testing.T) {
	// Check for required environment variables
	appID := os.Getenv("TEST_GITHUB_APP_ID")
	installationID := os.Getenv("TEST_GITHUB_INSTALLATION_ID")
	privateKey := os.Getenv("TEST_GITHUB_PRIVATE_KEY")
	configPath := os.Getenv("TEST_GITHUB_CONFIG")

	if appID == "" || installationID == "" || privateKey == "" {
		t.Skip("TEST_GITHUB_APP_ID, TEST_GITHUB_INSTALLATION_ID, and TEST_GITHUB_PRIVATE_KEY must be set for integration tests")
	}

	// If no config path provided, use test data
	if configPath == "" {
		configPath = "./testdata/config.yaml"
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skipf("Config file does not exist: %s", configPath)
	}

	// Create action
	action := &githubtool.Action{}

	// Convert string IDs to int64
	appIDInt, err := strconv.ParseInt(appID, 10, 64)
	gt.NoError(t, err)

	installIDInt, err := strconv.ParseInt(installationID, 10, 64)
	gt.NoError(t, err)

	action.SetTestData(appIDInt, installIDInt, privateKey, []string{configPath})

	// Configure
	ctx := context.Background()
	err = action.Configure(ctx)
	gt.NoError(t, err)

	// Test code search
	t.Run("code search", func(t *testing.T) {
		searchQuery := os.Getenv("TEST_GITHUB_SEARCH_QUERY")
		if searchQuery == "" {
			searchQuery = "func"
		}

		args := map[string]any{
			"query": searchQuery,
		}

		result, err := action.Run(ctx, "github_code_search", args)
		if err != nil {
			if err.Error() == "repository not in configured list" {
				t.Skip("No matching repositories in config")
			}
			gt.NoError(t, err)
		}

		results := gt.Cast[[]githubtool.CodeSearchResult](t, result["results"])
		t.Logf("Found %d code results for query '%s'", len(results), searchQuery)
	})

	// Test issue search
	t.Run("issue search", func(t *testing.T) {
		args := map[string]any{
			"query": "state:closed",
		}

		result, err := action.Run(ctx, "github_issue_search", args)
		gt.NoError(t, err)

		results := gt.Cast[[]githubtool.IssueSearchResult](t, result["results"])
		t.Logf("Found %d Closed issues/PRs", len(results))
	})
}
