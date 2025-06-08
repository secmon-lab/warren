package bigquery

import (
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
)

func TestAnalyzeSecurityFields(t *testing.T) {
	// Create a realistic security table schema
	schema := bigquery.Schema{
		// Identity fields
		{Name: "user_id", Type: bigquery.StringFieldType, Description: "Unique user identifier"},
		{Name: "username", Type: bigquery.StringFieldType, Description: "Human readable username"},
		{Name: "email", Type: bigquery.StringFieldType, Description: "User email address"},

		// Network fields
		{Name: "source_ip", Type: bigquery.StringFieldType, Description: "Source IP address"},
		{Name: "destination_ip", Type: bigquery.StringFieldType, Description: "Destination IP address"},
		{Name: "domain", Type: bigquery.StringFieldType, Description: "Domain name accessed"},
		{Name: "url", Type: bigquery.StringFieldType, Description: "Full URL accessed"},
		{Name: "src_port", Type: bigquery.IntegerFieldType, Description: "Source port number"},
		{Name: "dst_port", Type: bigquery.IntegerFieldType, Description: "Destination port number"},
		{Name: "protocol", Type: bigquery.StringFieldType, Description: "Network protocol used"},

		// Temporal fields
		{Name: "timestamp", Type: bigquery.TimestampFieldType, Description: "Event timestamp"},
		{Name: "created_time", Type: bigquery.TimestampFieldType, Description: "Record creation time"},

		// Authentication fields
		{Name: "auth_result", Type: bigquery.StringFieldType, Description: "Authentication result status"},
		{Name: "session_id", Type: bigquery.StringFieldType, Description: "User session identifier"},
		{Name: "login_method", Type: bigquery.StringFieldType, Description: "Authentication method used"},

		// Threat fields
		{Name: "threat_type", Type: bigquery.StringFieldType, Description: "Type of threat detected"},
		{Name: "malware_family", Type: bigquery.StringFieldType, Description: "Malware family name"},
		{Name: "risk_score", Type: bigquery.IntegerFieldType, Description: "Calculated risk score"},
		{Name: "severity", Type: bigquery.StringFieldType, Description: "Alert severity level"},

		// Hash fields
		{Name: "file_hash_md5", Type: bigquery.StringFieldType, Description: "MD5 hash of file"},
		{Name: "sha256_digest", Type: bigquery.StringFieldType, Description: "SHA256 hash value"},

		// Geographic fields
		{Name: "country_code", Type: bigquery.StringFieldType, Description: "Country code from geolocation"},
		{Name: "city", Type: bigquery.StringFieldType, Description: "City from geolocation"},
		{Name: "latitude", Type: bigquery.FloatFieldType, Description: "Geographic latitude"},

		// Event fields
		{Name: "event_type", Type: bigquery.StringFieldType, Description: "Type of security event"},
		{Name: "action", Type: bigquery.StringFieldType, Description: "Action performed"},

		// Resource fields
		{Name: "file_path", Type: bigquery.StringFieldType, Description: "File system path"},
		{Name: "resource_name", Type: bigquery.StringFieldType, Description: "Name of accessed resource"},

		// Non-security fields (should be filtered out)
		{Name: "internal_id", Type: bigquery.IntegerFieldType, Description: "Internal database ID"},
		{Name: "processing_metadata", Type: bigquery.StringFieldType, Description: "System processing information"},
		{Name: "schema_version", Type: bigquery.StringFieldType, Description: "Data schema version"},
	}

	securityFields := AnalyzeSecurityFields(schema)

	// Should identify security-relevant fields and filter out non-security ones
	gt.True(t, len(securityFields) > 20) // Should find most security fields
	gt.True(t, len(securityFields) < 30) // Should exclude non-security fields

	// Create a map for easier lookup
	fieldMap := make(map[string]securityField)
	for _, field := range securityFields {
		fieldMap[field.Name] = field
	}

	// Test specific field categorizations
	userField, exists := fieldMap["user_id"]
	gt.True(t, exists)
	gt.Equal(t, userField.Category, categoryIdentity)
	gt.Equal(t, userField.Priority, 9)

	ipField, exists := fieldMap["source_ip"]
	gt.True(t, exists)
	gt.Equal(t, ipField.Category, categoryNetwork)
	gt.Equal(t, ipField.Priority, 8)

	timeField, exists := fieldMap["timestamp"]
	gt.True(t, exists)
	gt.Equal(t, timeField.Category, categoryTemporal)
	gt.Equal(t, timeField.Priority, 7)

	threatField, exists := fieldMap["threat_type"]
	gt.True(t, exists)
	gt.Equal(t, threatField.Category, categoryThreat)
	gt.Equal(t, threatField.Priority, 9)

	hashField, exists := fieldMap["file_hash_md5"]
	gt.True(t, exists)
	gt.Equal(t, hashField.Category, categoryHash)
	gt.Equal(t, hashField.Priority, 6)

	geoField, exists := fieldMap["country_code"]
	gt.True(t, exists)
	gt.Equal(t, geoField.Category, categoryGeo)
	gt.Equal(t, geoField.Priority, 5)

	// Verify non-security fields are excluded
	_, exists = fieldMap["internal_id"]
	gt.False(t, exists)
	_, exists = fieldMap["processing_metadata"]
	gt.False(t, exists)
	_, exists = fieldMap["schema_version"]
	gt.False(t, exists)
}

