package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/m-mizutani/goerr/v2"
)

func (x *Action) runCodeSearch(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, goerr.New("query is required")
	}

	// Build search query
	searchQuery := x.buildCodeSearchQuery(query, args)
	if searchQuery == "" {
		// No repositories matched the filter, return empty result
		return map[string]any{
			"results": []CodeSearchResult{},
			"total":   0,
		}, nil
	}

	// Search options
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	}

	// Execute search
	result, _, err := x.githubClient.Search.Code(ctx, searchQuery, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search code", goerr.V("query", searchQuery))
	}

	// Process results
	var allResults []CodeSearchResult
	for _, item := range result.CodeResults {
		if item.Repository == nil || item.Repository.FullName == nil || item.Path == nil {
			continue
		}

		csr := CodeSearchResult{
			Repository: *item.Repository.FullName,
			Path:       *item.Path,
		}

		if item.HTMLURL != nil {
			csr.HTMLURL = *item.HTMLURL
		}

		// Extract text matches
		if len(item.TextMatches) > 0 {
			for _, tm := range item.TextMatches {
				if tm.Fragment != nil {
					csr.Matches = append(csr.Matches, *tm.Fragment)
				}
			}
		}

		allResults = append(allResults, csr)
	}

	// Use the total from the API response
	total := 0
	if result.Total != nil {
		total = *result.Total
	}

	return map[string]any{
		"results": allResults,
		"total":   total,
	}, nil
}

func (x *Action) buildCodeSearchQuery(baseQuery string, args map[string]any) string {
	var queryParts []string
	queryParts = append(queryParts, baseQuery)

	// Get repository filter if specified
	repoFilterPattern := ""
	if rf, ok := args["repo_filter"].(string); ok && rf != "" {
		repoFilterPattern = strings.ToLower(rf)
	}

	// Add repository filter for configured repos
	var repoFilters []string
	for _, config := range x.configs {
		fullName := fmt.Sprintf("%s/%s", config.Owner, config.Repository)
		if repoFilterPattern != "" && !strings.Contains(strings.ToLower(fullName), repoFilterPattern) {
			continue
		}
		repoFilters = append(repoFilters, fmt.Sprintf("repo:%s", fullName))
	}

	if len(repoFilters) == 0 {
		// No repositories matched the filter
		return ""
	}

	// Add all repo filters to the query
	queryParts = append(queryParts, strings.Join(repoFilters, " "))

	// Add optional filters
	if lang, ok := args["language"].(string); ok && lang != "" {
		queryParts = append(queryParts, fmt.Sprintf("language:%s", lang))
	}

	if path, ok := args["path"].(string); ok && path != "" {
		queryParts = append(queryParts, fmt.Sprintf("path:%s", path))
	}

	if filename, ok := args["filename"].(string); ok && filename != "" {
		queryParts = append(queryParts, fmt.Sprintf("filename:%s", filename))
	}

	return strings.Join(queryParts, " ")
}

func (x *Action) runIssueSearch(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, goerr.New("query is required")
	}

	// Build search query
	searchQuery := x.buildIssueSearchQuery(query, args)
	if searchQuery == "" {
		// No repositories matched the filter, return empty result
		return map[string]any{
			"results": []IssueSearchResult{},
			"total":   0,
		}, nil
	}

	// Search options
	opts := &github.SearchOptions{
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	}

	// Execute search
	result, _, err := x.githubClient.Search.Issues(ctx, searchQuery, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search issues", goerr.V("query", searchQuery))
	}

	// Format results
	results := make([]IssueSearchResult, 0, len(result.Issues))
	for _, issue := range result.Issues {
		if issue.Number == nil || issue.Title == nil {
			continue
		}

		isr := IssueSearchResult{
			Number: *issue.Number,
			Title:  *issue.Title,
		}

		if issue.CreatedAt != nil {
			isr.CreatedAt = issue.CreatedAt.Time
		}
		if issue.UpdatedAt != nil {
			isr.UpdatedAt = issue.UpdatedAt.Time
		}

		// Extract repository from URL
		if issue.RepositoryURL != nil {
			parts := strings.Split(*issue.RepositoryURL, "/")
			if len(parts) >= 2 {
				isr.Repository = fmt.Sprintf("%s/%s", parts[len(parts)-2], parts[len(parts)-1])
			}
		}

		if issue.State != nil {
			isr.State = *issue.State
		}

		if issue.HTMLURL != nil {
			isr.HTMLURL = *issue.HTMLURL
		}

		if issue.User != nil && issue.User.Login != nil {
			isr.User = *issue.User.Login
		}

		if issue.Body != nil {
			// Truncate body if too long
			body := *issue.Body
			if len(body) > 500 {
				body = body[:500] + "..."
			}
			isr.Body = body
		}

		// Check if it's a PR
		isr.IsPR = issue.IsPullRequest()

		// Extract labels
		for _, label := range issue.Labels {
			if label.Name != nil {
				isr.Labels = append(isr.Labels, *label.Name)
			}
		}

		results = append(results, isr)
	}

	// Use the total from the API response
	total := 0
	if result.Total != nil {
		total = *result.Total
	}

	return map[string]any{
		"results": results,
		"total":   total,
	}, nil
}

func (x *Action) buildIssueSearchQuery(baseQuery string, args map[string]any) string {
	var queryParts []string
	queryParts = append(queryParts, baseQuery)

	// Get repository filter if specified
	repoFilterPattern := ""
	if rf, ok := args["repo_filter"].(string); ok && rf != "" {
		repoFilterPattern = strings.ToLower(rf)
	}

	// Add repository filter for configured repos
	var repoFilters []string
	for _, config := range x.configs {
		fullName := fmt.Sprintf("%s/%s", config.Owner, config.Repository)
		if repoFilterPattern != "" && !strings.Contains(strings.ToLower(fullName), repoFilterPattern) {
			continue
		}
		repoFilters = append(repoFilters, fmt.Sprintf("repo:%s", fullName))
	}

	if len(repoFilters) == 0 {
		// No repositories matched the filter
		return ""
	}

	// Add all repo filters to the query
	queryParts = append(queryParts, strings.Join(repoFilters, " "))

	// Add optional filters
	if state, ok := args["state"].(string); ok && state != "" && state != "all" {
		queryParts = append(queryParts, fmt.Sprintf("state:%s", state))
	}

	if author, ok := args["author"].(string); ok && author != "" {
		queryParts = append(queryParts, fmt.Sprintf("author:%s", author))
	}

	if labels, ok := args["labels"].(string); ok && labels != "" {
		// Split comma-separated labels
		for _, label := range strings.Split(labels, ",") {
			label = strings.TrimSpace(label)
			if label != "" {
				queryParts = append(queryParts, fmt.Sprintf("label:\"%s\"", label))
			}
		}
	}

	// Filter by type
	if typeFilter, ok := args["type"].(string); ok && typeFilter != "" && typeFilter != "all" {
		switch typeFilter {
		case "issue":
			queryParts = append(queryParts, "type:issue")
		case "pr":
			queryParts = append(queryParts, "type:pr")
		}
	}

	return strings.Join(queryParts, " ")
}
