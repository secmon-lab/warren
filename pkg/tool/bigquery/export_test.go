package bigquery

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
