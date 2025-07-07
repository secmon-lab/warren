package bigquery

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"cloud.google.com/go/bigquery"
)

// securityFieldCategory represents different types of security-relevant fields
type securityFieldCategory string

const (
	categoryIdentity securityFieldCategory = "identity"
	categoryNetwork  securityFieldCategory = "network"
	categoryTemporal securityFieldCategory = "temporal"
	categoryAuth     securityFieldCategory = "authentication"
	categoryResource securityFieldCategory = "resource"
	categoryGeo      securityFieldCategory = "geographic"
	categoryEvent    securityFieldCategory = "event"
	categoryThreat   securityFieldCategory = "threat"
	categoryHash     securityFieldCategory = "hash"
	categoryMetadata securityFieldCategory = "metadata"
)

// securityField represents a field with its security relevance
type securityField struct {
	Name        string
	Type        bigquery.FieldType
	Description string
	Category    securityFieldCategory
	Priority    int // 1-10, higher is more important
	Examples    []string
}

// IoC patterns for automatic detection
var (
	ipv4Pattern     = regexp.MustCompile(`^(ip|addr|address|src|dst|source|dest|remote|client|server)`)
	emailPattern    = regexp.MustCompile(`^(email|mail|user_email|username|user_name)`)
	userPattern     = regexp.MustCompile(`^(user|uid|userid|user_id|username|account|subject)`)
	timePattern     = regexp.MustCompile(`^(time|timestamp|date|created|updated|modified|occurred|event_time)`)
	hashPattern     = regexp.MustCompile(`^(hash|md5|sha1|sha256|sha512|checksum|digest)`)
	urlPattern      = regexp.MustCompile(`^(url|uri|link|href|path|endpoint)`)
	domainPattern   = regexp.MustCompile(`^(domain|host|hostname|fqdn|server_name|site)`)
	portPattern     = regexp.MustCompile(`^(port|src_port|dst_port|source_port|dest_port)`)
	protocolPattern = regexp.MustCompile(`^(protocol|proto|scheme)`)
	actionPattern   = regexp.MustCompile(`^(action|event|activity|operation|command|method)`)
	statusPattern   = regexp.MustCompile(`^(status|result|outcome|success|failed|error|response_code)`)
	threatPattern   = regexp.MustCompile(`^(threat|malware|virus|attack|risk|severity|alert|incident)`)
	geoPattern      = regexp.MustCompile(`^(country|region|city|location|geo|latitude|longitude)`)
	sessionPattern  = regexp.MustCompile(`^(session|token|ticket|cookie|auth|login|logout)`)
	devicePattern   = regexp.MustCompile(`^(device|client|agent|browser|os|platform|fingerprint)`)
)

// AnalyzeSecurityFields analyzes BigQuery schema fields and categorizes them by security relevance
func AnalyzeSecurityFields(schema bigquery.Schema) []securityField {
	var securityFields []securityField

	for _, field := range schema {
		if secField := analyzeField(field); secField != nil {
			securityFields = append(securityFields, *secField)
		}
	}

	return securityFields
}

// analyzeField analyzes a single field for security relevance
func analyzeField(field *bigquery.FieldSchema) *securityField {
	fieldName := strings.ToLower(field.Name)
	fieldDesc := strings.ToLower(field.Description)
	combined := fieldName + " " + fieldDesc

	// Exclude system/ETL related fields that are not security-relevant
	if isSystemField(fieldName, fieldDesc) {
		return nil
	}

	// Check various patterns to categorize the field
	if isIdentityField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryIdentity,
			Priority:    9,
			Examples:    generateExamples(field.Type, categoryIdentity),
		}
	}

	if isNetworkField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryNetwork,
			Priority:    8,
			Examples:    generateExamples(field.Type, categoryNetwork),
		}
	}

	if isTemporalField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryTemporal,
			Priority:    7,
			Examples:    generateExamples(field.Type, categoryTemporal),
		}
	}

	if isAuthField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryAuth,
			Priority:    8,
			Examples:    generateExamples(field.Type, categoryAuth),
		}
	}

	if isThreatField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryThreat,
			Priority:    9,
			Examples:    generateExamples(field.Type, categoryThreat),
		}
	}

	if isHashField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryHash,
			Priority:    6,
			Examples:    generateExamples(field.Type, categoryHash),
		}
	}

	if isGeoField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryGeo,
			Priority:    5,
			Examples:    generateExamples(field.Type, categoryGeo),
		}
	}

	if isEventField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryEvent,
			Priority:    7,
			Examples:    generateExamples(field.Type, categoryEvent),
		}
	}

	if isResourceField(fieldName, fieldDesc) {
		return &securityField{
			Name:        field.Name,
			Type:        field.Type,
			Description: field.Description,
			Category:    categoryResource,
			Priority:    6,
			Examples:    generateExamples(field.Type, categoryResource),
		}
	}

	// Check if it contains any security-related keywords
	securityKeywords := []string{
		"security", "attack", "threat", "vulnerability", "exploit", "breach",
		"suspicious", "anomaly", "alert", "incident", "forensic", "investigation",
	}

	for _, keyword := range securityKeywords {
		if strings.Contains(combined, keyword) {
			return &securityField{
				Name:        field.Name,
				Type:        field.Type,
				Description: field.Description,
				Category:    categoryMetadata,
				Priority:    4,
				Examples:    generateExamples(field.Type, categoryMetadata),
			}
		}
	}

	// Not a security-relevant field
	return nil
}

