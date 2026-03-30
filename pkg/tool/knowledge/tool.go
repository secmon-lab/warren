package knowledge

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	knowledgeModel "github.com/secmon-lab/warren/pkg/domain/model/knowledge"
	"github.com/secmon-lab/warren/pkg/domain/types"
	svcknowledge "github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/msg"
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
}

var _ interfaces.Tool = &Tool{}

// New creates a new knowledge v2 tool.
// category is fixed per use case (e.g., "fact" for investigation context, "technique" for analysis procedures).
func New(svc *svcknowledge.Service, category types.KnowledgeCategory, mode Mode) *Tool {
	return &Tool{
		svc:      svc,
		category: category,
		mode:     mode,
	}
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

func (x *Tool) Specs(_ context.Context) ([]gollem.ToolSpec, error) {
	searchSpec := gollem.ToolSpec{
		Name:        cmdSearch,
		Description: "Search for knowledge. Returns relevant entries matching the query, filtered by tags.",
		Parameters: map[string]*gollem.Parameter{
			"query": {
				Type:        gollem.TypeString,
				Description: "Natural language query to search for",
				Required:    true,
			},
			"tags": {
				Type:        gollem.TypeArray,
				Items:       &gollem.Parameter{Type: gollem.TypeString},
				Description: "Tag IDs to filter by (at least one required)",
				Required:    true,
			},
		},
	}

	if x.mode == ModeSearchOnly {
		return []gollem.ToolSpec{searchSpec}, nil
	}

	specs := []gollem.ToolSpec{
		searchSpec,
		{
			Name:        cmdTagList,
			Description: "List all available knowledge tags with their IDs and descriptions.",
		},
	}

	if x.mode == ModeReadWrite {
		specs = append(specs,
			gollem.ToolSpec{
				Name:        cmdSave,
				Description: "Create or update a knowledge entry. Specify ID to update existing, omit for new.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Knowledge ID (for update). Omit for new entry.",
					},
					"title": {
						Type:        gollem.TypeString,
						Description: "Title of the knowledge entry (topic name)",
						Required:    true,
					},
					"claim": {
						Type:        gollem.TypeString,
						Description: "Markdown content with facts or techniques",
						Required:    true,
					},
					"tags": {
						Type:        gollem.TypeArray,
						Items:       &gollem.Parameter{Type: gollem.TypeString},
						Description: "Tag IDs to associate (at least one required)",
						Required:    true,
					},
					"message": {
						Type:        gollem.TypeString,
						Description: "Reason for this change",
						Required:    true,
					},
					"ticket_id": {
						Type:        gollem.TypeString,
						Description: "Related ticket ID (optional)",
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdDelete,
				Description: "Delete a knowledge entry that contains incorrect information. Records the reason in the log.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Knowledge ID to delete",
						Required:    true,
					},
					"reason": {
						Type:        gollem.TypeString,
						Description: "Reason for deletion",
						Required:    true,
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdList,
				Description: "List knowledge entries (titles and tags only, lightweight).",
			},
			gollem.ToolSpec{
				Name:        cmdGet,
				Description: "Get full details of a specific knowledge entry.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Knowledge ID",
						Required:    true,
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdHistory,
				Description: "Get change history of a knowledge entry.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Knowledge ID",
						Required:    true,
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdTagCreate,
				Description: "Create a new tag.",
				Parameters: map[string]*gollem.Parameter{
					"name": {
						Type:        gollem.TypeString,
						Description: "Tag name (lowercase, short)",
						Required:    true,
					},
					"description": {
						Type:        gollem.TypeString,
						Description: "Description of what this tag represents",
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdTagUpdate,
				Description: "Update an existing tag.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Tag ID",
						Required:    true,
					},
					"name": {
						Type:        gollem.TypeString,
						Description: "New tag name",
						Required:    true,
					},
					"description": {
						Type:        gollem.TypeString,
						Description: "New description",
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdTagDelete,
				Description: "Delete a tag. Removes it from all knowledge entries that reference it.",
				Parameters: map[string]*gollem.Parameter{
					"id": {
						Type:        gollem.TypeString,
						Description: "Tag ID to delete",
						Required:    true,
					},
				},
			},
			gollem.ToolSpec{
				Name:        cmdTagMerge,
				Description: "Merge two tags: replaces old_id with new_id in all knowledge entries, then deletes old_id.",
				Parameters: map[string]*gollem.Parameter{
					"old_id": {
						Type:        gollem.TypeString,
						Description: "Tag ID to be merged (will be deleted)",
						Required:    true,
					},
					"new_id": {
						Type:        gollem.TypeString,
						Description: "Tag ID to merge into (will be kept)",
						Required:    true,
					},
				},
			},
		)
	}

	return specs, nil
}

func (x *Tool) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case cmdSearch:
		return x.search(ctx, args)
	case cmdTagList:
		return x.tagList(ctx)
	case cmdSave:
		return x.save(ctx, args)
	case cmdDelete:
		return x.delete(ctx, args)
	case cmdList:
		return x.list(ctx)
	case cmdGet:
		return x.get(ctx, args)
	case cmdHistory:
		return x.history(ctx, args)
	case cmdTagCreate:
		return x.tagCreate(ctx, args)
	case cmdTagUpdate:
		return x.tagUpdate(ctx, args)
	case cmdTagDelete:
		return x.tagDelete(ctx, args)
	case cmdTagMerge:
		return x.tagMerge(ctx, args)
	default:
		return nil, goerr.New("unknown command", goerr.V("name", name))
	}
}

func (x *Tool) search(ctx context.Context, args map[string]any) (map[string]any, error) {
	query, _ := args["query"].(string)
	if query == "" {
		return nil, goerr.New("query is required")
	}

	tagIDs, err := extractTagIDs(args, "tags")
	if err != nil {
		return nil, err
	}
	if len(tagIDs) == 0 {
		return nil, goerr.New("at least one tag is required")
	}

	results, err := x.svc.SearchKnowledge(ctx, x.category, tagIDs, query, 10)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to search knowledge")
	}

	msg.Trace(ctx, "🔍 Found %d knowledge entries", len(results))

	return map[string]any{
		"results": formatKnowledgeList(results),
		"count":   len(results),
	}, nil
}

