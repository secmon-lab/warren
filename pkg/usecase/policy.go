package usecase

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/format"
	"github.com/secmon-lab/warren/pkg/domain/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/service/source"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/logging"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

const (
	maxRetryCountForIgnorePolicy = 8
)

func (uc *UseCases) GenerateIgnorePolicy(ctx context.Context, src source.Source, query string) (*model.PolicyDiff, error) {
	logger := logging.From(ctx)

	thread.Reply(ctx, "📝 Generating ignore policy...")

	alerts, err := src(ctx, uc.repository)
	if err != nil {
		return nil, err
	}

	diffID := model.NewPolicyDiffID()
	base, err := uc.policyService.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	policyData := model.PolicyData{
		Data:      base.Sources(),
		CreatedAt: clock.Now(ctx),
	}

	p, err := prompt.BuildIgnorePolicyPrompt(ctx, policyData, alerts, query)
	if err != nil {
		return nil, err
	}

	newTestData := model.NewTestDataSet()
	allTestData := uc.policyService.TestDataSet()
	if allTestData == nil {
		allTestData = model.NewTestDataSet()
	}
	for _, alert := range alerts {
		fpath := filepath.Join(diffID.String(), alert.ID.String()+".json")
		newTestData.Ignore.Add(alert.Schema, fpath, alert.Data)
		allTestData.Ignore.Add(alert.Schema, fpath, alert.Data)
	}

	// Generate ignore policy
	ssn := uc.llmClient.StartChat()

	var validResult *prompt.IgnorePolicyPromptResult
	for i := 0; i < maxRetryCountForIgnorePolicy && validResult == nil; i++ {
		resp, err := service.AskChat[prompt.IgnorePolicyPromptResult](ctx, ssn, p)
		if err != nil {
			if goerr.HasTag(err, model.ErrTagInvalidLLMResponse) {
				thread.Reply(ctx, fmt.Sprintf("💥 Failed to generate ignore policy: \n> %v\n\nRetry...", err))
				p = "Your response is invalid. Please try again: " + err.Error()
				continue
			}

			return nil, err
		}
		logger.Debug("generated policy", "policy", resp.Policy)

		formattedPolicy, err := formatPolicy(resp.Policy)
		if err != nil {
			thread.Reply(ctx, fmt.Sprintf("💥 Invalid Rego policy format: \n> %v\n\nRetry...", err))
			p = "Invalid Rego policy format: " + err.Error() + "\n\nPlease retry."
			continue
		}

		c, err := opaq.New(opaq.DataMap(formattedPolicy))
		if err != nil {
			thread.Reply(ctx, fmt.Sprintf("💥 Failed to build new policy: \n> %v\n\nRetry...", err))
			p = "Failed to build new policy client: " + err.Error()
			continue
		}

		if errs := policy.Test(ctx, c, allTestData); len(errs) > 0 {
			var runtimeErrs []error
			var replyLines []string
			for _, err := range errs {
				if goerr.HasTag(err, model.ErrTagTestFailed) {
					replyLines = append(replyLines, "❌ FAILED: "+err.Error())
					logger.Debug("test failed", "error", err)
					p = "Failed to test new policy: " + err.Error()
				} else {
					runtimeErrs = append(runtimeErrs, err)
				}
			}

			if len(runtimeErrs) > 0 {
				return nil, errors.New("failed to test new policy")
			}
			if len(replyLines) > 0 {
				p = "Failed to test new policy:\n> " + strings.Join(replyLines, "\n> ")
				thread.Reply(ctx, strings.Join(replyLines, "\n"))
				continue
			}
		}

		thread.Reply(ctx, "✅ Test PASSED")
		validResult = resp
		validResult.Policy = formattedPolicy
	}

	if validResult == nil {
		thread.Reply(ctx, "🛑 Failed to generate a new ignore policy. Stop generating.")
		return nil, errors.New("failed to generate a new ignore policy")
	}

	// Fill generated README for test data
	thread.Reply(ctx, "📝 Generating metadata for test data...")
	alertSchemaMap := make(map[string][]model.Alert)
	for _, alert := range alerts {
		alertSchemaMap[alert.Schema] = append(alertSchemaMap[alert.Schema], alert)
	}

	for schema, alerts := range alertSchemaMap {
		p, err := prompt.BuildTestDataReadmePrompt(ctx, "ignore", alerts)
		if err != nil {
			return nil, err
		}

		resp, err := service.AskPrompt[prompt.TestDataReadmePromptResult](ctx, uc.llmClient, p)
		if err != nil {
			thread.Reply(ctx, fmt.Sprintf("💥 Failed to generate README for test data: \n> %v\n\nSkip README generation.", err))
			logger.Warn("failed to generate test data readme, not fill readme", "error", err)
			continue
		}

		if newTestData.Ignore.Metafiles[schema] == nil {
			newTestData.Ignore.Metafiles[schema] = make(map[string]string)
		}
		fpath := filepath.Join(diffID.String(), "README.md")
		newTestData.Ignore.Metafiles[schema][fpath] = resp.Content
	}
	thread.Reply(ctx, "✅ Successfully generated metadata for test data")

	result := model.NewPolicyDiff(ctx,
		diffID,
		validResult.Title,
		validResult.Description,
		validResult.Policy,
		uc.policyService.Sources(),
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
