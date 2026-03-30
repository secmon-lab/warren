package knowledge_test

import (
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/repository"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/tool/knowledge"
)

func newTestService() *svcknowledge.Service {
	repo := repository.NewMemory()
	return svcknowledge.New(repo, nil)
}

func TestModeSearchOnly_ReturnsOnlySearchSpec(t *testing.T) {
	tool := knowledge.New(newTestService(), types.KnowledgeCategoryFact, knowledge.ModeSearchOnly)

	specs, err := tool.Specs(t.Context())
	gt.NoError(t, err)
	gt.A(t, specs).Length(1)
	gt.V(t, specs[0].Name).Equal("knowledge_search")
}

func TestModeReadOnly_ReturnsSearchAndTagList(t *testing.T) {
	tool := knowledge.New(newTestService(), types.KnowledgeCategoryFact, knowledge.ModeReadOnly)

	specs, err := tool.Specs(t.Context())
	gt.NoError(t, err)
	gt.A(t, specs).Length(2)

	names := make(map[string]bool)
	for _, s := range specs {
		names[s.Name] = true
	}
	gt.True(t, names["knowledge_search"])
	gt.True(t, names["knowledge_tag_list"])
}

func TestModeSearchOnly_PromptOmitsTagListInstruction(t *testing.T) {
	tool := knowledge.New(newTestService(), types.KnowledgeCategoryFact, knowledge.ModeSearchOnly)

	p, err := tool.Prompt(t.Context())
	gt.NoError(t, err)
	gt.True(t, strings.Contains(p, "knowledge_search"))
	gt.True(t, !strings.Contains(p, "knowledge_tag_list"))
}
