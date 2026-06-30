package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gollem-dev/gollem"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/secmon-lab/warren/pkg/utils/toolset"
	"github.com/urfave/cli/v3"
)

// Mode controls which tool commands are available.
type Mode int

const (
	// ModeReadOnly provides search + tag list only (for plan/child agent).
	ModeReadOnly Mode = iota
	// ModeReadWrite provides search + save + delete + tag management (for reflection agent).
	ModeReadWrite
	// ModeSearchOnly provides search only (for child task agent).
	// Tag list is provided via prompt, so knowledge_tag_list is not needed.
	ModeSearchOnly
)

// Tool provides knowledge_* tool commands for LLM agents.
type Tool struct {
	svc      *svcknowledge.Service
	category types.KnowledgeCategory
	mode     Mode

	tools gollem.ToolSet
}

var _ interfaces.Tool = &Tool{}

// New creates a new knowledge v2 tool.
// category is fixed per use case (e.g., "fact" for investigation context, "technique" for analysis procedures).
func New(svc *svcknowledge.Service, category types.KnowledgeCategory, mode Mode) *Tool {
	x := &Tool{
		svc:      svc,
		category: category,
		mode:     mode,
	}

	// The available tool set depends on the mode: search-only agents get just
	// knowledge_search, read-only agents also get knowledge_tag_list, and
	// read-write agents get the full CRUD + tag-management surface. Building the
	// set here (rather than branching inside Specs/Run) keeps the mode logic in
	// one place while each tool stays type-safe via gollem.NewTool.
	tools := []gollem.Tool{
		gollem.MustNewTool(cmdSearch, descSearch, x.search),
	}
	if mode != ModeSearchOnly {
		tools = append(tools, gollem.MustNewTool(cmdTagList, descTagList, x.tagList))
	}
	if mode == ModeReadWrite {
		tools = append(tools,
			gollem.MustNewTool(cmdSave, descSave, x.save),
			gollem.MustNewTool(cmdDelete, descDelete, x.delete),
			gollem.MustNewTool(cmdList, descList, x.list),
			gollem.MustNewTool(cmdGet, descGet, x.get),
			gollem.MustNewTool(cmdHistory, descHistory, x.history),
			gollem.MustNewTool(cmdTagCreate, descTagCreate, x.tagCreate),
			gollem.MustNewTool(cmdTagUpdate, descTagUpdate, x.tagUpdate),
			gollem.MustNewTool(cmdTagDelete, descTagDelete, x.tagDelete),
			gollem.MustNewTool(cmdTagMerge, descTagMerge, x.tagMerge),
		)
	}
	x.tools = toolset.New(tools...)

	return x
}

func (x *Tool) ID() string {
	return "knowledge"
}

func (x *Tool) Description() string {
	return "Knowledge base search for prior findings and investigation techniques"
}

func (x *Tool) Flags() []cli.Flag                 { return nil }
func (x *Tool) Configure(_ context.Context) error { return nil }
func (x *Tool) Helper() *cli.Command              { return nil }

func (x *Tool) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("category", x.category.String()),
		slog.Int("mode", int(x.mode)),
	)
}

func (x *Tool) Prompt(_ context.Context) (string, error) {
	var sb strings.Builder
	sb.WriteString("## Knowledge Base\n\n")
	fmt.Fprintf(&sb, "Category: `%s`\n\n", x.category)

	if x.category == types.KnowledgeCategoryFact {
		sb.WriteString("Search for factual information about processes, hosts, tools, and known patterns.\n")
	} else {
		sb.WriteString("Search for investigation techniques, tool usage, and analysis procedures.\n")
	}

	sb.WriteString("\n**IMPORTANT**: Before starting your work, use `knowledge_search` to check if relevant knowledge exists.\n")
	if x.mode != ModeSearchOnly {
		sb.WriteString("Specify at least one tag when searching. Use `knowledge_tag_list` to see available tags.\n")
	}

	sb.WriteString("\n**Two-phase workflow**:\n")
	sb.WriteString("1. Use `knowledge_list` or `knowledge_search` to browse entries (returns ID, title, tags only)\n")
	sb.WriteString("2. Use `knowledge_get` with specific IDs to retrieve full details (including claim content)\n")
	sb.WriteString("This approach minimizes token usage while allowing efficient knowledge exploration.\n")

	return sb.String(), nil
}

