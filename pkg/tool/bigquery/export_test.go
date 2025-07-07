package bigquery

var FlattenSchema = flattenSchema

type SchemaField = schemaField

// Test exports for mock types
var NewMockBigQueryClient = newMockBigQueryClient

type MockBigQueryClient = mockBigQueryClient
type MockBigQueryClientFactory = mockBigQueryClientFactory

// Test exports for helper functions
var GenerateConfigWithFactory = generateConfigWithFactoryInternal
var GenerateConfigSchema = generateConfigSchema

// Test exports for security analysis
var GenerateSecurityPrompt = generateSecurityPrompt

type SecurityField = securityField
type SecurityFieldCategory = securityFieldCategory

// Test exports for security field categories
const (
	CategoryIdentity = categoryIdentity
	CategoryNetwork  = categoryNetwork
	CategoryTemporal = categoryTemporal
	CategoryAuth     = categoryAuth
	CategoryResource = categoryResource
	CategoryGeo      = categoryGeo
	CategoryEvent    = categoryEvent
	CategoryThreat   = categoryThreat
	CategoryHash     = categoryHash
	CategoryMetadata = categoryMetadata
)

// Export functions for testing
var (
	ValidateColumnConfig   = validateColumnConfig
	FormatValidationReport = formatValidationReport
	LoadBulkConfig         = loadBulkConfig
	ParseTableID           = parseTableID
)

// Export types for testing
type GenerateConfigConfig = generateConfigConfig
type TestBulkConfigEntry = BulkConfigEntry
type TestBulkConfig = BulkConfig
