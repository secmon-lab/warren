package types

import "github.com/m-mizutani/goerr/v2"

type GenAIContentFormat string

const (
	GenAIContentFormatText GenAIContentFormat = "text"
	GenAIContentFormatJSON GenAIContentFormat = "json"
)

// Validate checks if the GenAIContentFormat has a valid value
func (x GenAIContentFormat) Validate() error {
	switch x {
	case GenAIContentFormatText, GenAIContentFormatJSON:
		return nil
	default:
		return goerr.New("invalid GenAI content format", goerr.V("content_format", string(x)), goerr.V("valid_formats", []string{string(GenAIContentFormatText), string(GenAIContentFormatJSON)}))
	}
}

type PublishType string

const (
	PublishTypeDiscard PublishType = "discard" // Discard the alert
	PublishTypeNotice  PublishType = "notice"  // Send as notice
	PublishTypeAlert   PublishType = "alert"   // Send as full alert (default)
)
