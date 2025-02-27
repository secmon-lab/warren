package githubapp

import (
	"context"
	"io"
	"net/http"

	"github.com/google/go-github/v69/github"
	"github.com/jferrl/go-githubauth"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/interfaces"
	"golang.org/x/oauth2"
)

type Client struct {
	client     *github.Client
	httpClient *http.Client
}

var _ interfaces.GitHubAppClient = &Client{}

func New(ctx context.Context, appID int64, installationID int64, privateKey []byte) (*Client, error) {
	appTokenSource, err := githubauth.NewApplicationTokenSource(appID, privateKey)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create application token source", goerr.V("appID", appID), goerr.V("installationID", installationID))
	}

	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource)

	httpClient := oauth2.NewClient(ctx, installationTokenSource)

	client := github.NewClient(httpClient)

	return &Client{
		client:     client,
		httpClient: httpClient,
	}, nil
}

func (c *Client) GetDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get repository", goerr.V("owner", owner), goerr.V("repo", repo))
	}
	return repository.GetDefaultBranch(), nil
}

func (c *Client) DownloadArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	// Clone the repository
	archiveURL, _, err := c.client.Repositories.GetArchiveLink(ctx, owner, repo, github.Zipball, &github.RepositoryContentGetOptions{
		Ref: ref,
	}, 1)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to clone repository", goerr.V("owner", owner), goerr.V("repo", repo))
	}

	// Download the archive
	resp, err := c.httpClient.Get(archiveURL.String())
	if err != nil {
		return nil, goerr.Wrap(err, "failed to download archive", goerr.V("url", archiveURL))
	}

	return resp.Body, nil
}

func (c *Client) CommitChanges(ctx context.Context, owner, repo, branch string, files map[string][]byte, message string) error {
	// Get current commit SHA
	ref, _, err := c.client.Git.GetRef(ctx, owner, repo, "refs/heads/"+branch)
	if err != nil {
		return err
	}

	entries := make([]*github.TreeEntry, 0, len(files))

	// Create blobs for each file
	for path, content := range files {
		blob := &github.Blob{
			Content:  github.Ptr(string(content)),
			Encoding: github.Ptr("utf-8"),
		}
		blob, _, err = c.client.Git.CreateBlob(ctx, owner, repo, blob)
		if err != nil {
			return err
		}

		entries = append(entries, &github.TreeEntry{
			Path: github.Ptr(path),
			Mode: github.Ptr("100644"),
			Type: github.Ptr("blob"),
			SHA:  blob.SHA,
		})
	}

	// Create tree with all files
	tree, _, err := c.client.Git.CreateTree(ctx, owner, repo, *ref.Object.SHA, entries)
	if err != nil {
		return err
	}

	// Create commit
	commit := &github.Commit{
		Message: github.Ptr(message),
		Tree:    tree,
		Parents: []*github.Commit{{SHA: ref.Object.SHA}},
	}
	newCommit, _, err := c.client.Git.CreateCommit(ctx, owner, repo, commit, nil)
	if err != nil {
		return err
	}

	// Update reference
	ref.Object.SHA = newCommit.SHA
	_, _, err = c.client.Git.UpdateRef(ctx, owner, repo, ref, false)
	return err
}

func (c *Client) CreateBranch(ctx context.Context, owner, repo, baseBranch, newBranch string) error {
	// Get SHA of base branch
	ref, _, err := c.client.Git.GetRef(ctx, owner, repo, "refs/heads/"+baseBranch)
	if err != nil {
		return err
	}

	// Create new branch
	newRef := &github.Reference{
		Ref:    github.Ptr("refs/heads/" + newBranch),
		Object: &github.GitObject{SHA: ref.Object.SHA},
	}
	_, _, err = c.client.Git.CreateRef(ctx, owner, repo, newRef)
	return err
}

func (c *Client) CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) (*github.PullRequest, error) {
	newPR := &github.NewPullRequest{
		Title: github.Ptr(title),
		Body:  github.Ptr(body),
		Head:  github.Ptr(head),
		Base:  github.Ptr(base),
	}
	pr, _, err := c.client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, err
	}
	return pr, nil
}
