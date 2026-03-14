package types

import "github.com/google/uuid"

type DiagnosisID string

func (x DiagnosisID) String() string {
	return string(x)
}

func NewDiagnosisID() DiagnosisID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return DiagnosisID(id.String())
}

const EmptyDiagnosisID DiagnosisID = ""
