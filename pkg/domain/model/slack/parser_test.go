package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/warren/pkg/domain/model/slack"
)

func TestParseMention(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []slack.Mention
	}{
		{
			name:  "single mention",
			input: "<@U123>hello",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "hello",
				},
			},
		},
		{
			name:  "multiple mentions",
			input: "<@U123>hello<@U456>goodbye",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "hello",
				},
				{
					UserID:  "U456",
					Message: "goodbye",
				},
			},
		},
		{
			name:  "mention without message",
			input: "<@U123>",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "",
				},
			},
		},
		{
			name:     "no mentions",
			input:    "hello world",
			expected: nil,
		},
		{
			name:  "mention with spaces",
			input: "<@U123> hello world ",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "hello world",
				},
			},
		},
		{
			name:  "mention with Japanese text",
			input: "<@U123>こんにちは",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "こんにちは",
				},
			},
		},
		{
			name:  "multiple mentions with mixed content",
			input: "start <@U123>hello world<@U456>goodbye everyone<@U789>最後",
			expected: []slack.Mention{
				{
					UserID:  "U123",
					Message: "hello world",
				},
				{
					UserID:  "U456",
					Message: "goodbye everyone",
				},
				{
					UserID:  "U789",
					Message: "最後",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := slack.ParseMention(c.input)
			gt.Equal(t, len(c.expected), len(got))

			for i := range got {
				gt.Equal(t, c.expected[i].UserID, got[i].UserID)
				gt.Equal(t, c.expected[i].Message, got[i].Message)
			}
		})
	}
}

/*
func TestParseArgs(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected []string
	}{
		"simple": {
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		"with quotes": {
			input:    `hello "world with spaces"`,
			expected: []string{"hello", "world with spaces"},
		},
		"with single quotes": {
			input:    `hello 'world with spaces'`,
			expected: []string{"hello", "world with spaces"},
		},
		"with escaped quotes": {
			input:    `hello \"world\" test`,
			expected: []string{"hello", `"world"`, "test"},
		},
		"with Japanese quotes": {
			input:    `hello "world" test`,
			expected: []string{"hello", "world", "test"},
		},
		"with multiple spaces": {
			input:    "hello   world",
			expected: []string{"hello", "world"},
		},
		"empty": {
			input:    "",
			expected: nil,
		},
		"with backslash": {
			input:    `hello\\world`,
			expected: []string{"hello\\world"},
		},
		"mixed quotes": {
			input:    `"hello" 'world' "test"`,
			expected: []string{"hello", "world", "test"},
		},
		"with backticks": {
			input:    "hello \\`world\\` test",
			expected: []string{"hello", "`world`", "test"},
		},
		"with utf-8 quotes": {
			input:    `ok “this is” “hello world”`,
			expected: []string{"ok", "this is", "hello world"},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := slack.ParseArgs(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d args, got %d", len(tc.expected), len(result))
				return
			}
			for i := range result {
				if result[i] != tc.expected[i] {
					t.Errorf("arg %d: expected %q, got %q", i, tc.expected[i], result[i])
				}
			}
		})
	}
}
func TestParseMention(t *testing.T) {
	testCases := map[string]struct {
		input    string
		expected []slack.Mention
	}{
		"single mention": {
			input: "<@U123> hello world",
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{"hello", "world"},
				},
			},
		},
		"multiple mentions": {
			input: "<@U123> hello <@U456> world",
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{"hello"},
				},
				{
					UserID: "U456",
					Args:   []string{"world"},
				},
			},
		},
		"mention with quoted args": {
			input: `<@U123> "hello world" test`,
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{"hello world", "test"},
				},
			},
		},
		"mention without args": {
			input: "<@U123>",
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{},
				},
			},
		},
		"empty string": {
			input:    "",
			expected: nil,
		},
		"text without mention": {
			input:    "hello world",
			expected: nil,
		},
		"mention with Japanese text": {
			input: "<@U123> こんにちは 世界",
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{"こんにちは", "世界"},
				},
			},
		},
		"mention with mixed quotes": {
			input: `<@U123> "hello world" 'test case' "quoted"`,
			expected: []slack.Mention{
				{
					UserID: "U123",
					Args:   []string{"hello world", "test case", "quoted"},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			result := slack.ParseMention(tc.input)
			if len(result) != len(tc.expected) {
				t.Errorf("expected %d mentions, got %d", len(tc.expected), len(result))
				return
			}
			for i := range result {
				if result[i].UserID != tc.expected[i].UserID {
					t.Errorf("mention %d: expected UserID %q, got %q", i, tc.expected[i].UserID, result[i].UserID)
				}
				if len(result[i].Args) != len(tc.expected[i].Args) {
					t.Errorf("mention %d: expected %d args, got %d", i, len(tc.expected[i].Args), len(result[i].Args))
					continue
				}
				for j := range result[i].Args {
					if result[i].Args[j] != tc.expected[i].Args[j] {
						t.Errorf("mention %d arg %d: expected %q, got %q", i, j, tc.expected[i].Args[j], result[i].Args[j])
					}
				}
			}
		})
	}
}
*/
