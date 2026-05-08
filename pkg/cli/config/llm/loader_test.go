package llm_test

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/cli/config/llm"
)

func parseFile(t *testing.T, body string) *llm.File {
	t.Helper()
	var f llm.File
	_, err := toml.Decode(body, &f)
	gt.NoError(t, err)
	return &f
}

func TestRenderTemplate_MissingEnvFails(t *testing.T) {
	t.Setenv("WARREN_PRESENT_KEY", "x")
	body := `claude = { api_key = "{{ .Env.WARREN_LLM_TEST_MISSING_KEY }}" }`
	_, err := llm.RenderTemplateForTest([]byte(body))
	gt.Error(t, err)
}

func TestRenderTemplate_PresentEnvSubstitutes(t *testing.T) {
	t.Setenv("WARREN_LLM_TEST_KEY", "abcdef")
	body := `value = "{{ .Env.WARREN_LLM_TEST_KEY }}"`
	out, err := llm.RenderTemplateForTest([]byte(body))
	gt.NoError(t, err)
	gt.True(t, strings.Contains(string(out), "abcdef"))
}

func TestValidate_OKVertexClaude(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "p"
task = ["p"]

[embedding]
provider = "gemini"
model = "text-embedding-004"
project_id = "proj"
location = "us-central1"

[[llm]]
id = "p"
description = "primary claude"
provider = "claude"
model = "claude-sonnet"
claude = { project_id = "proj", location = "us-east5" }
`)
	gt.NoError(t, llm.ValidateForTest(f))
}

func TestValidate_OKAPIKeyClaude(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "p"
task = ["p"]

[embedding]
provider = "gemini"
model = "m"
project_id = "proj"
location = "us-central1"

[[llm]]
id = "p"
description = "claude api"
provider = "claude"
model = "m"
claude = { api_key = "AAAA" }
`)
	gt.NoError(t, llm.ValidateForTest(f))
}

func TestValidate_OKVertexGeminiTask(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "p"
task = ["p", "fast"]

[embedding]
provider = "gemini"
model = "m"
project_id = "proj"
location = "us-central1"

[[llm]]
id = "p"
description = "primary"
provider = "claude"
model = "claude-sonnet"
claude = { project_id = "proj", location = "us-east5" }

[[llm]]
id = "fast"
description = "fast gemini"
provider = "gemini"
model = "gemini-flash"
gemini = { project_id = "proj", location = "us-central1", thinking_budget = 0 }
`)
	gt.NoError(t, llm.ValidateForTest(f))
}

// --- Top-level errors ---

func TestValidate_AgentSectionMissing(t *testing.T) {
	f := parseFile(t, `
[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[agent]"))
}

func TestValidate_NoLLMEntries(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[[llm]]"))
}

func TestValidate_EmbeddingSectionMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[embedding]"))
}

// --- [agent] errors ---

func TestValidate_AgentMainEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = ""
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "agent"))
}

func TestValidate_AgentMainUnknownID(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "ghost"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "agent"))
}

func TestValidate_AgentTaskEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = []

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "task"))
}

func TestValidate_AgentTaskUnknownID(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x", "ghost"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[agent].task"))
}

func TestValidate_AgentTaskDuplicate(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x", "x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "duplicate"))
}

// --- [[llm]] entry errors ---

func TestValidate_LLMIDEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = ""
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "empty id"))
}

func TestValidate_LLMIDDuplicate(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "duplicated"))
}

func TestValidate_LLMDescriptionEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = ""
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "description"))
}

func TestValidate_LLMProviderEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = ""
model = "m"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "provider"))
}

func TestValidate_LLMProviderInvalid(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "foo"
model = "m"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "claude") || strings.Contains(err.Error(), "gemini"))
}

func TestValidate_LLMModelEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = ""
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "model"))
}

func TestValidate_LLMSectionMismatch(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
gemini = { project_id = "p", location = "l" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "must not include gemini"))
}

