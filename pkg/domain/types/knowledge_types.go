package types

import (
	"github.com/google/uuid"
	"github.com/m-mizutani/goerr/v2"
)

// KnowledgeID uniquely identifies a knowledge entry
type KnowledgeID string

func (x KnowledgeID) String() string {
	return string(x)
}

func (x KnowledgeID) Validate() error {
	if x == "" {
		return goerr.New("knowledge ID is required")
	}
	return nil
}

func NewKnowledgeID() KnowledgeID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return KnowledgeID(id.String())
}

// KnowledgeLogID uniquely identifies a knowledge log entry
type KnowledgeLogID string

func (x KnowledgeLogID) String() string {
	return string(x)
}

func (x KnowledgeLogID) Validate() error {
	if x == "" {
		return goerr.New("knowledge log ID is required")
	}
	return nil
}

func NewKnowledgeLogID() KnowledgeLogID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return KnowledgeLogID(id.String())
}

// KnowledgeTagID uniquely identifies a knowledge tag
type KnowledgeTagID string

func (x KnowledgeTagID) String() string {
	return string(x)
}

func (x KnowledgeTagID) Validate() error {
	if x == "" {
		return goerr.New("knowledge tag ID is required")
	}
	return nil
}

func NewKnowledgeTagID() KnowledgeTagID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return KnowledgeTagID(id.String())
}

// KnowledgeCategory classifies the nature of knowledge.
// Categories are fixed and cannot be added or removed by users.
type KnowledgeCategory string

const (
	// KnowledgeCategoryFact represents factual information
	// (false positive patterns, process behavior, environment info, etc.)
	KnowledgeCategoryFact KnowledgeCategory = "fact"

	// KnowledgeCategoryTechnique represents investigation methods and know-how
	// (tool usage, investigation procedures, etc.)
	KnowledgeCategoryTechnique KnowledgeCategory = "technique"
)

func (c KnowledgeCategory) String() string {
	return string(c)
}

func (c KnowledgeCategory) Validate() error {
	switch c {
	case KnowledgeCategoryFact, KnowledgeCategoryTechnique:
		return nil
	}
	return goerr.New("invalid knowledge category", goerr.V("category", c))
}
