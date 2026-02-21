package bigquery

import (
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
)

// SchemaValidationResult holds the result of schema validation
type SchemaValidationResult struct {
	Valid  bool
	Issues []SchemaValidationIssue
}

// SchemaValidationIssue represents a single validation issue
type SchemaValidationIssue struct {
	Type         string // "field_not_found", "type_mismatch"
	FieldPath    string
	ExpectedType string
	ActualType   string
	Message      string
}

// ValidateConfigAgainstSchema validates the generated config against the actual BigQuery table schema
func ValidateConfigAgainstSchema(config *Config, tableMetadata *bigquery.TableMetadata) SchemaValidationResult {
	result := SchemaValidationResult{
		Valid:  true,
		Issues: []SchemaValidationIssue{},
	}

	// Create a map of actual schema fields for quick lookup
	actualFields := BuildSchemaFieldMap(tableMetadata.Schema, "")

	// Validate each column in the config
	for _, column := range config.Columns {
		validateColumnConfig(column, actualFields, &result, "")
	}

	if len(result.Issues) > 0 {
		result.Valid = false
	}

	return result
}

// BuildSchemaFieldMap creates a flat map of all fields (including nested) from BigQuery schema
func BuildSchemaFieldMap(schema bigquery.Schema, prefix string) map[string]*bigquery.FieldSchema {
	fieldMap := make(map[string]*bigquery.FieldSchema)

	for _, field := range schema {
		fieldName := field.Name
		if prefix != "" {
			fieldName = prefix + "." + field.Name
		}

		fieldMap[fieldName] = field

		// Handle nested RECORD fields
		if field.Type == bigquery.RecordFieldType && len(field.Schema) > 0 {
			nestedFields := BuildSchemaFieldMap(field.Schema, fieldName)
			for k, v := range nestedFields {
				fieldMap[k] = v
			}
		}
	}

	return fieldMap
}

// validateColumnConfig validates a single column config against the actual schema
func validateColumnConfig(column ColumnConfig, actualFields map[string]*bigquery.FieldSchema, result *SchemaValidationResult, prefix string) {
	fieldPath := column.Name
	if prefix != "" {
		fieldPath = prefix + "." + column.Name
	}

	actualField, exists := actualFields[fieldPath]
	if !exists {
		result.Issues = append(result.Issues, SchemaValidationIssue{
			Type:      "field_not_found",
			FieldPath: fieldPath,
			Message:   fmt.Sprintf("Field '%s' does not exist in the actual table schema", fieldPath),
		})
		return
	}

	// Validate data type
	expectedType := strings.ToUpper(column.Type)
	actualType := string(actualField.Type)
	if expectedType != actualType {
		result.Issues = append(result.Issues, SchemaValidationIssue{
			Type:         "type_mismatch",
			FieldPath:    fieldPath,
			ExpectedType: expectedType,
			ActualType:   actualType,
			Message:      fmt.Sprintf("Field '%s' type mismatch: config has '%s', actual schema has '%s'", fieldPath, expectedType, actualType),
		})
	}

	// Validate nested fields for RECORD types
	if column.Type == "RECORD" && len(column.Fields) > 0 {
		for _, nestedField := range column.Fields {
			validateColumnConfig(nestedField, actualFields, result, fieldPath)
		}
	}
}

// formatValidationReport creates a formatted report of validation issues
func formatValidationReport(result SchemaValidationResult) string {
	if result.Valid {
		return "✅ Schema validation passed"
	}

	var report strings.Builder
	report.WriteString("❌ Schema validation failed\n\n")
	fmt.Fprintf(&report, "Found %d issue(s):\n\n", len(result.Issues))

	// Group issues by type
	fieldNotFound := []SchemaValidationIssue{}
	typeMismatch := []SchemaValidationIssue{}

	for _, issue := range result.Issues {
		switch issue.Type {
		case "field_not_found":
			fieldNotFound = append(fieldNotFound, issue)
		case "type_mismatch":
			typeMismatch = append(typeMismatch, issue)
		}
	}

	if len(fieldNotFound) > 0 {
		report.WriteString("Fields not found:\n")
		for i, issue := range fieldNotFound {
			fmt.Fprintf(&report, "  %d. %s\n", i+1, issue.Message)
		}
		report.WriteString("\n")
	}

	if len(typeMismatch) > 0 {
		report.WriteString("Type mismatches:\n")
		for i, issue := range typeMismatch {
			fmt.Fprintf(&report, "  %d. %s\n", i+1, issue.Message)
		}
		report.WriteString("\n")
	}

	report.WriteString("Please fix these issues and regenerate the configuration.")

	return report.String()
}
