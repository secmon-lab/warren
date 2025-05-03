package base

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/session"
)

func (x *Base) putNote(ctx context.Context, args map[string]any) (map[string]any, error) {
	noteRaw, ok := args["note"]
	if !ok {
		return nil, goerr.New("note is required")
	}

	note, ok := noteRaw.(string)
	if !ok {
		return nil, goerr.New("note is not a string")
	}

	data := session.NewNote(x.sessionID, note)

	if err := x.repo.PutNote(ctx, data); err != nil {
		return nil, goerr.Wrap(err, "failed to put note")
	}

	return map[string]any{
		"note": note,
	}, nil
}

func (x *Base) getNotes(ctx context.Context, args map[string]any) (map[string]any, error) {
	var limit, offset int64

	if limitVal, ok := args["limit"].(float64); ok {
		limit = int64(limitVal)
	}
	if offsetVal, ok := args["offset"].(float64); ok {
		offset = int64(offsetVal)
	}

	notes, err := x.repo.GetNotes(ctx, x.sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get notes")
	}

	// Apply pagination
	if offset > 0 {
		if offset >= int64(len(notes)) {
			notes = nil
		} else {
			notes = notes[offset:]
		}
	}

	if limit > 0 && limit < int64(len(notes)) {
		notes = notes[:limit]
	}

	var rows []string
	for _, note := range notes {
		rows = append(rows, note.Note)
	}

	return map[string]any{
		"notes":  rows,
		"count":  len(notes),
		"offset": offset,
		"limit":  limit,
	}, nil
}
