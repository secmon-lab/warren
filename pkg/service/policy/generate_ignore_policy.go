package policy

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/errs"
	"github.com/secmon-lab/warren/pkg/domain/model/policy"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/prompt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/llm"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

const (
	maxRetryCountForIgnorePolicy = 8
)

type GenerateIgnorePolicyInput struct {
	// Required, Repo is the repository to get alerts from.
	Repo interfaces.Repository

	// Required, LLM is the LLM client to use.
	LLM interfaces.LLMQuery

	// Required, PolicyClient is the policy client to use.
	PolicyClient interfaces.PolicyClient

	// Required, Source is the source of alert data to generate ignore policy for.
	Source source.Source

	// Optional, TestDataSet is the test data set to use.
	TestDataSet *policy.TestDataSet

	// Optional, Prompt is the prompt to use for the ignore policy.
	Prompt string
}

func (x GenerateIgnorePolicyInput) Validate() error {
	if x.Repo == nil {
		return goerr.New("Repo is required")
	}
	if x.LLM == nil {
		return goerr.New("LLM is required")
	}
	if x.PolicyClient == nil {
		return goerr.New("PolicyClient is required")
	}
	if x.Source == nil {
		return goerr.New("Source is required")
	}

	return nil
}

func GenerateIgnorePolicy(ctx context.Context, input GenerateIgnorePolicyInput) (*policy.Diff, error) {
	if err := input.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid GenerateIgnorePolicy input")
	}

	logger := logging.From(ctx)

	ctx = msg.NewTrace(ctx, "📝 Generating ignore policy...")

	alerts, err := input.Source(ctx, input.Repo)
	if err != nil {
		return nil, err
	}

	diffID := policy.NewPolicyDiffID()
	contents := policy.Contents(input.PolicyClient.Sources())
	p, err := prompt.BuildIgnorePolicyPrompt(ctx, contents, alerts, input.Prompt)
	if err != nil {
		return nil, err
	}

	newTestData := policy.NewTestDataSet()
	allTestData := input.TestDataSet
	if allTestData == nil {
		allTestData = policy.NewTestDataSet()
	}
	for _, a := range alerts {
		fpath := filepath.Join(diffID.String(), a.ID.String()+".json")
		newTestData.Ignore.Add(a.Schema, fpath, a.Data)
		allTestData.Ignore.Add(a.Schema, fpath, a.Data)
	}

	// Generate ignore policy
	var validResult *prompt.IgnorePolicyPromptResult
	for i := 0; i < maxRetryCountForIgnorePolicy && validResult == nil; i++ {
		resp, err := llm.Ask[prompt.IgnorePolicyPromptResult](ctx, input.LLM, p)
		if err != nil {
			if goerr.HasTag(err, errs.TagInvalidLLMResponse) {
				ctx = msg.Trace(ctx, "💥 Failed to generate ignore policy: \n> %v\n\nRetry...", err)
				p = "Your response is invalid. Please try again: " + err.Error()
				continue
			}

			return nil, err
		}
		logger.Debug("generated policy", "policy", resp.Policy)

		formattedPolicy, err := formatPolicy(resp.Policy)
		if err != nil {
			ctx = msg.Trace(ctx, "💥 Invalid Rego policy format: \n> %v\n\nRetry...", err)
			p = "Invalid Rego policy format: " + err.Error() + "\n\nPlease retry."
			continue
		}

		c, err := opaq.New(opaq.DataMap(formattedPolicy))
		if err != nil {
			ctx = msg.Trace(ctx, "💥 Failed to build new policy: \n> %v\n\nRetry...", err)
			p = "Failed to build new policy client: " + err.Error()
			continue
		}

		if errors := allTestData.Test(ctx, c.Query); len(errors) > 0 {
			var runtimeErrs []error
			var replyLines []string
			for _, err := range errors {
				if goerr.HasTag(err, errs.TagTestFailed) {
					replyLines = append(replyLines, "❌ FAILED: "+err.Error())
					logger.Debug("test failed", "error", err)
					p = "Failed to test new policy: " + err.Error()
				} else {
					runtimeErrs = append(runtimeErrs, err)
				}
			}

			if len(runtimeErrs) > 0 {
				return nil, goerr.New("failed to test new policy", goerr.V("errors", runtimeErrs))
			}
			if len(replyLines) > 0 {
				lines := strings.Join(replyLines, "\n")
				p = "Failed to test new policy:\n> " + lines
				ctx = msg.Trace(ctx, "💥 Failed to test new policy: \n> %v\n\nRetry...", lines)
				continue
			}
		}

		ctx = msg.Trace(ctx, "✅ Test PASSED")
		validResult = resp
		validResult.Policy = formattedPolicy
	}

	if validResult == nil {
		_ = msg.Trace(ctx, "🛑 Failed to generate a new ignore policy. Stop generating.")
		return nil, goerr.New("failed to generate a new ignore policy")
	}

	// Fill generated README for test data
	ctx = msg.Trace(ctx, "📝 Generating metadata for test data...")
	alertSchemaMap := make(map[types.AlertSchema][]*alert.Alert)
	for _, alert := range alerts {
		alertSchemaMap[alert.Schema] = append(alertSchemaMap[alert.Schema], alert)
	}

	for schema, alerts := range alertSchemaMap {
		p, err := prompt.BuildTestDataReadmePrompt(ctx, "ignore", alerts)
		if err != nil {
			return nil, err
		}

		resp, err := llm.Ask[prompt.TestDataReadmePromptResult](ctx, input.LLM, p)
		if err != nil {
			ctx = msg.Trace(ctx, "💥 Failed to generate README for test data: \n> %v\n\nSkip README generation.", err)
		} else {
			if newTestData.Ignore.Metafiles[schema] == nil {
				newTestData.Ignore.Metafiles[schema] = make(map[string]string)
			}
			fpath := filepath.Join(diffID.String(), "README.md")
			newTestData.Ignore.Metafiles[schema][fpath] = resp.Content
		}
	}
	ctx = msg.Trace(ctx, "✅ Successfully generated metadata for test data")

	result := policy.NewDiff(ctx,
		validResult.Title,
		validResult.Description,
		validResult.Policy,
		policy.Contents(input.PolicyClient.Sources()),
		newTestData,
	)

	return result, nil
}

func formatPolicy(policy map[string]string) (map[string]string, error) {
	formattedPolicy := make(map[string]string)
	for fileName, contents := range policy {
		formatted, err := formatRegoPolicy(fileName, []byte(contents))
		if err != nil {
			return nil, err
		}
		formattedPolicy[fileName] = string(formatted)
	}
	return formattedPolicy, nil
}

func formatRegoPolicy(fileName string, contents []byte) ([]byte, error) {
	var opts format.Opts
	opts.ParserOptions = &ast.ParserOptions{
		RegoVersion: ast.RegoV1,
	}

	formatted, err := format.SourceWithOpts(fileName, contents, opts)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to format rego policy", goerr.V("fileName", fileName), goerr.V("contents", string(contents)))
	}

	return formatted, nil
}
