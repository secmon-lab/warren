package action

type ResultType string

const (
	ResultTypeString ResultType = "string"
	ResultTypeJSON   ResultType = "json"
	ResultTypeCSV    ResultType = "csv"
	ResultTypeTSV    ResultType = "tsv"
	ResultTypeText   ResultType = "text"
)

type Result struct {
	Name string
	Data map[string]any
}
