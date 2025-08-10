package github

import (
	"time"
)

// RepositoryConfig represents a GitHub repository configuration
type RepositoryConfig struct {
	Owner         string `yaml:"owner" json:"owner"`
	Repository    string `yaml:"repository" json:"repository"`
	Description   string `yaml:"description" json:"description"`
	DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty"`
}

// Config represents the GitHub tool configuration
type Config struct {
	Repositories []*RepositoryConfig `yaml:"repositories" json:"repositories"`
}

// CodeSearchResult represents a code search result
type CodeSearchResult struct {
	Repository   string    `json:"repository"`
	Path         string    `json:"path"`
	HTMLURL      string    `json:"html_url"`
	Matches      []string  `json:"matches"`
	Language     string    `json:"language,omitempty"`
	LastModified time.Time `json:"last_modified,omitempty"`
}

// IssueSearchResult represents an issue/PR search result
type IssueSearchResult struct {
	Repository string    `json:"repository"`
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	HTMLURL    string    `json:"html_url"`
	User       string    `json:"user"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	IsPR       bool      `json:"is_pr"`
	Body       string    `json:"body,omitempty"`
	Labels     []string  `json:"labels,omitempty"`
}

// ContentResult represents file content result
type ContentResult struct {
	Repository string `json:"repository"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	SHA        string `json:"sha"`
	HTMLURL    string `json:"html_url"`
	Size       int    `json:"size"`
}