// Tool command names
const (
	cmdSearch    = "knowledge_search"
	cmdSave      = "knowledge_save"
	cmdDelete    = "knowledge_delete"
	cmdList      = "knowledge_list"
	cmdGet       = "knowledge_get"
	cmdHistory   = "knowledge_history"
	cmdTagList   = "knowledge_tag_list"
	cmdTagCreate = "knowledge_tag_create"
	cmdTagUpdate = "knowledge_tag_update"
	cmdTagDelete = "knowledge_tag_delete"
	cmdTagMerge  = "knowledge_tag_merge"
)

// Tool descriptions. Kept as constants so the typed-tool registration in New
// stays readable and the wire-level descriptions are unchanged.
const (
	descSearch    = "Search for knowledge. Returns lightweight summaries (ID, title, tags only). Use knowledge_get to retrieve full details."
	descTagList   = "List all available knowledge tags with their IDs and descriptions."
	descSave      = "Create or update a knowledge entry. Specify ID to update existing, omit for new."
	descDelete    = "Delete a knowledge entry that contains incorrect information. Records the reason in the log."
	descList      = "List knowledge entries (ID, title, tags only - lightweight for browsing). Use knowledge_get to retrieve full details."
	descGet       = "Get full details (including claim) of one or more knowledge entries by ID."
	descHistory   = "Get change history of a knowledge entry."
	descTagCreate = "Create a new tag."
	descTagUpdate = "Update an existing tag."
	descTagDelete = "Delete a tag. Removes it from all knowledge entries that reference it."
	descTagMerge  = "Merge two tags: replaces old_id with new_id in all knowledge entries, then deletes old_id."
)

// Typed inputs for each tool. The schema is inferred from these struct tags.
type searchInput struct {
	Query string   `json:"query" required:"true" description:"Natural language query to search for"`
	Tags  []string `json:"tags" required:"true" description:"Tag IDs to filter by (at least one required)"`
}

// emptyInput is used for tools that take no arguments (knowledge_tag_list).
type emptyInput struct{}

type saveInput struct {
	ID       string   `json:"id" description:"Knowledge ID (for update). Omit for new entry."`
	Title    string   `json:"title" required:"true" description:"Title of the knowledge entry (topic name)"`
	Claim    string   `json:"claim" required:"true" description:"Markdown content with facts or techniques"`
	Tags     []string `json:"tags" required:"true" description:"Tag IDs to associate (at least one required)"`
	Message  string   `json:"message" required:"true" description:"Reason for this change"`
	TicketID string   `json:"ticket_id" description:"Related ticket ID (optional)"`
}

type deleteInput struct {
	ID     string `json:"id" required:"true" description:"Knowledge ID to delete"`
	Reason string `json:"reason" required:"true" description:"Reason for deletion"`
}

type listInput struct {
	Limit  int64 `json:"limit" description:"Maximum number of entries to return (default: 25, max: 100)"`
	Offset int64 `json:"offset" description:"Number of entries to skip for pagination (default: 0)"`
}

type getInput struct {
	IDs []string `json:"ids" required:"true" description:"Knowledge IDs to retrieve (1-10 IDs)"`
}

type historyInput struct {
	ID string `json:"id" required:"true" description:"Knowledge ID"`
}

type tagCreateInput struct {
	Name        string `json:"name" required:"true" description:"Tag name (lowercase, short)"`
	Description string `json:"description" description:"Description of what this tag represents"`
}

type tagUpdateInput struct {
	ID          string `json:"id" required:"true" description:"Tag ID"`
	Name        string `json:"name" required:"true" description:"New tag name"`
	Description string `json:"description" description:"New description"`
}

type tagDeleteInput struct {
	ID string `json:"id" required:"true" description:"Tag ID to delete"`
}

type tagMergeInput struct {
	OldID string `json:"old_id" required:"true" description:"Tag ID to be merged (will be deleted)"`
	NewID string `json:"new_id" required:"true" description:"Tag ID to merge into (will be kept)"`
}

func (x *Tool) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	return x.tools.Specs(ctx)
}

func (x *Tool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	return x.tools.Run(ctx, name, args)
}

func (x *Tool) search(ctx context.Context, in searchInput) (map[string]any, error) {
	query := in.Query
	if query == "" {
		return nil, goerr.New("query is required")
	}

	tagIDs := toTagIDs(in.Tags)
	if len(tagIDs) == 0 {
		return nil, goerr.New("at least one tag is required")
	}

	results, err := x.svc.SearchKnowledge(ctx, x.category, tagIDs, query, 10)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search knowledge")
	}

	msg.Trace(ctx, "🔍 Found %d knowledge entries", len(results))

	return map[string]any{
		"results": formatKnowledgeSummary(results),
		"count":   len(results),
	}, nil
}

