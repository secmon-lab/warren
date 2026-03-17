package knowledge

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/service/knowledge"
	"github.com/secmon-lab/warren/pkg/utils/msg"
	"github.com/urfave/cli/v3"
)

type Knowledge struct {
	svc   *knowledge.Service
	topic types.KnowledgeTopic
}

var _ interfaces.Tool = &Knowledge{}

func New(repo interfaces.Repository, topic types.KnowledgeTopic) *Knowledge {
	return &Knowledge{
		svc:   knowledge.New(repo),
		topic: topic,
	}
}

// SetTopic sets the topic from current alert/ticket context
func (x *Knowledge) SetTopic(topic types.KnowledgeTopic) {
	x.topic = topic
}

func (x *Knowledge) Name() string {
	return "knowledge"
}

func (x *Knowledge) Flags() []cli.Flag {
	return nil
}

func (x *Knowledge) Configure(ctx context.Context) error {
	return nil
}

func (x *Knowledge) Helper() *cli.Command {
	return nil
}

func (x *Knowledge) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("topic", x.topic.String()),
	)
}

func (x *Knowledge) Prompt(ctx context.Context) (string, error) {
	topicInfo := "Current topic: `" + x.topic.String() + "`"
	if x.topic == "" {
		topicInfo = "Topic is dynamic: specify the `topic` parameter in each command. Use a specific topic name if known, or `general` for general-purpose knowledge."
	}

	return `## Knowledge Management

**CRITICAL**: Save/store/remember requests MUST use tools, not just words.

### Requests Requiring Tool Execution
- Save/store/remember → ` + "`knowledge_save`" + `
- Delete/archive → ` + "`knowledge_archive`" + `

### Examples

❌ WRONG:
User: "Remember this alert needs no action"
Agent: {"needs_plan": false, "direct_response": "Understood"}
→ Nothing is saved!

✅ CORRECT:
User: "Remember this alert needs no action"
Agent: {"needs_plan": true, "tasks": [{"description": "Use knowledge_save..."}]}
→ Actually saved.

### Workflow
1. List: ` + "`knowledge_list`" + `
2. Check existing: ` + "`knowledge_get`" + `
3. Save/update: ` + "`knowledge_save`" + `
4. Confirm with commit ID

` + topicInfo, nil
}

const (
	cmdListKnowledgeSlugs = "knowledge_list"
	cmdGetKnowledges      = "knowledge_get"
	cmdSaveKnowledge      = "knowledge_save"
	cmdArchiveKnowledge   = "knowledge_archive"
)

func (x *Knowledge) Specs(ctx context.Context) ([]gollem.ToolSpec, error) {
	topicParam := &gollem.Parameter{
		Type:        gollem.TypeString,
		Description: "Knowledge topic to operate on. Use a specific topic name if known, or 'general' for general-purpose knowledge.",
		Required:    x.topic == "",
	}

	return []gollem.ToolSpec{
		{
			Name:        cmdListKnowledgeSlugs,
			Description: "List all knowledge slugs and names in the current topic. Use this to see what knowledges are available before retrieving or updating them.",
			Parameters: map[string]*gollem.Parameter{
				"topic": topicParam,
			},
		},
		{
			Name:        cmdGetKnowledges,
			Description: "Get knowledge content. If slug is provided, get that specific knowledge. If slug is not provided, get all knowledges in the current topic.",
			Parameters: map[string]*gollem.Parameter{
				"slug": {
					Type:        gollem.TypeString,
					Description: "Optional. Specific knowledge slug to retrieve. If omitted, all knowledges are retrieved.",
				},
				"topic": topicParam,
			},
		},
		{
			Name:        cmdSaveKnowledge,
			Description: "Save or update a knowledge. This creates a new version (append-only). Always retrieve the existing knowledge first before updating. Content size is limited to 10KB total per topic.",
			Parameters: map[string]*gollem.Parameter{
				"slug": {
					Type:        gollem.TypeString,
					Description: "Unique identifier for the knowledge within the topic",
					Required:    true,
				},
				"name": {
					Type:        gollem.TypeString,
					Description: "Human-readable name (max 100 characters)",
					Required:    true,
				},
				"content": {
					Type:        gollem.TypeString,
					Description: "The knowledge content",
					Required:    true,
				},
				"topic": topicParam,
			},
		},
		{
			Name:        cmdArchiveKnowledge,
			Description: "Archive a knowledge (logical delete). Archived knowledges are not retrieved and do not count towards quota. Use this to free up space when quota is exceeded.",
			Parameters: map[string]*gollem.Parameter{
				"slug": {
					Type:        gollem.TypeString,
					Description: "Slug of the knowledge to archive",
					Required:    true,
				},
				"topic": topicParam,
			},
		},
	}, nil
}

