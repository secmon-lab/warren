package chat

import (
	"github.com/m-mizutani/goerr/v2"
)

var (
	// ErrSessionAborted is returned when a session is aborted by user request
	ErrSessionAborted = goerr.New("session aborted by user")
)
