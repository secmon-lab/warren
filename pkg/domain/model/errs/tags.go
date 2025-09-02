package errs

import "github.com/m-mizutani/goerr/v2"

var (
	// Client errors (4xx)
	TagNotFound     = goerr.NewTag("not_found")    // 404
	TagValidation   = goerr.NewTag("validation")   // 400
	TagUnauthorized = goerr.NewTag("unauthorized") // 401
	TagForbidden    = goerr.NewTag("forbidden")    // 403
	TagConflict     = goerr.NewTag("conflict")     // 409
	TagRateLimit    = goerr.NewTag("rate_limit")   // 429

	// Server errors (5xx)
	TagInternal = goerr.NewTag("internal") // 500
	TagExternal = goerr.NewTag("external") // 502/503
	TagTimeout  = goerr.NewTag("timeout")  // 504
	TagDatabase = goerr.NewTag("database") // 500 (specific to DB errors)

	// Business logic errors
	TagInvalidState      = goerr.NewTag("invalid_state")
	TagDuplicateResource = goerr.NewTag("duplicate_resource")
	TagQuotaExceeded     = goerr.NewTag("quota_exceeded")
	TagResourceLocked    = goerr.NewTag("resource_locked")

	// External service errors
	TagSlackError  = goerr.NewTag("slack_error")
	TagGitHubError = goerr.NewTag("github_error")
	TagLLMError    = goerr.NewTag("llm_error")

	// Keep existing tags for compatibility
	TagInvalidLLMResponse = goerr.NewTag("invalid_llm_response")
	TagInvalidRequest     = goerr.NewTag("invalid_request")
	TagTestFailed         = goerr.NewTag("test_failed")
)