// resolveTopic returns the effective topic. If a fixed topic is set (ticket-bound chat),
// it takes precedence. Otherwise, the topic from the command arguments is used,
// falling back to "general".
func (x *Knowledge) resolveTopic(args map[string]any) types.KnowledgeTopic {
	if x.topic != "" {
		return x.topic
	}
	if topic, ok := args["topic"].(string); ok && topic != "" {
		return types.KnowledgeTopic(topic)
	}
	return types.KnowledgeTopic("general")
}

func (x *Knowledge) Run(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	switch name {
	case cmdListKnowledgeSlugs:
		return x.listSlugs(ctx, args)
	case cmdGetKnowledges:
		return x.getKnowledges(ctx, args)
	case cmdSaveKnowledge:
		return x.saveKnowledge(ctx, args)
	case cmdArchiveKnowledge:
		return x.archiveKnowledge(ctx, args)
	default:
		return nil, goerr.New("unknown command", goerr.V("name", name))
	}
}

func (x *Knowledge) listSlugs(ctx context.Context, args map[string]any) (map[string]any, error) {
	slugs, err := x.svc.ListSlugs(ctx, x.resolveTopic(args))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list knowledge slugs")
	}

	msg.Trace(ctx, "📚 Found *%d* knowledge entries", len(slugs))

	return map[string]any{
		"slugs": slugs,
	}, nil
}

func (x *Knowledge) getKnowledges(ctx context.Context, args map[string]any) (map[string]any, error) {
	slug, _ := args["slug"].(string)
	topic := x.resolveTopic(args)

	if slug != "" {
		// Get specific knowledge
		k, err := x.svc.GetKnowledge(ctx, topic, types.KnowledgeSlug(slug))
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get knowledge")
		}
		if k == nil {
			msg.Trace(ctx, "📚 Knowledge *%s* not found", slug)
			return map[string]any{
				"found":   false,
				"message": fmt.Sprintf("Knowledge with slug '%s' not found", slug),
			}, nil
		}

		msg.Trace(ctx, "📚 Retrieved knowledge *%s* (ID: `%s`, version: `%s`)",
			k.Name, k.Slug, k.CommitID[:8])

		return map[string]any{
			"found":     true,
			"knowledge": k,
		}, nil
	}

	// Get all knowledges
	knowledges, err := x.svc.GetKnowledges(ctx, topic)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get knowledges")
	}

	msg.Trace(ctx, "📚 Retrieved *%d* knowledge entries", len(knowledges))

	return map[string]any{
		"knowledges": knowledges,
	}, nil
}

func (x *Knowledge) saveKnowledge(ctx context.Context, args map[string]any) (map[string]any, error) {
	slug, ok := args["slug"].(string)
	if !ok || slug == "" {
		return nil, goerr.New("slug is required")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, goerr.New("name is required")
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return nil, goerr.New("content is required")
	}

	// Author is always system when updated by agent
	commitID, err := x.svc.SaveKnowledge(ctx, x.resolveTopic(args), types.KnowledgeSlug(slug), name, content, types.SystemUserID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to save knowledge")
	}

	msg.Trace(ctx, "📚 Saved knowledge *%s* (ID: `%s`, version: `%s`)",
		name, slug, commitID[:8])

	return map[string]any{
		"success":   true,
		"commit_id": commitID,
		"message":   fmt.Sprintf("Knowledge saved successfully with commit ID: %s", commitID),
	}, nil
}

func (x *Knowledge) archiveKnowledge(ctx context.Context, args map[string]any) (map[string]any, error) {
	slug, ok := args["slug"].(string)
	if !ok || slug == "" {
		return nil, goerr.New("slug is required")
	}

	if err := x.svc.ArchiveKnowledge(ctx, x.resolveTopic(args), types.KnowledgeSlug(slug)); err != nil {
		return nil, goerr.Wrap(err, "failed to archive knowledge")
	}

	msg.Trace(ctx, "📚 Archived knowledge (ID: `%s`)", slug)

	return map[string]any{
		"success": true,
		"message": fmt.Sprintf("Knowledge '%s' archived successfully", slug),
	}, nil
}
