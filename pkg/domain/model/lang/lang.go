package lang

import "github.com/m-mizutani/goerr/v2"

type Lang string

const (
	English  Lang = "en"
	Japanese Lang = "ja"

	Default Lang = English
)

func (l Lang) Name() string {
	// Return the language code directly for any language
	return string(l)
}

func (l Lang) Validate() error {
	// Accept any non-empty language code
	if string(l) == "" {
		return goerr.New("language cannot be empty")
	}
	return nil
}
