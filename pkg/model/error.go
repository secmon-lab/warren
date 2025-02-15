package model

import (
	"errors"

	"github.com/m-mizutani/goerr/v2"
)

var ErrActionUnavailable = errors.New("action is not available")

var (
	ErrTagInvalidLLMResponse = goerr.NewTag("invalid_llm_response")
	ErrTagInvalidRequest     = goerr.NewTag("invalid_request")
)