func (x *Tool) save(ctx context.Context, in saveInput) (map[string]any, error) {
	title := in.Title
	claim := in.Claim
	message := in.Message
	ticketID := in.TicketID
	id := in.ID

	tagIDs := toTagIDs(in.Tags)

	k := &knowledgeModel.Knowledge{
		ID:       types.KnowledgeID(id),
		Category: x.category,
		Title:    title,
		Claim:    claim,
		Tags:     tagIDs,
		Author:   types.SystemUserID,
	}

	if err := x.svc.SaveKnowledge(ctx, k, message, types.TicketID(ticketID)); err != nil {
		return nil, goerr.Wrap(err, "failed to save knowledge")
	}

	msg.Trace(ctx, "💾 Saved knowledge '%s' (ID: `%s`)", title, k.ID)

	return map[string]any{
		"success": true,
		"id":      k.ID.String(),
	}, nil
}

func (x *Tool) delete(ctx context.Context, in deleteInput) (map[string]any, error) {
	id := in.ID
	reason := in.Reason
	if id == "" {
		return nil, goerr.New("id is required")
	}
	if reason == "" {
		return nil, goerr.New("reason is required")
	}

	if err := x.svc.DeleteKnowledge(ctx, types.KnowledgeID(id), reason, types.SystemUserID, ""); err != nil {
		return nil, goerr.Wrap(err, "failed to delete knowledge")
	}

	msg.Trace(ctx, "🗑️ Deleted knowledge (ID: `%s`)", id)

	return map[string]any{"success": true}, nil
}

func (x *Tool) list(ctx context.Context, in listInput) (map[string]any, error) {
	// Extract limit and offset parameters
	limit := 25 // default
	offset := 0 // default

	if in.Limit > 0 {
		limit = int(in.Limit)
	}
	if in.Offset > 0 {
		offset = int(in.Offset)
	}

	// Apply max limit
	const maxLimit = 100
	if limit > maxLimit {
		limit = maxLimit
	}
	if limit <= 0 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	// List all tags first to resolve names
	tags, err := x.svc.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}

	// For list, we search across all tags in this category
	if len(tags) == 0 {
		return map[string]any{"entries": []any{}, "count": 0, "returned": 0, "offset": offset}, nil
	}

	allTagIDs := make([]types.KnowledgeTagID, len(tags))
	for i, t := range tags {
		allTagIDs[i] = t.ID
	}

	// We need to query for each tag individually since Firestore array-contains only supports one value
	seen := make(map[types.KnowledgeID]bool)
	var results []*knowledgeModel.Knowledge
	for _, tagID := range allTagIDs {
		// Query more than needed to account for offset + limit
		knowledges, err := x.svc.SearchKnowledge(ctx, x.category, []types.KnowledgeTagID{tagID}, "", offset+limit+50)
		if err != nil {
			continue
		}
		for _, k := range knowledges {
			if !seen[k.ID] {
				seen[k.ID] = true
				results = append(results, k)
			}
		}
	}

	totalCount := len(results)

	// Apply offset and limit
	if offset >= len(results) {
		results = nil
	} else {
		results = results[offset:]
		if len(results) > limit {
			results = results[:limit]
		}
	}

	return map[string]any{
		"entries":  formatKnowledgeSummary(results),
		"count":    totalCount,
		"returned": len(results),
		"offset":   offset,
	}, nil
}

