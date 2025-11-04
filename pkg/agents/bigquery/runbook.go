package bigquery

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/bigquery"
	"github.com/secmon-lab/warren/pkg/domain/types"
)

// runbookLoader loads SQL runbook files and processes them
type runbookLoader struct {
	paths []string
}

// newRunbookLoader creates a new runbookLoader
func newRunbookLoader(paths []string) *runbookLoader {
	return &runbookLoader{
		paths: paths,
	}
}

// loadRunbooks loads all SQL files from configured paths and returns RunbookEntries
func (r *runbookLoader) loadRunbooks(ctx context.Context) (bigquery.RunbookEntries, error) {
	var entries bigquery.RunbookEntries

	for _, path := range r.paths {
		pathEntries, err := r.loadFromPath(ctx, path)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to load runbooks from path", goerr.V("path", path))
		}
		entries = append(entries, pathEntries...)
	}

	return entries, nil
}

// loadFromPath loads runbooks from a single path (file or directory)
func (r *runbookLoader) loadFromPath(ctx context.Context, path string) (bigquery.RunbookEntries, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to stat path", goerr.V("path", path))
	}

	if info.IsDir() {
		return r.loadFromDirectory(ctx, path)
	}

	// Single file
	entry, err := r.loadFromFile(ctx, path)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, nil // Not a SQL file
	}

	return bigquery.RunbookEntries{entry}, nil
}

// loadFromDirectory loads all SQL files from a directory recursively
func (r *runbookLoader) loadFromDirectory(ctx context.Context, dirPath string) (bigquery.RunbookEntries, error) {
	var entries bigquery.RunbookEntries

	// Clean the base directory path
	cleanDirPath := filepath.Clean(dirPath)

	err := filepath.WalkDir(cleanDirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return goerr.Wrap(err, "failed to walk directory", goerr.V("path", path))
		}

		if d.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}

		entry, err := r.loadFromFile(ctx, path)
		if err != nil {
			return err
		}

		if entry != nil {
			entries = append(entries, entry)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return entries, nil
}

// loadFromFile loads a single SQL file and creates a RunbookEntry
func (r *runbookLoader) loadFromFile(_ context.Context, filePath string) (*bigquery.RunbookEntry, error) {
	// Only process .sql files
	if !strings.HasSuffix(strings.ToLower(filePath), ".sql") {
		return nil, nil
	}

	// Clean the file path to prevent path traversal
	cleanPath := filepath.Clean(filePath)

	// Read file content
	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read SQL file", goerr.V("path", cleanPath))
	}

	// Use UUID for ID to prevent collisions
	id := types.NewRunbookID()

	// Extract filename (base name) for default title
	filename := filepath.Base(cleanPath)

	// Extract title and description from comments
	title, description := r.extractTitleAndDescription(string(content))
	if title == "" {
		// Use filename as title if no title found in comments
		title = strings.TrimSuffix(filename, ".sql")
	}

	entry := &bigquery.RunbookEntry{
		ID:          id,
		Title:       title,
		Description: description,
		SQLContent:  string(content),
	}

	return entry, nil
}

// extractTitleAndDescription extracts title and description from SQL comments
// Format:
// -- Title: Title of the runbook
// -- Description: Description of the runbook
// -- This can span multiple lines
func (r *runbookLoader) extractTitleAndDescription(content string) (string, string) {
	scanner := bufio.NewScanner(strings.NewReader(content))

	var title string
	var description strings.Builder
	var inDescription bool

	titleRegex := regexp.MustCompile(`(?i)^--\s*title\s*:\s*(.+)$`)
	descRegex := regexp.MustCompile(`(?i)^--\s*description\s*:\s*(.+)$`)
	commentRegex := regexp.MustCompile(`^--\s*(.*)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// If we hit a non-comment line, stop processing
		if !strings.HasPrefix(line, "--") {
			break
		}

		// Check for title
		if matches := titleRegex.FindStringSubmatch(line); matches != nil {
			title = strings.TrimSpace(matches[1])
			inDescription = false
			continue
		}

		// Check for description start
		if matches := descRegex.FindStringSubmatch(line); matches != nil {
			inDescription = true
			if description.Len() > 0 {
				description.WriteString(" ")
			}
			description.WriteString(strings.TrimSpace(matches[1]))
			continue
		}

		// If we're in description mode, continue collecting comment lines
		if inDescription {
			if matches := commentRegex.FindStringSubmatch(line); matches != nil {
				if description.Len() > 0 {
					description.WriteString(" ")
				}
				description.WriteString(strings.TrimSpace(matches[1]))
			} else {
				// Non-comment line, stop collecting description
				break
			}
		}
	}

	return title, description.String()
}
