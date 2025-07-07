package errs

import (
	"errors"

	"github.com/m-mizutani/goerr/v2"
)

var ErrActionUnavailable = errors.New("action is not available")

var (
	TagInvalidLLMResponse = goerr.NewTag("invalid_llm_response")
	TagInvalidRequest     = goerr.NewTag("invalid_request")
	TagTestFailed         = goerr.NewTag("test_failed")
)
