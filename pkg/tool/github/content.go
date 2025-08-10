package github

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
)

func (x *Action) runGetContent(ctx context.Context, args map[string]any) (map[string]any, error) {
	// Parse required parameters
	owner, ok := args["owner"].(string)
	if !ok || owner == "" {
		return nil, goerr.New("owner is required")
	}

	repo, ok := args["repo"].(string)
	if !ok || repo == "" {
		return nil, goerr.New("repo is required")
	}

	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, goerr.New("path is required")
	}

	// Check if the repository is in our configured list
	allowed := false
	for _, config := range x.configs {
		if config.Owner == owner && config.Repository == repo {
			allowed = true
			break
		}
	}

	if !allowed {
		return nil, goerr.New("repository not in configured list",
			goerr.V("owner", owner),
			goerr.V("repo", repo))
	}

	// Prepare options
	opts := &github.RepositoryContentGetOptions{}

	// Optional ref parameter
	if ref, ok := args["ref"].(string); ok && ref != "" {
		opts.Ref = ref
	}

	// Get content
	fileContent, _, _, err := x.githubClient.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get content",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
			goerr.V("path", path))
	}

	if fileContent == nil {
		return nil, goerr.New("no content found")
	}

	// Decode content
	var content string
	if fileContent.Content != nil {
		decoded, err := base64.StdEncoding.DecodeString(*fileContent.Content)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to decode content")
		}
		content = string(decoded)
	}

	result := ContentResult{
		Repository: fmt.Sprintf("%s/%s", owner, repo),
		Path:       path,
		Content:    content,
	}

	if fileContent.SHA != nil {
		result.SHA = *fileContent.SHA
	}

	if fileContent.HTMLURL != nil {
		result.HTMLURL = *fileContent.HTMLURL
	}

	if fileContent.Size != nil {
		result.Size = *fileContent.Size
	}

	return map[string]any{
		"repository": result.Repository,
		"path":       result.Path,
		"content":    result.Content,
		"sha":        result.SHA,
		"html_url":   result.HTMLURL,
		"size":       result.Size,
	}, nil
}
