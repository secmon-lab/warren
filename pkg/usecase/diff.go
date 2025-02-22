package usecase

import (
	"fmt"
	"strings"
)

func diffPolicy(oldPolicy, newPolicy map[string]string) string {
	var result string

	// Find modified, added and deleted files
	modifiedFiles := make(map[string]struct{})
	addedFiles := make(map[string]struct{})
	deletedFiles := make(map[string]struct{})

	for fname := range oldPolicy {
		if _, ok := newPolicy[fname]; ok {
			if oldPolicy[fname] != newPolicy[fname] {
				modifiedFiles[fname] = struct{}{}
			}
		} else {
			deletedFiles[fname] = struct{}{}
		}
	}

	for fname := range newPolicy {
		if _, ok := oldPolicy[fname]; !ok {
			addedFiles[fname] = struct{}{}
		}
	}

	// Generate diff for modified files
	for fname := range modifiedFiles {
		oldLines := strings.Split(oldPolicy[fname], "\n")
		newLines := strings.Split(newPolicy[fname], "\n")

		result += fmt.Sprintf("diff old/%s new/%s\n", fname, fname)

		// Simple diff implementation showing changed lines with context
		for i := 0; i < len(oldLines) || i < len(newLines); i++ {
			if i >= len(oldLines) {
				// Added lines in new file
				result += fmt.Sprintf("+ %s\n", newLines[i])
				continue
			}
			if i >= len(newLines) {
				// Deleted lines from old file
				result += fmt.Sprintf("- %s\n", oldLines[i])
				continue
			}
			if oldLines[i] != newLines[i] {
				// Show context (3 lines before)
				start := max(0, i-3)
				for j := start; j < i; j++ {
					result += fmt.Sprintf("  %s\n", oldLines[j])
				}
				result += fmt.Sprintf("- %s\n", oldLines[i])
				result += "---\n"
				result += fmt.Sprintf("+ %s\n", newLines[i])
			}
		}
	}

	// Show added files
	for fname := range addedFiles {
		result += fmt.Sprintf("Only in new: %s\n", fname)
		result += "--- /dev/null\n"
		result += fmt.Sprintf("+++ b/%s\n", fname)
		lines := strings.Split(newPolicy[fname], "\n")
		for _, line := range lines {
			result += fmt.Sprintf("+ %s\n", line)
		}
	}

	// Show deleted files
	for fname := range deletedFiles {
		result += fmt.Sprintf("Only in old: %s\n", fname)
		result += "+++ /dev/null\n"
		result += fmt.Sprintf("--- a/%s\n", fname)
		lines := strings.Split(oldPolicy[fname], "\n")
		for _, line := range lines {
			result += fmt.Sprintf("- %s\n", line)
		}
	}

	return strings.TrimSpace(result)
}
