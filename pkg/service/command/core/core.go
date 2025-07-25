package core

import (
	"context"
	"strconv"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/warren/pkg/domain/interfaces"
	"github.com/secmon-lab/warren/pkg/domain/model/alert"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
	"github.com/secmon-lab/warren/pkg/utils/msg"
)

type Clients struct {
	repo   interfaces.Repository
	llm    gollem.LLMClient
	thread interfaces.SlackThreadService
}

func NewClients(repo interfaces.Repository, llm gollem.LLMClient, thread interfaces.SlackThreadService) *Clients {
	return &Clients{
		repo:   repo,
		llm:    llm,
		thread: thread,
	}
}

func (s *Clients) Repo() interfaces.Repository {
	return s.repo
}

func (s *Clients) LLM() gollem.LLMClient {
	return s.llm
}

func (s *Clients) Thread() interfaces.SlackThreadService {
	return s.thread
}

func (x *Clients) CreateList(ctx context.Context, thread slack.Thread, user *slack.User, alerts alert.Alerts) (*alert.List, error) {
	ctx = msg.Trace(ctx, "🤖 Creating alert list...")
	list := alert.NewList(ctx, thread, user, alerts)

	if err := list.FillMetadata(ctx, x.llm); err != nil {
		return nil, goerr.Wrap(err, "failed to fill metadata")
	}

	// Register the list to the repository
	if err := x.repo.PutAlertList(ctx, list); err != nil {
		return nil, goerr.Wrap(err, "failed to put alert list")
	}

	return list, nil
}

func ParseTime(timeStr string) (time.Time, error) {
	// Try parsing as time format (HH:MM)
	if t, err := time.Parse("15:04", timeStr); err == nil {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location()), nil
	}

	// Try parsing as date format (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", timeStr); err == nil {
		return t, nil
	}

	return time.Time{}, goerr.New("invalid time format", goerr.V("time", timeStr))
}

func ParseDuration(durationStr string) (time.Duration, error) {
	// Parse duration like "10m", "1h", "1d"
	unit := durationStr[len(durationStr)-1:]
	value, err := strconv.Atoi(durationStr[:len(durationStr)-1])
	if err != nil {
		return 0, goerr.Wrap(err, "failed to parse duration value")
	}

	switch unit {
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	default:
		return 0, goerr.New("invalid duration unit", goerr.V("unit", unit))
	}
}
