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
	Message string
	Type    ResultType
	Rows    []string
}

type Exit struct {
	Conclusion string
}
