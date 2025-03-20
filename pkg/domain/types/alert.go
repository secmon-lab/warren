package types

import "github.com/google/uuid"

type AlertID string

func (x AlertID) String() string {
	return string(x)
}

func NewAlertID() AlertID {
	return AlertID(uuid.New().String())
}

type AlertListID string

func (x AlertListID) String() string {
	return string(x)
}

func NewAlertListID() AlertListID {
	return AlertListID(uuid.New().String())
}
