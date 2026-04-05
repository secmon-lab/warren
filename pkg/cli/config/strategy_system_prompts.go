package config

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// PromptEntry represents a user-defined system prompt loaded from a markdown file.
type PromptEntry struct {
	ID          string // frontmatter id (required, unique)
	Name        string // frontmatter name (required, human-readable display name)
	Description string // frontmatter description (required)
	Content     string // markdown body after frontmatter
	FilePath    string // source file path (for debugging/logging)
}

// promptFrontmatter is the YAML frontmatter structure in prompt files.
type promptFrontmatter struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// StrategySystemPrompts holds CLI configuration for loading multiple user system prompt files.
type StrategySystemPrompts struct {
	dirPath string
}

// NewStrategySystemPrompts creates a StrategySystemPrompts with the given directory path.
// This is primarily for testing; in production, use Flags() to configure via CLI.
func NewStrategySystemPrompts(dirPath string) *StrategySystemPrompts {
	return &StrategySystemPrompts{dirPath: dirPath}
}

// Flags returns CLI flags for user system prompts configuration.
func (x *StrategySystemPrompts) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "strategy-system-prompts",
			Usage:       "Path to a directory containing strategy system prompt files (markdown with YAML frontmatter)",
			Destination: &x.dirPath,
			Sources:     cli.EnvVars("WARREN_STRATEGY_SYSTEM_PROMPTS"),
		},
	}
}

// Configure reads all .md files in the configured directory and returns parsed PromptEntry slice.
// Returns empty slice if no directory is configured.
// Returns error if the directory doesn't exist, files are malformed, or ids are duplicated.
func (x *StrategySystemPrompts) Configure() ([]PromptEntry, error) {
	if x.dirPath == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(x.dirPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read user system prompts directory",
			goerr.V("path", x.dirPath),
		)
	}

	prompts := make([]PromptEntry, 0, len(entries))
	seenIDs := make(map[string]string) // id -> filePath

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		filePath := filepath.Join(x.dirPath, entry.Name())
		prompt, err := parsePromptFile(filePath)
		if err != nil {
			return nil, err
		}

		// Check for duplicate ids
		if existingFile, exists := seenIDs[prompt.ID]; exists {
			return nil, goerr.New("duplicate prompt id",
				goerr.V("id", prompt.ID),
				goerr.V("file1", existingFile),
				goerr.V("file2", filePath),
			)
		}
		seenIDs[prompt.ID] = filePath

		prompts = append(prompts, *prompt)
	}

	return prompts, nil
}

// parsePromptFile reads a markdown file with YAML frontmatter and returns a PromptEntry.
func parsePromptFile(filePath string) (*PromptEntry, error) {
	data, err := os.ReadFile(filePath) // #nosec G304 -- filePath comes from CLI flag or directory listing, not user input
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read prompt file",
			goerr.V("path", filePath),
		)
	}

	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to parse frontmatter",
			goerr.V("path", filePath),
		)
	}

	var meta promptFrontmatter
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return nil, goerr.Wrap(err, "failed to parse YAML frontmatter",
			goerr.V("path", filePath),
		)
	}

	if meta.ID == "" {
		return nil, goerr.New("prompt file missing required 'id' in frontmatter",
			goerr.V("path", filePath),
		)
	}
	if meta.Name == "" {
		return nil, goerr.New("prompt file missing required 'name' in frontmatter",
			goerr.V("path", filePath),
		)
	}
	if meta.Description == "" {
		return nil, goerr.New("prompt file missing required 'description' in frontmatter",
			goerr.V("path", filePath),
		)
	}

	return &PromptEntry{
		ID:          meta.ID,
		Name:        meta.Name,
		Description: meta.Description,
		Content:     strings.TrimSpace(body),
		FilePath:    filePath,
	}, nil
}

// splitFrontmatter splits a markdown file into YAML frontmatter and body.
// Frontmatter is delimited by "---" lines at the start of the file.
func splitFrontmatter(data []byte) ([]byte, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Expand buffer for large prompt files (default 64KB may be insufficient).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// First line must be "---"
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, "", goerr.Wrap(err, "failed to read file")
		}
		return nil, "", goerr.New("empty file")
	}
	if strings.TrimSpace(scanner.Text()) != "---" {
		return nil, "", goerr.New("file does not start with frontmatter delimiter '---'")
	}

	// Read until closing "---"
	var fmLines []string
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			found = true
			break
		}
		fmLines = append(fmLines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, "", goerr.Wrap(err, "failed to read frontmatter")
	}
	if !found {
		return nil, "", goerr.New("frontmatter closing delimiter '---' not found")
	}

	fm := []byte(strings.Join(fmLines, "\n"))

	// Rest is body
	var bodyLines []string
	for scanner.Scan() {
		bodyLines = append(bodyLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, "", goerr.Wrap(err, "failed to read file body")
	}

	return fm, strings.Join(bodyLines, "\n"), nil
}