func (x *Tool) save(ctx context.Context, args map[string]any) (map[string]any, error) {
	title, _ := args["title"].(string)
	claim, _ := args["claim"].(string)
	message, _ := args["message"].(string)
	ticketID, _ := args["ticket_id"].(string)
	id, _ := args["id"].(string)

	tagIDs, err := extractTagIDs(args, "tags")
	if err != nil {
		return nil, err
	}

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

func (x *Tool) delete(ctx context.Context, args map[string]any) (map[string]any, error) {
	id, _ := args["id"].(string)
	reason, _ := args["reason"].(string)
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

func (x *Tool) list(ctx context.Context) (map[string]any, error) {
	// List all tags first to resolve names
	tags, err := x.svc.ListTags(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tags")
	}

	// For list, we search across all tags in this category
	if len(tags) == 0 {
		return map[string]any{"entries": []any{}, "count": 0}, nil
	}

	allTagIDs := make([]types.KnowledgeTagID, len(tags))
	for i, t := range tags {
		allTagIDs[i] = t.ID
	}

	// We need to query for each tag individually since Firestore array-contains only supports one value
	seen := make(map[types.KnowledgeID]bool)
	var results []*knowledgeModel.Knowledge
	for _, tagID := range allTagIDs {
		knowledges, err := x.svc.SearchKnowledge(ctx, x.category, []types.KnowledgeTagID{tagID}, "", 100)
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

	return map[string]any{
		"entries": formatKnowledgeList(results),
		"count":   len(results),
	}, nil
}

func (x *Tool) get(ctx context.Context, args map[string]any) (map[string]any, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return nil, goerr.New("id is required")
	}

	k, err := x.svc.GetKnowledge(ctx, types.KnowledgeID(id))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get knowledge")
	}
	if k == nil {
		return map[string]any{"found": false}, nil
	}

	return map[string]any{
		"found":      true,
		"id":         k.ID.String(),
		"category":   string(k.Category),
		"title":      k.Title,
		"claim":      k.Claim,
		"tags":       k.Tags,
		"author":     k.Author.String(),
		"updated_at": k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (x *Tool) history(ctx context.Context, args map[string]any) (map[string]any, error) {
	id, _ := args["id"].(string)
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

func (x *Tool) tagList(ctx context.Context) (map[string]any, error) {
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

func (x *Tool) tagCreate(ctx context.Context, args map[string]any) (map[string]any, error) {
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)

	tag, err := x.svc.CreateTag(ctx, name, description)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create tag")
	}

	msg.Trace(ctx, "🏷️ Created tag '%s' (ID: `%s`)", name, tag.ID)

	return map[string]any{"success": true, "id": tag.ID.String()}, nil
}

func (x *Tool) tagUpdate(ctx context.Context, args map[string]any) (map[string]any, error) {
	id, _ := args["id"].(string)
	name, _ := args["name"].(string)
	description, _ := args["description"].(string)
	if id == "" {
		return nil, goerr.New("id is required")
	}

	tag, err := x.svc.UpdateTag(ctx, types.KnowledgeTagID(id), name, description)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to update tag")
	}

	return map[string]any{"success": true, "id": tag.ID.String()}, nil
}

func (x *Tool) tagDelete(ctx context.Context, args map[string]any) (map[string]any, error) {
	id, _ := args["id"].(string)
	if id == "" {
		return nil, goerr.New("id is required")
	}

	if err := x.svc.DeleteTag(ctx, types.KnowledgeTagID(id)); err != nil {
		return nil, goerr.Wrap(err, "failed to delete tag")
	}

	msg.Trace(ctx, "🗑️ Deleted tag (ID: `%s`)", id)

	return map[string]any{"success": true}, nil
}

func (x *Tool) tagMerge(ctx context.Context, args map[string]any) (map[string]any, error) {
	oldID, _ := args["old_id"].(string)
	newID, _ := args["new_id"].(string)
	if oldID == "" || newID == "" {
		return nil, goerr.New("old_id and new_id are required")
	}

	if err := x.svc.MergeTags(ctx, types.KnowledgeTagID(oldID), types.KnowledgeTagID(newID)); err != nil {
		return nil, goerr.Wrap(err, "failed to merge tags")
	}

	msg.Trace(ctx, "🔀 Merged tag `%s` into `%s`", oldID, newID)

	return map[string]any{"success": true}, nil
}

// extractTagIDs extracts tag IDs from tool arguments.
func extractTagIDs(args map[string]any, key string) ([]types.KnowledgeTagID, error) {
	raw, ok := args[key]
	if !ok {
		return nil, nil
	}

	arr, ok := raw.([]any)
	if !ok {
		return nil, goerr.New("tags must be an array")
	}

	ids := make([]types.KnowledgeTagID, 0, len(arr))
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return nil, goerr.New("tag ID must be a string")
		}
		ids = append(ids, types.KnowledgeTagID(s))
	}
	return ids, nil
}

// formatKnowledgeList formats knowledges for tool output.
func formatKnowledgeList(knowledges []*knowledgeModel.Knowledge) []map[string]any {
	result := make([]map[string]any, len(knowledges))
	for i, k := range knowledges {
		tagStrs := make([]string, len(k.Tags))
		for j, t := range k.Tags {
			tagStrs[j] = t.String()
		}
		result[i] = map[string]any{
			"id":         k.ID.String(),
			"title":      k.Title,
			"claim":      k.Claim,
			"tags":       tagStrs,
			"updated_at": k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}
	return result
}
