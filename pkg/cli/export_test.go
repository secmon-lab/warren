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

// MigrationJobNamesForTest returns the registered migration job names
// in declaration order so tests can verify the registry.
func MigrationJobNamesForTest() []string {
	names := make([]string, 0, len(migrationJobs))
	for _, j := range migrationJobs {
		names = append(names, j.Name)
	}
	return names
}

// MigrationJobDescriptionForTest returns the description of a
// registered job, or the empty string when no job has the given name.
func MigrationJobDescriptionForTest(name string) string {
	if j, ok := findMigrationJob(name); ok {
		return j.Description
	}
	return ""
}

// ChatSessionRedesignBundleStepsForTest returns the names of the jobs
// the v0.16.0 bundle runs, in invocation order.
func ChatSessionRedesignBundleStepsForTest() []string {
	out := make([]string, 0, len(chatSessionRedesignBundleSteps))
	for _, s := range chatSessionRedesignBundleSteps {
		out = append(out, s.Name)
	}
	return out
}
