package types

import "github.com/m-mizutani/goerr/v2"

type GenAIContentType string

const (
	GenAIContentTypeText GenAIContentType = "text"
	GenAIContentTypeJSON GenAIContentType = "json"
)

// Validate checks if the GenAIContentType has a valid value
func (x GenAIContentType) Validate() error {
	switch x {
	case GenAIContentTypeText, GenAIContentTypeJSON:
		return nil
	default:
		return goerr.New("invalid GenAI content type", goerr.V("content_type", string(x)), goerr.V("valid_types", []string{string(GenAIContentTypeText), string(GenAIContentTypeJSON)}))
	}
}

type PublishType string

const (
	PublishTypeDiscard PublishType = "discard" // Discard the alert
	PublishTypeNotice  PublishType = "notice"  // Send as notice
	PublishTypeAlert   PublishType = "alert"   // Send as full alert (default)
)
