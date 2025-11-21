package usecase

import "errors"

// errAgentAuthPolicyNotDefined is returned when agent authorization policy is not defined
var errAgentAuthPolicyNotDefined = errors.New("agent authorization policy not defined")

// errAgentAuthDenied is returned when agent authorization is denied
var errAgentAuthDenied = errors.New("agent request not authorized")

// errAgentAuthFailed is returned when agent authorization fails
var errAgentAuthFailed = errors.New("agent authorization failed")
