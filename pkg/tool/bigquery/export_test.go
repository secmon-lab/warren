package bigquery

import extbq "github.com/gollem-dev/tools/bigquery"

// Opts exposes the accumulated external options for testing that flag Action
// callbacks append the expected option.
func (x *Action) Opts() []extbq.Option {
	return x.opts
}

var FlattenSchema = flattenSchema

type SchemaField = schemaField

// Test exports for mock types
var NewMockBigQueryClient = newMockBigQueryClient

type MockBigQueryClient = mockBigQueryClient
type MockBigQueryClientFactory = mockBigQueryClientFactory

// Test exports for helper functions
var GenerateConfigWithFactory = generateConfigWithFactory
var GenerateConfigSchema = generateConfigSchema

// Export functions for testing
var (
	ValidateColumnConfig   = validateColumnConfig
	FormatValidationReport = formatValidationReport
	LoadBulkConfig         = loadBulkConfig
	ParseTableID           = parseTableID
	GenerateOutputPath     = generateOutputPath
)

// Export types for testing
type GenerateConfigInput = generateConfigInput
type TestBulkConfigEntry = bulkConfigEntry
type TestBulkConfig = bulkConfig