func TestValidate_LLMSectionMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "requires claude"))
}

// --- claude options ---

func TestValidate_ClaudeAmbiguousMode(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { project_id = "p", location = "l", api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "ambiguous"))
}

func TestValidate_ClaudeVertexProjectIDMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { location = "us-east5" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "project_id"))
}

func TestValidate_ClaudeVertexLocationMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { project_id = "p" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "location"))
}

func TestValidate_ClaudeBothModesMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = {}
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "either"))
}

// --- gemini options ---

func TestValidate_GeminiAPIKeyRejected(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "gemini"
model = "m"
gemini = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "gemini api_key"))
}

func TestValidate_GeminiVertexBothMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "gemini"
model = "m"
gemini = {}
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
}

func TestValidate_GeminiProjectIDMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "gemini"
model = "m"
gemini = { location = "us-central1" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "project_id"))
}

func TestValidate_GeminiLocationMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "gemini"
model = "m"
gemini = { project_id = "p" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "location"))
}

// --- embedding ---

func TestValidate_EmbeddingProviderNotGemini(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "claude"
model = "m"
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[embedding] provider"))
}

func TestValidate_EmbeddingModelEmpty(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = ""
project_id = "p"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[embedding] model"))
}

func TestValidate_EmbeddingAPIKeyRejected(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
api_key = "K"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "api_key"))
}

func TestValidate_EmbeddingVertexProjectIDMissing(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "x"
task = ["x"]

[embedding]
provider = "gemini"
model = "m"
location = "l"

[[llm]]
id = "x"
description = "d"
provider = "claude"
model = "m"
claude = { api_key = "K" }
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "[embedding] project_id"))
}

func TestValidate_MultipleErrorsAggregated(t *testing.T) {
	// Trigger several issues at once and verify all surface in the joined error.
	f := parseFile(t, `
[agent]
main = "ghost"
task = []

[[llm]]
id = ""
description = ""
provider = "foo"
model = ""
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	msg := err.Error()
	// Spot-check several distinct errors are present.
	gt.True(t, strings.Contains(msg, "[embedding]"))
	gt.True(t, strings.Contains(msg, "task"))
}

func TestValidate_OKNoopProvider(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "n"
task = ["n"]

[[llm]]
id          = "n"
description = "test"
provider    = "noop"
model       = "noop"

[embedding]
provider = "noop"
model    = "noop"
`)
	gt.NoError(t, llm.ValidateForTest(f))
}

func TestValidate_NoopRejectsNestedSections(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "n"
task = ["n"]

[[llm]]
id          = "n"
description = "test"
provider    = "noop"
model       = "noop"
claude      = { api_key = "x" }

[embedding]
provider = "noop"
model    = "noop"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "noop"))
}

func TestValidate_OKOpenAI(t *testing.T) {
	t.Setenv("WARREN_TEST_OPENAI_KEY", "sk-test")
	out, err := llm.RenderTemplateForTest([]byte(`
[agent]
main = "o"
task = ["o"]

[[llm]]
id          = "o"
description = "openai"
provider    = "openai"
model       = "gpt-4o-mini"
openai      = { api_key = "{{ .Env.WARREN_TEST_OPENAI_KEY }}" }

[embedding]
provider = "openai"
model    = "text-embedding-3-small"
api_key  = "{{ .Env.WARREN_TEST_OPENAI_KEY }}"
`))
	gt.NoError(t, err)
	f := parseFile(t, string(out))
	gt.NoError(t, llm.ValidateForTest(f))
}

func TestValidate_OpenAIRequiresAPIKey(t *testing.T) {
	f := parseFile(t, `
[agent]
main = "o"
task = ["o"]

[[llm]]
id          = "o"
description = "openai"
provider    = "openai"
model       = "gpt-4o"
openai      = { api_key = "" }

[embedding]
provider = "noop"
model    = "noop"
`)
	err := llm.ValidateForTest(f)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "openai.api_key"))
}
