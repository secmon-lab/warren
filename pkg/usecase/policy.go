package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/opaq"
	"github.com/secmon-lab/warren/pkg/model"
	"github.com/secmon-lab/warren/pkg/prompt"
	"github.com/secmon-lab/warren/pkg/service"
	"github.com/secmon-lab/warren/pkg/service/policy"
	"github.com/secmon-lab/warren/pkg/utils/clock"
	"github.com/secmon-lab/warren/pkg/utils/thread"
)

const (
	maxRetryCountForIgnorePolicy = 8
)

func (uc *UseCases) GenerateIgnorePolicy(ctx context.Context, alerts []model.Alert, note string) (*policy.Service, error) {
	base, err := uc.policyService.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	policyData := model.PolicyData{
		Data:      base.Sources(),
		CreatedAt: clock.Now(ctx),
	}

	p, err := prompt.BuildIgnorePolicyPrompt(ctx, policyData, alerts, note)
	if err != nil {
		return nil, err
	}

	ssn := uc.geminiStartChat()
	var newSvc *policy.Service

	newTestData := uc.policyService.TestData()
	for _, alert := range alerts {
		newTestData.Detect[alert.Schema][alert.ID.String()+".json"] = alert.Data
	}

	for i := 0; i < maxRetryCountForIgnorePolicy && newSvc == nil; i++ {
		resp, err := service.AskChat[prompt.IgnorePolicyPromptResult](ctx, ssn, p)
		if err != nil {
			if goerr.HasTag(err, model.ErrTagInvalidLLMResponse) {
				thread.Reply(ctx, fmt.Sprintf("💥 Failed to generate ignore policy: \n> %v\n\nRetry...", err))
				p = "Your response is invalid. Please try again: " + err.Error()
				continue
			}

			return nil, err
		}

		c, err := opaq.New(opaq.DataMap(resp.Policy))
		if err != nil {
			thread.Reply(ctx, fmt.Sprintf("💥 Failed to build new policy: \n> %v\n\nRetry...", err))
			p = "Failed to build new policy client: " + err.Error()
			continue
		}

		svc := uc.policyService.Clone(c, newTestData)

		if errs := policy.Test(ctx, c, newTestData); len(errs) > 0 {
			var runtimeErrs []error
			var replyLines []string
			for _, err := range errs {
				if goerr.HasTag(err, model.ErrTagTestFailed) {
					replyLines = append(replyLines, "❌ FAILED: "+err.Error())
					p = "Failed to test new policy: " + err.Error()
				} else {
					runtimeErrs = append(runtimeErrs, err)
				}
			}

			if len(runtimeErrs) > 0 {
				return nil, errors.New("failed to test new policy")
			}
			if len(replyLines) > 0 {
				p = "Failed to test new policy:\n" + strings.Join(replyLines, "\n")
				thread.Reply(ctx, strings.Join(replyLines, "\n"))
				continue
			}
		}

		thread.Reply(ctx, "✅ Test PASSED")
		newSvc = svc
	}

	if newSvc == nil {
		thread.Reply(ctx, "🛑 Failed to generate a new ignore policy. Stop generating.")
		return nil, errors.New("failed to generate a new ignore policy")
	}

	return newSvc, nil
}
