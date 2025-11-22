package slack

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
)

func TestParseSlackTimestamp(t *testing.T) {
	t.Run("parses timestamp with sub-second precision", func(t *testing.T) {
		// Slack timestamp: 1234567890.123456
		ts := parseSlackTimestamp("1234567890.123456")

		// Expected: 2009-02-13 23:31:30.123456 UTC
		expected := time.Unix(1234567890, 123456000)

		gt.V(t, ts.Unix()).Equal(expected.Unix())
		gt.V(t, ts.Nanosecond()).Equal(expected.Nanosecond())
	})

	t.Run("parses timestamp without sub-second precision", func(t *testing.T) {
		// Slack timestamp: 1234567890
		ts := parseSlackTimestamp("1234567890")

		expected := time.Unix(1234567890, 0)

		gt.V(t, ts.Unix()).Equal(expected.Unix())
		gt.V(t, ts.Nanosecond()).Equal(0)
	})

	t.Run("returns zero time for invalid timestamp", func(t *testing.T) {
		ts := parseSlackTimestamp("invalid")

		gt.True(t, ts.IsZero())
	})

	t.Run("returns zero time for empty timestamp", func(t *testing.T) {
		ts := parseSlackTimestamp("")

		gt.True(t, ts.IsZero())
	})

	t.Run("preserves microsecond precision", func(t *testing.T) {
		// Test various microsecond values
		testCases := []struct {
			input    string
			expected int64 // nanoseconds
		}{
			{"1234567890.000001", 1000},      // 1 microsecond
			{"1234567890.000100", 100000},    // 100 microseconds
			{"1234567890.100000", 100000000}, // 0.1 second
			{"1234567890.999999", 999999000}, // 999999 microseconds
		}

		for _, tc := range testCases {
			ts := parseSlackTimestamp(tc.input)
			gt.V(t, ts.Nanosecond()).Equal(int(tc.expected))
		}
	})
}
