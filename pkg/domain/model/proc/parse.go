package proc

import (
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
)

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Try RFC3339 format first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// Try time only format (e.g. "10:00")
	if t, err := time.Parse("15:04", s); err == nil {
		return time.Date(today.Year(), today.Month(), today.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}

	// Try date only format (e.g. "2/3")
	if parts := strings.Split(s, "/"); len(parts) == 2 {
		if month, err := time.Parse("1", parts[0]); err == nil {
			if day, err := time.Parse("2", parts[1]); err == nil {
				return time.Date(today.Year(), month.Month(), day.Day(), 0, 0, 0, 0, time.Local), nil
			}
		}
	}

	// Try date+time format (e.g. "02-03T00:00")
	if t, err := time.Parse("01-02T15:04", s); err == nil {
		return time.Date(today.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
	}

	switch s {
	case "today":
		return today, nil
	case "yesterday":
		return today.Add(-24 * time.Hour), nil
	}

	return time.Time{}, goerr.New("invalid time format: expected format: RFC3339, time only (15:04), date only (2/3), date+time (02-03T00:00), today, yesterday", goerr.V("time", s))
}