func (x *Tool) get(ctx context.Context, in getInput) (map[string]any, error) {
	if len(in.IDs) == 0 {
		return nil, goerr.New("at least one ID is required")
	}

	// Convert to KnowledgeID slice and deduplicate
	idMap := make(map[types.KnowledgeID]struct{})
	var ids []types.KnowledgeID
	for _, s := range in.IDs {
		id := types.KnowledgeID(s)
		if _, exists := idMap[id]; !exists {
			idMap[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	if len(ids) > 10 {
		return nil, goerr.New("maximum 10 unique IDs allowed")
	}

	knowledges, err := x.svc.GetKnowledges(ctx, ids)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get knowledges")
	}

	return map[string]any{
		"entries": formatKnowledgeDetail(knowledges),
		"count":   len(knowledges),
	}, nil
}

func (x *Tool) history(ctx context.Context, in historyInput) (map[string]any, error) {
	id := in.ID
	if id == "" {
		return nil, goerr.New("id is required")
	}

	logs, err := x.svc.ListKnowledgeLogs(ctx, types.KnowledgeID(id))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list knowledge logs")
	}

	entries := make([]map[string]any, len(logs))
	for i, l := range logs {
		entries[i] = map[string]any{
			"id":         l.ID.String(),
			"author":     l.Author.String(),
			"ticket_id":  l.TicketID.String(),
			"message":    l.Message,
			"created_at": l.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	return map[string]any{"logs": entries, "count": len(entries)}, nil
}

func (x *Tool) tagList(ctx context.Context, _ emptyInput) (map[string]any, error) {
	tags, err := x.svc.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}

	entries := make([]map[string]any, len(tags))
	for i, t := range tags {
		entries[i] = map[string]any{
			"id":          t.ID.String(),
			"name":        t.Name,
			"description": t.Description,
		}
	}

	return map[string]any{"tags": entries, "count": len(entries)}, nil
}

func (x *Tool) tagCreate(ctx context.Context, in tagCreateInput) (map[string]any, error) {
	name := in.Name
	description := in.Description

	tag, err := x.svc.CreateTag(ctx, name, description)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}

	msg.Trace(ctx, "🏷️ Created tag '%s' (ID: `%s`)", name, tag.ID)

	return map[string]any{"success": true, "id": tag.ID.String()}, nil
}

func (x *Tool) tagUpdate(ctx context.Context, in tagUpdateInput) (map[string]any, error) {
	id := in.ID
	name := in.Name
	description := in.Description
	if id == "" {
		return nil, goerr.New("id is required")
	}

	tag, err := x.svc.UpdateTag(ctx, types.KnowledgeTagID(id), name, description)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update tag")
	}

	return map[string]any{"success": true, "id": tag.ID.String()}, nil
}

func (x *Tool) tagDelete(ctx context.Context, in tagDeleteInput) (map[string]any, error) {
	id := in.ID
	if id == "" {
		return nil, goerr.New("id is required")
	}

	if err := x.svc.DeleteTag(ctx, types.KnowledgeTagID(id)); err != nil {
		return nil, goerr.Wrap(err, "failed to delete tag")
	}

	msg.Trace(ctx, "🗑️ Deleted tag (ID: `%s`)", id)

	return map[string]any{"success": true}, nil
}

func (x *Tool) tagMerge(ctx context.Context, in tagMergeInput) (map[string]any, error) {
	oldID := in.OldID
	newID := in.NewID
	if oldID == "" || newID == "" {
		return nil, goerr.New("old_id and new_id are required")
	}

	if err := x.svc.MergeTags(ctx, types.KnowledgeTagID(oldID), types.KnowledgeTagID(newID)); err != nil {
		return nil, goerr.Wrap(err, "failed to merge tags")
	}

	msg.Trace(ctx, "🔀 Merged tag `%s` into `%s`", oldID, newID)

	return map[string]any{"success": true}, nil
}

// toTagIDs converts decoded string tag IDs into the typed KnowledgeTagID slice.
func toTagIDs(tags []string) []types.KnowledgeTagID {
	ids := make([]types.KnowledgeTagID, 0, len(tags))
	for _, s := range tags {
		ids = append(ids, types.KnowledgeTagID(s))
	}
	return ids
}

// formatKnowledgeSummary formats knowledges as lightweight summaries (no claim).
func formatKnowledgeSummary(knowledges []*knowledgeModel.Knowledge) []map[string]any {
	result := make([]map[string]any, len(knowledges))
	for i, k := range knowledges {
		tagStrs := make([]string, len(k.Tags))
		for j, t := range k.Tags {
			tagStrs[j] = t.String()
		}
		result[i] = map[string]any{
			"id":         k.ID.String(),
			"title":      k.Title,
			"tags":       tagStrs,
			"updated_at": k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result
}

// formatKnowledgeDetail formats knowledges with full details (including claim).
func formatKnowledgeDetail(knowledges []*knowledgeModel.Knowledge) []map[string]any {
	result := make([]map[string]any, len(knowledges))
	for i, k := range knowledges {
		tagStrs := make([]string, len(k.Tags))
		for j, t := range k.Tags {
			tagStrs[j] = t.String()
		}
		result[i] = map[string]any{
			"id":         k.ID.String(),
			"category":   string(k.Category),
			"title":      k.Title,
			"claim":      k.Claim,
			"tags":       tagStrs,
			"author":     k.Author.String(),
			"updated_at": k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result
}
