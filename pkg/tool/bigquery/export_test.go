package bigquery

var FlattenSchema = flattenSchema

type SchemaField = schemaField

// Test exports for mock types
var NewMockBigQueryClient = newMockBigQueryClient

type MockBigQueryClient = mockBigQueryClient
type MockBigQueryClientFactory = mockBigQueryClientFactory

// Test exports for helper functions
var GenerateConfigWithFactory = generateConfigWithFactoryInternal

// Test exports for security analysis
var AnalyzeSecurityFields = analyzesecurityFields
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