// Field categorization helper functions
func isIdentityField(name, desc string) bool {
	return userPattern.MatchString(name) || emailPattern.MatchString(name) ||
		strings.Contains(name, "subject") || strings.Contains(desc, "user") ||
		strings.Contains(desc, "identity")
}

func isNetworkField(name, desc string) bool {
	return ipv4Pattern.MatchString(name) || urlPattern.MatchString(name) ||
		domainPattern.MatchString(name) || portPattern.MatchString(name) ||
		protocolPattern.MatchString(name) || strings.Contains(desc, "network") ||
		strings.Contains(desc, "ip") || strings.Contains(desc, "domain")
}

func isTemporalField(name, desc string) bool {
	return timePattern.MatchString(name) || strings.Contains(desc, "time") ||
		strings.Contains(desc, "date") || strings.Contains(desc, "timestamp")
}

func isAuthField(name, desc string) bool {
	return sessionPattern.MatchString(name) || statusPattern.MatchString(name) ||
		devicePattern.MatchString(name) || strings.Contains(name, "auth") ||
		strings.Contains(desc, "authentication") || strings.Contains(desc, "login") ||
		strings.Contains(desc, "session") || strings.Contains(desc, "device")
}

func isThreatField(name, desc string) bool {
	return threatPattern.MatchString(name) || strings.Contains(desc, "threat") ||
		strings.Contains(desc, "malware") || strings.Contains(desc, "attack") ||
		strings.Contains(desc, "risk") || strings.Contains(desc, "severity")
}

func isHashField(name, desc string) bool {
	return hashPattern.MatchString(name) || strings.Contains(desc, "hash") ||
		strings.Contains(desc, "checksum") || strings.Contains(desc, "digest")
}

func isGeoField(name, desc string) bool {
	return geoPattern.MatchString(name) || strings.Contains(desc, "country") ||
		strings.Contains(desc, "location") || strings.Contains(desc, "geographic")
}

func isEventField(name, desc string) bool {
	return actionPattern.MatchString(name) || strings.Contains(desc, "event") ||
		strings.Contains(desc, "action") || strings.Contains(desc, "activity")
}

func isResourceField(name, desc string) bool {
	return strings.Contains(name, "file") || strings.Contains(name, "path") ||
		strings.Contains(name, "resource") || strings.Contains(desc, "file") ||
		strings.Contains(desc, "resource") || strings.Contains(desc, "path")
}

func isSystemField(name, desc string) bool {
	// First check if it's a security-relevant field that should not be excluded
	if isKnownsecurityField(name) {
		return false
	}

	// System metadata patterns
	systemPatterns := []string{
		"internal_", "system_", "etl_", "pipeline_", "processing_",
		"metadata_", "schema_", "partition_", "ingestion_", "batch_",
		"_version", "_key", "_flags", "_status",
	}

	for _, pattern := range systemPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}

	// Check for generic system IDs but exclude security-related IDs
	if strings.HasSuffix(name, "_id") && !isSecurityID(name) {
		return true
	}

	// System descriptions
	systemDescriptions := []string{
		"internal", "system", "etl", "pipeline", "processing",
		"metadata", "schema", "partition", "ingestion", "batch",
		"primary key", "database", "record id", "version",
	}

	for _, sysDesc := range systemDescriptions {
		if strings.Contains(desc, sysDesc) {
			return true
		}
	}

	// Specific exclusions for common non-security temporal fields
	if strings.Contains(name, "etl") && strings.Contains(desc, "timestamp") {
		return true
	}
	if strings.Contains(name, "created_by") && strings.Contains(desc, "system") {
		return true
	}

	return false
}

