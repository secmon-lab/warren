package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config"
)

func writePromptFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	gt.NoError(t, err)
}

func TestUserSystemPrompts_LoadMultipleFiles(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "security.md", `---
id: security-investigation
name: Security Investigation
description: Security threat investigation
---

Investigate security threats thoroughly.
`)
	writePromptFile(t, dir, "infra.md", `---
id: infra-incident
name: Infrastructure Incident
description: Infrastructure incident investigation
---

Focus on availability and performance.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	prompts, err := cfg.Configure()
	gt.NoError(t, err)
	gt.A(t, prompts).Length(2)

	byID := make(map[string]config.PromptEntry)
	for _, p := range prompts {
		byID[p.ID] = p
	}

	gt.V(t, byID["security-investigation"].Name).Equal("Security Investigation")
	gt.V(t, byID["security-investigation"].Description).Equal("Security threat investigation")
	gt.V(t, byID["security-investigation"].Content).Equal("Investigate security threats thoroughly.")
	gt.V(t, byID["infra-incident"].Name).Equal("Infrastructure Incident")
	gt.V(t, byID["infra-incident"].Description).Equal("Infrastructure incident investigation")
	gt.V(t, byID["infra-incident"].Content).Equal("Focus on availability and performance.")
}

func TestUserSystemPrompts_ExtraFrontmatterFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "extra.md", `---
id: with-extra
name: Extra Fields
description: Has extra fields
author: someone
version: 2
---

Content here.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	prompts, err := cfg.Configure()
	gt.NoError(t, err)
	gt.A(t, prompts).Length(1)
	gt.V(t, prompts[0].ID).Equal("with-extra")
	gt.V(t, prompts[0].Name).Equal("Extra Fields")
	gt.V(t, prompts[0].Description).Equal("Has extra fields")
	gt.V(t, prompts[0].Content).Equal("Content here.")
}

func TestUserSystemPrompts_MissingID(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "no-id.md", `---
name: No ID
description: Missing id field
---

Content.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "missing required 'id'"))
}

func TestUserSystemPrompts_MissingName(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "no-name.md", `---
id: no-name
description: Missing name field
---

Content.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "missing required 'name'"))
}

func TestUserSystemPrompts_MissingDescription(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "no-desc.md", `---
id: no-description
name: No Description
---

Content.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "missing required 'description'"))
}

func TestUserSystemPrompts_DuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "first.md", `---
id: duplicate-id
name: First
description: First file
---

First content.
`)
	writePromptFile(t, dir, "second.md", `---
id: duplicate-id
name: Second
description: Second file
---

Second content.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "duplicate prompt id"))
}

func TestUserSystemPrompts_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "bad.md", `---
id: [invalid yaml
description: broken
---

Content.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "failed to parse YAML frontmatter"))
}

func TestUserSystemPrompts_NonExistentDirectory(t *testing.T) {
	cfg := config.NewStrategySystemPrompts("/nonexistent/path/to/prompts")
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "failed to read user system prompts directory"))
}

func TestUserSystemPrompts_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	cfg := config.NewStrategySystemPrompts(dir)
	prompts, err := cfg.Configure()
	gt.NoError(t, err)
	gt.V(t, len(prompts)).Equal(0)
}

func TestUserSystemPrompts_NonMdFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "valid.md", `---
id: valid
name: Valid Prompt
description: Valid prompt
---

Content.
`)
	writePromptFile(t, dir, "readme.txt", "not a prompt")
	writePromptFile(t, dir, "config.yaml", "key: value")

	cfg := config.NewStrategySystemPrompts(dir)
	prompts, err := cfg.Configure()
	gt.NoError(t, err)
	gt.A(t, prompts).Length(1)
	gt.V(t, prompts[0].ID).Equal("valid")
}

func TestUserSystemPrompts_EmptyDirPath(t *testing.T) {
	cfg := config.NewStrategySystemPrompts("")
	prompts, err := cfg.Configure()
	gt.NoError(t, err)
	gt.V(t, prompts == nil).Equal(true)
}

func TestUserSystemPrompts_NoFrontmatterDelimiter(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "no-fm.md", `# Just a markdown file

No frontmatter here.
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "does not start with frontmatter delimiter"))
}

func TestUserSystemPrompts_UnclosedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writePromptFile(t, dir, "unclosed.md", `---
id: unclosed
description: Missing closing delimiter
`)

	cfg := config.NewStrategySystemPrompts(dir)
	_, err := cfg.Configure()
	gt.V(t, err).NotNil()
	gt.True(t, strings.Contains(err.Error(), "closing delimiter '---' not found"))
}
