package chat

import (
	"errors"

	"github.com/m-mizutani/goerr/v2"
)

var (
	// ErrSessionAborted is returned when a session is aborted by user request
	ErrSessionAborted = goerr.New("session aborted by user")

	// ErrAgentAuthPolicyNotDefined is returned when agent authorization policy is not defined
	ErrAgentAuthPolicyNotDefined = errors.New("agent authorization policy not defined")

	// ErrAgentAuthDenied is returned when agent authorization is denied
	ErrAgentAuthDenied = errors.New("agent request not authorized")
)
