package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/utils/safe"
)

// graphQLRequest represents a GitHub GraphQL API request
type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

// graphQLResponse represents the top-level GraphQL response
type graphQLResponse struct {
	Data   graphQLBlameData `json:"data"`
	Errors []graphQLError   `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLBlameData struct {
	Repository graphQLRepository `json:"repository"`
}

type graphQLRepository struct {
	Object *graphQLObject `json:"object"`
}

type graphQLObject struct {
	Blame *graphQLBlame `json:"blame"`
}

type graphQLBlame struct {
	Ranges []graphQLBlameRange `json:"ranges"`
}

type graphQLBlameRange struct {
	StartingLine int              `json:"startingLine"`
	EndingLine   int              `json:"endingLine"`
	Commit       graphQLCommitRef `json:"commit"`
}

type graphQLCommitRef struct {
	OID     string              `json:"oid"`
	Message string              `json:"message"`
	Author  graphQLCommitAuthor `json:"author"`
}

type graphQLCommitAuthor struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}

const blameQuery = `query($owner: String!, $name: String!, $expression: String!) {
  repository(owner: $owner, name: $name) {
    object(expression: $expression) {
      ... on Blob {
        blame(startingLine: 1) {
          ranges {
            startingLine
            endingLine
            commit {
              oid
              message
              author {
                name
                date
              }
            }
          }
        }
      }
    }
  }
}`

func (x *Action) runGetBlame(ctx context.Context, args map[string]any) (map[string]any, error) {
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
	if !x.isAllowedRepo(owner, repo) {
		return nil, goerr.New("repository not in configured list",
			goerr.V("owner", owner),
			goerr.V("repo", repo))
	}

	// Determine ref
	ref := x.getDefaultBranch(owner, repo)
	if r, ok := args["ref"].(string); ok && r != "" {
		ref = r
	}

	// Build GraphQL expression: "ref:path"
	expression := fmt.Sprintf("%s:%s", ref, path)

	// Execute GraphQL request
	reqBody := graphQLRequest{
		Query: blameQuery,
		Variables: map[string]any{
			"owner":      owner,
			"name":       repo,
			"expression": expression,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to marshal GraphQL request")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create GraphQL request")
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := x.httpClient.Do(httpReq)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to execute GraphQL request",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
			goerr.V("path", path))
	}
	defer safe.Close(ctx, resp.Body)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read GraphQL response")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, goerr.New("GraphQL request failed",
			goerr.V("status", resp.StatusCode),
			goerr.V("body", string(respBody)))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return nil, goerr.Wrap(err, "failed to parse GraphQL response")
	}

	if len(gqlResp.Errors) > 0 {
		return nil, goerr.New("GraphQL errors",
			goerr.V("errors", gqlResp.Errors))
	}

	if gqlResp.Data.Repository.Object == nil || gqlResp.Data.Repository.Object.Blame == nil {
		return nil, goerr.New("no blame data found",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
			goerr.V("path", path),
			goerr.V("ref", ref))
	}

	// Convert to BlameResult
	ranges := make([]BlameRange, 0, len(gqlResp.Data.Repository.Object.Blame.Ranges))
	for _, r := range gqlResp.Data.Repository.Object.Blame.Ranges {
		// Truncate long commit messages
		message := r.Commit.Message
		if len(message) > 200 {
			message = message[:200] + "..."
		}

		ranges = append(ranges, BlameRange{
			StartLine:     r.StartingLine,
			EndLine:       r.EndingLine,
			CommitSHA:     r.Commit.OID,
			CommitMessage: message,
			Author:        r.Commit.Author.Name,
			Date:          r.Commit.Author.Date,
		})
	}

	return map[string]any{
		"repository": fmt.Sprintf("%s/%s", owner, repo),
		"path":       path,
		"ref":        ref,
		"ranges":     ranges,
		"count":      len(ranges),
	}, nil
}

// getDefaultBranch returns the default branch for a configured repository
func (x *Action) getDefaultBranch(owner, repo string) string {
	for _, config := range x.configs {
		if config.Owner == owner && config.Repository == repo {
			if config.DefaultBranch != "" {
				return config.DefaultBranch
			}
			break
		}
	}
	return "main"
}
