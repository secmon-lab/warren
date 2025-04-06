package list

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/warren/pkg/domain/model/source"
	"github.com/secmon-lab/warren/pkg/domain/types"
	"github.com/secmon-lab/warren/pkg/utils/clock"
)

func parseArgsToSource(ctx context.Context, args []string) (source.Source, error) {
	if len(args) == 0 {
		return source.Span(clock.Now(ctx).Add(-24*time.Hour), clock.Now(ctx)), nil
	}

	// Check for known commands first
	switch args[0] {
	case "unresolved":
		return source.Unresolved(), nil

	case "status":
		if len(args) < 2 || args[1] == "" {
			return nil, goerr.New("invalid status format", goerr.V("args", args))
		}

		statuses := []types.AlertStatus{}
		for _, arg := range args[1:] {
			status := types.AlertStatus(arg)
			if err := status.Validate(); err != nil {
				return nil, goerr.Wrap(err, "failed to parse status")
			}
			statuses = append(statuses, status)
		}
		return source.Status(statuses...), nil

	case "between":
		if len(args) != 3 || args[1] == "" || args[2] == "" {
			return nil, goerr.New("invalid date range format", goerr.V("args", args))
		}
		begin, err := parseTime(ctx, args[1])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse begin date")
		}
		end, err := parseTime(ctx, args[2])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse end date")
		}
		return source.Span(begin, end), nil

	case "after":
		if len(args) < 2 || args[1] == "" {
			return nil, goerr.New("invalid date format", goerr.V("args", args))
		}
		begin, err := parseTime(ctx, args[1])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse date")
		}
		return source.Span(begin, clock.Now(ctx)), nil

	case "since":
		if len(args) < 2 || args[1] == "" {
			return nil, goerr.New("invalid duration format", goerr.V("args", args))
		}
		duration, err := parseDuration(args[1])
		if err != nil {
			return nil, goerr.Wrap(err, "failed to parse duration")
		}
		return source.Span(clock.Now(ctx).Add(-duration), clock.Now(ctx)), nil

	default:
		// If not a known command, try to interpret as AlertListID
		listID := types.AlertListID(args[0])
		if err := listID.Validate(); err != nil {
			return nil, goerr.New("unknown command", goerr.V("command", args[0]))
		}
		return source.AlertListID(listID), nil
	}
}

var timeOnlyFormats = []string{
	"15:04",
	"15:04:05",
	"3:04 PM",
	"3:04:05 PM",
	"3:04 AM",
	"3:04:05 AM",
	"3:04pm",
	"3:04:05pm",
	"3:04am",
	"3:04:05am",
	"15:04:05.000",
	"15:04:05.000000",
	"15:04:05.000000000",
	"15:04:05.000000000 -0700",
}

var dateIncludedFormats = []string{
	"2006-01-02",
	"2006/01/02",
	"20060102",
	"02/01/2006",
	"01/02/2006",

	"2006-01-02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05.000Z",
	"2006-01-02T15:04:05.000-07:00",
	"2006-01-02T15:04:05.000000Z",
	"2006-01-02T15:04:05.000000-07:00",
	"2006-01-02T15:04:05.000000000Z",
	"2006-01-02T15:04:05.000000000-07:00",
	"2006/01/02 15:04",
	"2006/01/02 15:04:05",
	"20060102 150405",
	"20060102T150405",
	"20060102T150405Z",
	"20060102T150405-0700",
	"20060102T150405.000Z",
	"20060102T150405.000-0700",
	"20060102T150405.000000Z",
	"20060102T150405.000000-0700",
	"20060102T150405.000000000Z",
	"20060102T150405.000000000-0700",
}

var timezonePattern = regexp.MustCompile(`^[+-]\d{2}:?\d{2}$`)

func parseTime(ctx context.Context, s string) (time.Time, error) {
	loc := clock.Timezone(ctx)

	// First, try time-only formats
	for _, layout := range timeOnlyFormats {
		t, err := time.Parse(layout, s)
		if err == nil {
			now := clock.Now(ctx)
			return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}

	// Next, try formats that include dates
	for _, layout := range dateIncludedFormats {
		// If the input string contains a timezone (Z or offset), parse in UTC
		hasTimezone := false
		if strings.Contains(s, "Z") {
			hasTimezone = true
		} else if strings.Contains(s, "+") || strings.Contains(s, "-") {
			parts := strings.Split(s, "-")
			// Check if the last part looks like a timezone offset (e.g. "+0700", "-07:00")
			if len(parts) > 0 {
				lastPart := parts[len(parts)-1]
				hasTimezone = timezonePattern.MatchString(lastPart)
			}
		}
		if hasTimezone {
			t, err := time.Parse(layout, s)
			if err == nil {
				return t, nil
			}
		} else {
			// For dates without timezone, parse in the target timezone
			t, err := time.ParseInLocation(layout, s, loc)
			if err == nil {
				return t, nil
			}
		}
	}

	return time.Time{}, goerr.New("failed to parse time with any format", goerr.V("time", s))
}

func parseDuration(s string) (time.Duration, error) {
	// Parse duration like "10m", "1h", "1d"
	var value int
	var unit string
	_, err := fmt.Sscanf(s, "%d%s", &value, &unit)
	if err != nil {
		return 0, goerr.New("invalid duration format", goerr.V("duration", s))
	}

	switch unit {
	case "m":
		return time.Duration(value) * time.Minute, nil
	case "h":
		return time.Duration(value) * time.Hour, nil
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	default:
		return 0, goerr.New("unsupported duration unit", goerr.V("unit", unit))
	}
}