func isKnownsecurityField(name string) bool {
	securityFields := []string{
		"user_id", "account_id", "session_id", "device_id", "user", "email", "username",
		"source_ip", "dest_ip", "src_ip", "dst_ip", "domain", "url", "host",
		"threat", "malware", "attack", "risk", "severity", "hash", "checksum",
		"country", "region", "city", "location", "event", "action", "activity",
		"auth", "login", "logout", "token", "credential",
	}

	for _, secField := range securityFields {
		if strings.Contains(name, secField) {
			return true
		}
	}
	return false
}

func isSecurityID(name string) bool {
	securityIDs := []string{
		"user_id", "account_id", "session_id", "device_id", "request_id",
		"correlation_id", "trace_id", "alert_id", "incident_id",
	}

	for _, secID := range securityIDs {
		if name == secID || strings.HasPrefix(name, secID) {
			return true
		}
	}
	return false
}

// generateExamples generates example values for different field categories
func generateExamples(fieldType bigquery.FieldType, category securityFieldCategory) []string {
	switch category {
	case categoryIdentity:
		switch fieldType {
		case bigquery.StringFieldType:
			return []string{"user123", "john.doe", "admin", "service_account"}
		case bigquery.IntegerFieldType:
			return []string{"12345", "67890"}
		default:
			return []string{"user_identifier"}
		}

	case categoryNetwork:
		if strings.Contains(strings.ToLower(string(fieldType)), "string") {
			return []string{"192.168.1.100", "203.0.113.50", "example.com", "https://api.example.com"}
		}
		return []string{"443", "80", "22", "3389"}

	case categoryTemporal:
		return []string{"2024-01-01T12:00:00Z", "1704110400", "2024-01-01 12:00:00"}

	case categoryAuth:
		return []string{"login", "logout", "authenticate", "success", "failure", "session_abc123"}

	case categoryThreat:
		return []string{"malware", "phishing", "high", "critical", "suspicious_activity"}

	case categoryHash:
		return []string{"d41d8cd98f00b204e9800998ecf8427e", "e3b0c44298fc1c149afbf4c8996fb924"}

	case categoryGeo:
		return []string{"US", "United States", "California", "San Francisco", "37.7749"}

	case categoryEvent:
		return []string{"CREATE", "DELETE", "UPDATE", "ACCESS", "MODIFY"}

	case categoryResource:
		return []string{"/etc/passwd", "C:\\Windows\\System32", "document.pdf", "config.json"}

	default:
		return []string{"example_value"}
	}
}

// EnhanceSecurityAnalysis adds security-specific analysis tools
func (x *generateConfigTool) EnhanceSecurityAnalysis(ctx context.Context) error {
	// This could be expanded to add more security-specific tools to the LLM agent
	return nil
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// generateSecurityPrompt creates a security-focused prompt for LLM analysis
func generateSecurityPrompt(fields []securityField) string {
	var prompt strings.Builder

	prompt.WriteString("## Security Field Analysis\n\n")
	prompt.WriteString("The following fields have been identified as security-relevant:\n\n")

	categories := make(map[securityFieldCategory][]securityField)
	for _, field := range fields {
		categories[field.Category] = append(categories[field.Category], field)
	}

	for category, categoryFields := range categories {
		prompt.WriteString(fmt.Sprintf("### %s Fields\n", capitalizeFirst(string(category))))
		for _, field := range categoryFields {
			prompt.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", field.Name, field.Type, field.Description))
			if len(field.Examples) > 0 {
				prompt.WriteString(fmt.Sprintf("  - Examples: %s\n", strings.Join(field.Examples, ", ")))
			}
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("## Analysis Focus\n\n")
	prompt.WriteString("When analyzing this data for security purposes, focus on:\n")
	prompt.WriteString("1. **Identity correlation**: Link user activities across different events\n")
	prompt.WriteString("2. **Network analysis**: Identify suspicious IP addresses, domains, and network patterns\n")
	prompt.WriteString("3. **Temporal analysis**: Detect time-based anomalies and patterns\n")
	prompt.WriteString("4. **Threat detection**: Look for indicators of compromise and malicious activity\n")
	prompt.WriteString("5. **Geographic analysis**: Identify unusual locations and geo-based anomalies\n")

	return prompt.String()
}