func TestSecurityFieldCategories(t *testing.T) {
	testCases := []struct {
		name        string
		fieldType   bigquery.FieldType
		description string
		expected    securityFieldCategory
		priority    int
	}{
		// Identity fields
		{"user_email", bigquery.StringFieldType, "User email address", categoryIdentity, 9},
		{"account_id", bigquery.StringFieldType, "Account identifier", categoryIdentity, 9},
		{"subject_name", bigquery.StringFieldType, "Subject name", categoryIdentity, 9},

		// Network fields
		{"client_ip", bigquery.StringFieldType, "Client IP address", categoryNetwork, 8},
		{"hostname", bigquery.StringFieldType, "Host name", categoryNetwork, 8},
		{"server_port", bigquery.IntegerFieldType, "Server port number", categoryNetwork, 8},

		// Temporal fields
		{"event_time", bigquery.TimestampFieldType, "Event occurrence time", categoryTemporal, 7},
		{"updated_date", bigquery.DateFieldType, "Last updated date", categoryTemporal, 7},

		// Authentication fields
		{"login_status", bigquery.StringFieldType, "Login result status", categoryAuth, 8},
		{"auth_token", bigquery.StringFieldType, "Authentication token", categoryAuth, 8},

		// Threat fields
		{"attack_vector", bigquery.StringFieldType, "Attack vector used", categoryThreat, 9},
		{"virus_name", bigquery.StringFieldType, "Virus name detected", categoryThreat, 9},

		// Hash fields
		{"checksum", bigquery.StringFieldType, "File checksum", categoryHash, 6},
		{"sha1_hash", bigquery.StringFieldType, "SHA1 hash value", categoryHash, 6},

		// Geographic fields
		{"region", bigquery.StringFieldType, "Geographic region", categoryGeo, 5},
		{"location_data", bigquery.StringFieldType, "Location information", categoryGeo, 5},

		// Event fields
		{"operation", bigquery.StringFieldType, "Operation performed", categoryEvent, 7},
		{"activity_type", bigquery.StringFieldType, "Type of activity", categoryEvent, 7},

		// Resource fields
		{"filename", bigquery.StringFieldType, "File name", categoryResource, 6},
		{"resource_path", bigquery.StringFieldType, "Resource path", categoryResource, 6},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field := &bigquery.FieldSchema{
				Name:        tc.name,
				Type:        tc.fieldType,
				Description: tc.description,
			}

			secField := analyzeField(field)
			gt.NotNil(t, secField)
			gt.Equal(t, secField.Category, tc.expected)
			gt.Equal(t, secField.Priority, tc.priority)
			gt.True(t, len(secField.Examples) > 0)
		})
	}
}

func TestNonsecurityFields(t *testing.T) {
	nonsecurityFields := []struct {
		name        string
		fieldType   bigquery.FieldType
		description string
	}{
		{"id", bigquery.IntegerFieldType, "Primary key"},
		{"created_by_system", bigquery.StringFieldType, "System that created record"},
		{"data_version", bigquery.IntegerFieldType, "Data version number"},
		{"internal_flags", bigquery.StringFieldType, "Internal processing flags"},
		{"etl_timestamp", bigquery.TimestampFieldType, "ETL processing time"},
		{"partition_key", bigquery.StringFieldType, "Table partition key"},
		{"row_count", bigquery.IntegerFieldType, "Number of rows processed"},
	}

	for _, field := range nonsecurityFields {
		t.Run(field.name, func(t *testing.T) {
			bqField := &bigquery.FieldSchema{
				Name:        field.name,
				Type:        field.fieldType,
				Description: field.description,
			}

			secField := analyzeField(bqField)
			// These should not be identified as security fields
			gt.Nil(t, secField)
		})
	}
}

