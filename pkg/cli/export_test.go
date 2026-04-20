package cli

import (
	"io"

	"github.com/m-mizutani/fireconf"
	"github.com/secmon-lab/warren/pkg/usecase"
)

// ReadAlertDataForTest exposes readAlertData for testing
func ReadAlertDataForTest(inputFile string) (any, error) {
	return readAlertData(inputFile)
}

// DisplayPipelineResultForTest exposes displayPipelineResult for testing
func DisplayPipelineResultForTest(results []*usecase.AlertPipelineResult) error {
	return displayPipelineResult(results)
}

// DefineFirestoreIndexes exposes defineFirestoreIndexes for testing
func DefineFirestoreIndexes() *fireconf.Config {
	return defineFirestoreIndexes()
}

// FormatIndexFieldsForTest exposes formatIndexFields for testing
func FormatIndexFieldsForTest(fields []fireconf.IndexField) string {
	return formatIndexFields(fields)
}

// PrintMigrationPlanForTest exposes printMigrationPlan for testing
func PrintMigrationPlanForTest(w io.Writer, projectID, databaseID string, dryRun bool, want *fireconf.Config, diff *fireconf.DiffResult) {
	printMigrationPlan(w, projectID, databaseID, dryRun, want, diff)
}
