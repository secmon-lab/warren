package github

// RepositoryConfig represents a GitHub repository hint surfaced to the planner
// via Prompt(). It is advisory only and does not gate API access.
type RepositoryConfig struct {
	Owner         string `yaml:"owner" json:"owner"`
	Repository    string `yaml:"repository" json:"repository"`
	Description   string `yaml:"description" json:"description"`
	DefaultBranch string `yaml:"default_branch,omitempty" json:"default_branch,omitempty"`
}

// Config represents the GitHub repository-hint configuration file.
type Config struct {
	Repositories []*RepositoryConfig `yaml:"repositories" json:"repositories"`
}
