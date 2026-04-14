package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
)

func (x *Action) runListCommits(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Parse required parameters
	owner, ok := args["owner"].(string)
	if !ok || owner == "" {
		return nil, goerr.New("owner is required")
	}

	repo, ok := args["repo"].(string)
	if !ok || repo == "" {
		return nil, goerr.New("repo is required")
	}

	// Check if the repository is in our configured list
	if !x.isAllowedRepo(owner, repo) {
		return nil, goerr.New("repository not in configured list",
			goerr.V("owner", owner),
			goerr.V("repo", repo))
	}

	// Build options
	opts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	}

	// Optional parameters
	if sha, ok := args["sha"].(string); ok && sha != "" {
		opts.SHA = sha
	}

	if path, ok := args["path"].(string); ok && path != "" {
		opts.Path = path
	}

	if author, ok := args["author"].(string); ok && author != "" {
		opts.Author = author
	}

	if perPage, ok := args["per_page"].(float64); ok && perPage > 0 {
		pp := int(perPage)
		if pp > 100 {
			pp = 100
		}
		opts.PerPage = pp
	}

	if page, ok := args["page"].(float64); ok && page > 0 {
		opts.Page = int(page)
	}

	// Execute API call
	commits, _, err := x.githubClient.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list commits",
			goerr.V("owner", owner),
			goerr.V("repo", repo))
	}

	// Format results
	results := make([]CommitResult, 0, len(commits))
	for _, commit := range commits {
		cr := CommitResult{}

		if commit.SHA != nil {
			cr.SHA = *commit.SHA
		}

		if commit.HTMLURL != nil {
			cr.HTMLURL = *commit.HTMLURL
		}

		if commit.Commit != nil {
			if commit.Commit.Message != nil {
				cr.Message = *commit.Commit.Message
			}

			if commit.Commit.Author != nil {
				if commit.Commit.Author.Name != nil {
					cr.Author = *commit.Commit.Author.Name
				}
				if commit.Commit.Author.Date != nil {
					cr.Date = commit.Commit.Author.Date.Time
				}
			}
		}

		// Use login name if available (preferred over commit author name)
		if commit.Author != nil && commit.Author.Login != nil {
			cr.Author = *commit.Author.Login
		}

		cr.FilesChanged = len(commit.Files)

		results = append(results, cr)
	}

	return map[string]any{
		"repository": fmt.Sprintf("%s/%s", owner, repo),
		"commits":    results,
		"count":      len(results),
	}, nil
}

// isAllowedRepo checks if the repository is in the configured list
func (x *Action) isAllowedRepo(owner, repo string) bool {
	for _, config := range x.configs {
		if config.Owner == owner && config.Repository == repo {
			return true
		}
	}
	return false
}
