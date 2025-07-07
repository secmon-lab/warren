package slack

import (
	"strings"
)

/*
// parseArgs parses a string into arguments, handling various types of quotes
func parseArgs(input string) []string {
	var result []string
	var current []rune
	var inQuotes bool
	var quoteChar rune

	// Unicode code points for quotes
	const (
		leftDoubleQuote  = '\u201c' // "
		rightDoubleQuote = '\u201d' // "
		leftSingleQuote  = '\u2018' // '
		rightSingleQuote = '\u2019' // '
	)

	// isMatchingQuote checks if two quote characters form a matching pair
	isMatchingQuote := func(open, close rune) bool {
		return open == close || // Same quotes
			(open == leftDoubleQuote && close == rightDoubleQuote) || // Unicode double quotes
			(open == leftSingleQuote && close == rightSingleQuote) // Unicode single quotes
	}

	for i := 0; i < len(input); {
		char, size := utf8.DecodeRuneInString(input[i:])
		i += size

		switch char {
		case '\\':
			if i < len(input) {
				nextChar, size := utf8.DecodeRuneInString(input[i:])
				if nextChar == '\\' || nextChar == '"' || nextChar == '\'' ||
					nextChar == leftDoubleQuote || nextChar == rightDoubleQuote ||
					nextChar == leftSingleQuote || nextChar == rightSingleQuote ||
					nextChar == '`' {
					current = append(current, nextChar)
					i += size
				} else {
					current = append(current, char)
				}
			} else {
				current = append(current, char)
			}
		case '"', '\'', leftDoubleQuote, rightDoubleQuote, leftSingleQuote, rightSingleQuote, '`':
			if inQuotes {
				if isMatchingQuote(quoteChar, char) {
					inQuotes = false
					if len(current) > 0 {
						result = append(result, string(current))
						current = nil
					}
				} else {
					current = append(current, char)
				}
			} else {
				inQuotes = true
				quoteChar = char
				if len(current) > 0 {
					result = append(result, string(current))
					current = nil
				}
			}
		case ' ':
			if inQuotes {
				current = append(current, char)
			} else if len(current) > 0 {
				result = append(result, string(current))
				current = nil
			}
		default:
			current = append(current, char)
		}
	}

	if len(current) > 0 {
		result = append(result, string(current))
	}

	return result
}
*/

func ParseMention(input string) []Mention {
	mentions := make([]Mention, 0)
	var current *Mention
	var messageBuilder strings.Builder

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		if i+1 < len(runes) && string(runes[i:i+2]) == "<@" {
			// Found start of mention
			if current != nil {
				current.Message = strings.TrimSpace(messageBuilder.String())
				mentions = append(mentions, *current)
				messageBuilder.Reset()
			}

			// Extract user ID
			start := i + 2
			for i = start; i < len(runes) && runes[i] != '>'; i++ {
			}
			if i < len(runes) {
				current = &Mention{
					UserID: string(runes[start:i]),
				}
			}
		} else if current != nil {
			messageBuilder.WriteRune(runes[i])
		}
	}

	if current != nil {
		current.Message = strings.TrimSpace(messageBuilder.String())
		mentions = append(mentions, *current)
	}

	return mentions
}
