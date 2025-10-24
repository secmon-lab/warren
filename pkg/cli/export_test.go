package cli

import "github.com/secmon-lab/warren/pkg/usecase"

// ReadAlertDataForTest exposes readAlertData for testing
func ReadAlertDataForTest(inputFile string) (any, error) {
	return readAlertData(inputFile)
}

// DisplayPipelineResultForTest exposes displayPipelineResult for testing
func DisplayPipelineResultForTest(results []*usecase.AlertPipelineResult) error {
	return displayPipelineResult(results)
}
