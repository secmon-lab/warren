package errs

import (
	"errors"
)

var ErrActionUnavailable = errors.New("action is not available")
var ErrKnowledgeQuotaExceeded = errors.New("knowledge quota exceeded")