func TestGenerateSecurityPrompt(t *testing.T) {
	securityFields := []securityField{
		{
			Name:        "user_id",
			Type:        bigquery.StringFieldType,
			Description: "User identifier",
			Category:    categoryIdentity,
			Priority:    9,
			Examples:    []string{"user123", "admin"},
		},
		{
			Name:        "source_ip",
			Type:        bigquery.StringFieldType,
			Description: "Source IP address",
			Category:    categoryNetwork,
			Priority:    8,
			Examples:    []string{"192.168.1.100", "203.0.113.50"},
		},
		{
			Name:        "threat_level",
			Type:        bigquery.StringFieldType,
			Description: "Threat severity level",
			Category:    categoryThreat,
			Priority:    9,
			Examples:    []string{"high", "critical"},
		},
	}

	prompt := GenerateSecurityPrompt(securityFields)

	// Verify the prompt contains expected sections
	gt.S(t, prompt).Contains("## Security Field Analysis")
	gt.S(t, prompt).Contains("### Identity Fields")
	gt.S(t, prompt).Contains("### Network Fields")
	gt.S(t, prompt).Contains("### Threat Fields")
	gt.S(t, prompt).Contains("## Analysis Focus")

	// Verify field details are included
	gt.S(t, prompt).Contains("user_id")
	gt.S(t, prompt).Contains("source_ip")
	gt.S(t, prompt).Contains("threat_level")
	gt.S(t, prompt).Contains("Examples: user123, admin")
	gt.S(t, prompt).Contains("Examples: 192.168.1.100, 203.0.113.50")

	// Verify analysis guidance is included
	gt.S(t, prompt).Contains("Identity correlation")
	gt.S(t, prompt).Contains("Network analysis")
	gt.S(t, prompt).Contains("Temporal analysis")
	gt.S(t, prompt).Contains("Threat detection")
	gt.S(t, prompt).Contains("Geographic analysis")
}

func TestGenerateExamples(t *testing.T) {
	testCases := []struct {
		fieldType bigquery.FieldType
		category  securityFieldCategory
		expected  []string
	}{
		{bigquery.StringFieldType, categoryIdentity, []string{"user123", "john.doe", "admin", "service_account"}},
		{bigquery.IntegerFieldType, categoryIdentity, []string{"12345", "67890"}},
		{bigquery.StringFieldType, categoryNetwork, []string{"192.168.1.100", "203.0.113.50", "example.com", "https://api.example.com"}},
		{bigquery.IntegerFieldType, categoryNetwork, []string{"443", "80", "22", "3389"}},
		{bigquery.TimestampFieldType, categoryTemporal, []string{"2024-01-01T12:00:00Z", "1704110400", "2024-01-01 12:00:00"}},
		{bigquery.StringFieldType, categoryAuth, []string{"login", "logout", "authenticate", "success", "failure", "session_abc123"}},
		{bigquery.StringFieldType, categoryThreat, []string{"malware", "phishing", "high", "critical", "suspicious_activity"}},
		{bigquery.StringFieldType, categoryHash, []string{"d41d8cd98f00b204e9800998ecf8427e", "e3b0c44298fc1c149afbf4c8996fb924"}},
		{bigquery.StringFieldType, categoryGeo, []string{"US", "United States", "California", "San Francisco", "37.7749"}},
		{bigquery.StringFieldType, categoryEvent, []string{"CREATE", "DELETE", "UPDATE", "ACCESS", "MODIFY"}},
		{bigquery.StringFieldType, categoryResource, []string{"/etc/passwd", "C:\\Windows\\System32", "document.pdf", "config.json"}},
	}

	for _, tc := range testCases {
		t.Run(string(tc.category)+"_"+string(tc.fieldType), func(t *testing.T) {
			examples := generateExamples(tc.fieldType, tc.category)
			gt.A(t, examples).Equal(tc.expected)
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "A"},
		{"hello", "Hello"},
		{"HELLO", "HELLO"},
		{"identity", "Identity"},
		{"network", "Network"},
		{"temporal", "Temporal"},
		{"authentication", "Authentication"},
		{"threat", "Threat"},
		{"hash", "Hash"},
		{"geographic", "Geographic"},
		{"event", "Event"},
		{"resource", "Resource"},
		{"metadata", "Metadata"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := capitalizeFirst(tc.input)
			gt.Equal(t, result, tc.expected)
		})
	}
}
