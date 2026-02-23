package types

import (
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

type RefineGroupID string

func (x RefineGroupID) String() string {
	return string(x)
}

func NewRefineGroupID() RefineGroupID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return RefineGroupID(id.String())
}

func (x RefineGroupID) Validate() error {
	if x == EmptyRefineGroupID {
		return goerr.New("empty refine group ID")
	}
	if _, err := uuid.Parse(string(x)); err != nil {
		return goerr.Wrap(err, "invalid refine group ID format", goerr.V("id", x))
	}
	return nil
}

const (
	EmptyRefineGroupID RefineGroupID = ""
)
