package diagnosis

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	diagnosismodel "github.com/secmon-lab/warren/pkg/domain/model/diagnosis"
	"github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
)

const RuleIDLegacyKnowledge diagnosismodel.RuleID = "legacy_knowledge"

// LegacyKnowledgeRule detects old-format knowledge entries that need migration to the new format.
type LegacyKnowledgeRule struct {
	knowledgeSvc *svcknowledge.Service
}

// NewLegacyKnowledgeRule creates a new LegacyKnowledgeRule.
func NewLegacyKnowledgeRule(knowledgeSvc *svcknowledge.Service) *LegacyKnowledgeRule {
	return &LegacyKnowledgeRule{
		knowledgeSvc: knowledgeSvc,
	}
}

func (r *LegacyKnowledgeRule) ID() diagnosismodel.RuleID {
	return RuleIDLegacyKnowledge
}

func (r *LegacyKnowledgeRule) Check(ctx context.Context, repo interfaces.Repository) ([]diagnosismodel.Issue, error) {
	legacyKnowledges, err := repo.ListLegacyKnowledges(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list legacy knowledges")
	}

	var issues []diagnosismodel.Issue
	for _, lk := range legacyKnowledges {
		targetID := lk.Topic + "/" + lk.Slug
		issue := diagnosismodel.NewIssue(
			types.EmptyDiagnosisID,
			RuleIDLegacyKnowledge,
			targetID,
			fmt.Sprintf("Legacy knowledge %q needs migration to new format", targetID),
		)
		issues = append(issues, issue)
	}

	return issues, nil
}

func (r *LegacyKnowledgeRule) Fix(ctx context.Context, repo interfaces.Repository, issue diagnosismodel.Issue) error {
	// Find the specific legacy knowledge matching this issue's TargetID
	legacyKnowledges, err := repo.ListLegacyKnowledges(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to list legacy knowledges")
	}

	var target *interfaces.LegacyKnowledge
	for _, lk := range legacyKnowledges {
		if lk.Topic+"/"+lk.Slug == issue.TargetID {
			target = lk
			break
		}
	}

	if target == nil {
		return goerr.New("legacy knowledge not found",
			goerr.V("target_id", issue.TargetID))
	}

	// Create or get a tag from the topic name
	tag, err := r.knowledgeSvc.CreateTag(ctx, target.Topic, "Migrated from legacy topic: "+target.Topic)
	if err != nil {
		return goerr.Wrap(err, "failed to create tag for legacy topic",
			goerr.V("topic", target.Topic))
	}

	// Build author; use "system" if empty
	author := types.UserID(target.Author)
	if strings.TrimSpace(target.Author) == "" {
		author = types.UserID("system")
	}

	// Create new knowledge with mapped fields
	k := &knowledge.Knowledge{
		Category:  types.KnowledgeCategoryFact,
		Title:     target.Name,
		Claim:     target.Content,
		Tags:      []types.KnowledgeTagID{tag.ID},
		Author:    author,
		CreatedAt: target.CreatedAt,
		UpdatedAt: target.UpdatedAt,
	}

	// SaveKnowledge generates a new ID, sets embedding, validates, and creates a log
	if err := r.knowledgeSvc.SaveKnowledge(ctx, k, "Migrated from legacy knowledge", ""); err != nil {
		return goerr.Wrap(err, "failed to save migrated knowledge",
			goerr.V("target_id", issue.TargetID))
	}

	return nil
}
