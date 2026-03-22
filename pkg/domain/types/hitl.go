package types

import "github.com/google/uuid"

// HITLRequestID represents a unique HITL request identifier
type HITLRequestID string

// NewHITLRequestID generates a new HITL request ID
func NewHITLRequestID() HITLRequestID {
	return HITLRequestID(uuid.New().String())
}

func (x HITLRequestID) String() string {
	return string(x)
}
