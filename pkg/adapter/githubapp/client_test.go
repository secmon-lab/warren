package githubapp_test

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/adapter/githubapp"
	"github.com/secmon-lab/warren/pkg/utils/test"
)

func genTestClient(t *testing.T) *githubapp.Client {
	appID := os.Getenv("TEST_GITHUB_APP_ID")
	privateKey := os.Getenv("TEST_GITHUB_PRIVATE_KEY")
	installationID := os.Getenv("TEST_GITHUB_INSTALLATION_ID")

	if appID == "" || privateKey == "" || installationID == "" {
		t.Skip("TEST_GITHUB_APP_ID or TEST_GITHUB_PRIVATE_KEY is not set")
	}

	appIDInt, err := strconv.ParseInt(appID, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse appID: %v", err)
	}

	installationIDInt, err := strconv.ParseInt(installationID, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse installationID: %v", err)
	}

	client, err := githubapp.New(t.Context(), appIDInt, installationIDInt, []byte(privateKey))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	return client
}

func TestGitHubCreatePR(t *testing.T) {
	vars := test.NewEnvVars(t, "TEST_GITHUB_OWNER", "TEST_GITHUB_REPO")
	client := genTestClient(t)

	owner := vars.Get("TEST_GITHUB_OWNER")
	repo := vars.Get("TEST_GITHUB_REPO")

	ctx := t.Context()

	defaultBranch, err := client.GetDefaultBranch(ctx, owner, repo)
	gt.NoError(t, err)

	// Clone default branch
	archive, err := client.DownloadArchive(ctx, owner, repo, defaultBranch)
	if err != nil {
		t.Fatalf("failed to clone default branch: %v", err)
	}
	defer archive.Close()

	archiveData, err := io.ReadAll(archive)
	gt.NoError(t, err)

	tmpDir := t.TempDir()
	archiveReader := bytes.NewReader(archiveData)

	// unzip the archive
	zipReader, err := zip.NewReader(archiveReader, int64(len(archiveData)))
	gt.NoError(t, err)

	// Extract all files from zip to tmpDir
	for _, file := range zipReader.File {
		filePath := filepath.Join(tmpDir, file.Name)
		t.Logf("filePath: %s -> %s", file.Name, filePath)

		// Create directory if needed
		if file.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			gt.NoError(t, err)
			continue
		}

		// Ensure parent directory exists
		err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		gt.NoError(t, err)

		// Create file
		dstFile, err := os.Create(filePath)
		gt.NoError(t, err)
		defer dstFile.Close()

		// Open zip file
		srcFile, err := file.Open()
		gt.NoError(t, err)
		defer srcFile.Close()

		// Copy contents
		_, err = io.Copy(dstFile, srcFile)
		gt.NoError(t, err)
	}

	// Create new branch
	newBranch := "test-branch/" + uuid.New().String()
	err = client.CreateBranch(ctx, owner, repo, defaultBranch, newBranch)
	if err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	// Add test line to README.md
	content := []byte("test\n")
	files := map[string][]byte{
		"README.md": content,
		"newfile":   []byte("new file"),
		"test.md":   []byte("Hello!\n"),
	}

	err = client.CommitChanges(ctx, owner, repo, newBranch, files, "Add test line")
	if err != nil {
		t.Fatalf("failed to commit changes: %v", err)
	}

	// Create pull request
	title := "Test PR: " + time.Now().Format("2006-01-02 15:04:05")
	pr, err := client.CreatePullRequest(ctx, owner, repo, title, "Add test line to README", newBranch, defaultBranch)
	if err != nil {
		t.Fatalf("failed to create pull request: %v", err)
	}

	if pr == nil {
		t.Fatal("pull request is nil")
	}
}
