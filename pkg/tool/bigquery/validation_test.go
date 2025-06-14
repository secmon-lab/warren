package bigquery

import (
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
)

func TestValidateConfigAgainstSchema(t *testing.T) {
	// Test data setup
	schema := bigquery.Schema{
		{Name: "user_id", Type: bigquery.StringFieldType, Required: true},
		{Name: "email", Type: bigquery.StringFieldType, Required: false},
		{Name: "age", Type: bigquery.IntegerFieldType, Required: false},
		{Name: "profile", Type: bigquery.RecordFieldType, Required: false, Schema: bigquery.Schema{
			{Name: "name", Type: bigquery.StringFieldType, Required: true},
			{Name: "preferences", Type: bigquery.RecordFieldType, Required: false, Schema: bigquery.Schema{
				{Name: "language", Type: bigquery.StringFieldType, Required: false},
				{Name: "timezone", Type: bigquery.StringFieldType, Required: false},
			}},
		}},
		{Name: "tags", Type: bigquery.StringFieldType, Required: false, Repeated: true},
	}

	tableMetadata := &bigquery.TableMetadata{
		Schema: schema,
	}

	t.Run("valid config with simple fields", func(t *testing.T) {
		config := &Config{
			Columns: []ColumnConfig{
				{Name: "user_id", Type: "STRING"},
				{Name: "email", Type: "STRING"},
				{Name: "age", Type: "INTEGER"},
			},
		}
		result := ValidateConfigAgainstSchema(config, tableMetadata)
		gt.Value(t, result.Valid).Equal(true)
		gt.Value(t, len(result.Issues)).Equal(0)
	})

	t.Run("invalid field name", func(t *testing.T) {
		config := &Config{
			Columns: []ColumnConfig{
				{Name: "user_id", Type: "STRING"},
				{Name: "invalid_field", Type: "STRING"}, // This field doesn't exist
			},
		}
		result := ValidateConfigAgainstSchema(config, tableMetadata)
		gt.Value(t, result.Valid).Equal(false)
		gt.Value(t, len(result.Issues)).Equal(1)
		gt.Value(t, result.Issues[0].Type).Equal("field_not_found")
	})

	t.Run("invalid field type", func(t *testing.T) {
		config := &Config{
			Columns: []ColumnConfig{
				{Name: "user_id", Type: "INTEGER"}, // Should be STRING
				{Name: "age", Type: "STRING"},      // Should be INTEGER
			},
		}
		result := ValidateConfigAgainstSchema(config, tableMetadata)
		gt.Value(t, result.Valid).Equal(false)
		gt.Value(t, len(result.Issues)).Equal(2)
	})

	t.Run("valid nested record fields", func(t *testing.T) {
		config := &Config{
			Columns: []ColumnConfig{
				{
					Name: "profile",
					Type: "RECORD",
					Fields: []ColumnConfig{
						{Name: "name", Type: "STRING"},
						{
							Name: "preferences",
							Type: "RECORD",
							Fields: []ColumnConfig{
								{Name: "language", Type: "STRING"},
								{Name: "timezone", Type: "STRING"},
							},
						},
					},
				},
			},
		}
		result := ValidateConfigAgainstSchema(config, tableMetadata)
		gt.Value(t, result.Valid).Equal(true)
		gt.Value(t, len(result.Issues)).Equal(0)
	})

	t.Run("invalid nested field", func(t *testing.T) {
		config := &Config{
			Columns: []ColumnConfig{
				{
					Name: "profile",
					Type: "RECORD",
					Fields: []ColumnConfig{
						{Name: "name", Type: "STRING"},
						{Name: "invalid_nested_field", Type: "STRING"}, // This doesn't exist
					},
				},
			},
		}
		result := ValidateConfigAgainstSchema(config, tableMetadata)
		gt.Value(t, result.Valid).Equal(false)
		gt.Value(t, len(result.Issues)).Equal(1)
		gt.Value(t, result.Issues[0].Type).Equal("field_not_found")
	})
}

func TestBuildSchemaFieldMap(t *testing.T) {
	schema := bigquery.Schema{
		{Name: "user_id", Type: bigquery.StringFieldType},
		{Name: "profile", Type: bigquery.RecordFieldType, Schema: bigquery.Schema{
			{Name: "name", Type: bigquery.StringFieldType},
			{Name: "settings", Type: bigquery.RecordFieldType, Schema: bigquery.Schema{
				{Name: "theme", Type: bigquery.StringFieldType},
				{Name: "notifications", Type: bigquery.BooleanFieldType},
			}},
		}},
		{Name: "tags", Type: bigquery.StringFieldType, Repeated: true},
	}

	fieldMap := BuildSchemaFieldMap(schema, "")

	// Test direct fields
	gt.Value(t, fieldMap["user_id"]).NotNil()
	gt.Value(t, fieldMap["user_id"].Name).Equal("user_id")
	gt.Value(t, fieldMap["user_id"].Type).Equal(bigquery.StringFieldType)

	gt.Value(t, fieldMap["profile"]).NotNil()
	gt.Value(t, fieldMap["profile"].Name).Equal("profile")
	gt.Value(t, fieldMap["profile"].Type).Equal(bigquery.RecordFieldType)

	gt.Value(t, fieldMap["tags"]).NotNil()
	gt.Value(t, fieldMap["tags"].Name).Equal("tags")
	gt.Value(t, fieldMap["tags"].Repeated).Equal(true)

	// Test nested fields
	gt.Value(t, fieldMap["profile.name"]).NotNil()
	gt.Value(t, fieldMap["profile.name"].Name).Equal("name")
	gt.Value(t, fieldMap["profile.name"].Type).Equal(bigquery.StringFieldType)

	gt.Value(t, fieldMap["profile.settings"]).NotNil()
	gt.Value(t, fieldMap["profile.settings"].Name).Equal("settings")
	gt.Value(t, fieldMap["profile.settings"].Type).Equal(bigquery.RecordFieldType)

	// Test deeply nested fields
	gt.Value(t, fieldMap["profile.settings.theme"]).NotNil()
	gt.Value(t, fieldMap["profile.settings.theme"].Name).Equal("theme")
	gt.Value(t, fieldMap["profile.settings.theme"].Type).Equal(bigquery.StringFieldType)

	gt.Value(t, fieldMap["profile.settings.notifications"]).NotNil()
	gt.Value(t, fieldMap["profile.settings.notifications"].Name).Equal("notifications")
	gt.Value(t, fieldMap["profile.settings.notifications"].Type).Equal(bigquery.BooleanFieldType)

	// Test field count
	expectedFields := []string{
		"user_id",
		"profile",
		"profile.name",
		"profile.settings",
		"profile.settings.theme",
		"profile.settings.notifications",
		"tags",
	}
	gt.Value(t, len(fieldMap)).Equal(len(expectedFields))

	// Test non-existent fields
	gt.Value(t, fieldMap["nonexistent"]).Nil()
	gt.Value(t, fieldMap["profile.nonexistent"]).Nil()
}

func TestValidateColumnConfig(t *testing.T) {
	// Setup schema field map
	schema := bigquery.Schema{
		{Name: "user_id", Type: bigquery.StringFieldType},
		{Name: "age", Type: bigquery.IntegerFieldType},
		{Name: "profile", Type: bigquery.RecordFieldType, Schema: bigquery.Schema{
			{Name: "name", Type: bigquery.StringFieldType},
		}},
	}
	fieldMap := BuildSchemaFieldMap(schema, "")

	t.Run("valid field", func(t *testing.T) {
		result := &SchemaValidationResult{Valid: true, Issues: []SchemaValidationIssue{}}
		column := ColumnConfig{Name: "user_id", Type: "STRING"}
		validateColumnConfig(column, fieldMap, result, "")
		gt.Value(t, len(result.Issues)).Equal(0)
	})

	t.Run("field not found", func(t *testing.T) {
		result := &SchemaValidationResult{Valid: true, Issues: []SchemaValidationIssue{}}
		column := ColumnConfig{Name: "invalid_field", Type: "STRING"}
		validateColumnConfig(column, fieldMap, result, "")
		gt.Value(t, len(result.Issues)).Equal(1)
		gt.Value(t, result.Issues[0].Type).Equal("field_not_found")
	})

	t.Run("type mismatch", func(t *testing.T) {
		result := &SchemaValidationResult{Valid: true, Issues: []SchemaValidationIssue{}}
		column := ColumnConfig{Name: "age", Type: "STRING"} // Should be INTEGER
		validateColumnConfig(column, fieldMap, result, "")
		gt.Value(t, len(result.Issues)).Equal(1)
		gt.Value(t, result.Issues[0].Type).Equal("type_mismatch")
	})

	t.Run("valid nested field", func(t *testing.T) {
		result := &SchemaValidationResult{Valid: true, Issues: []SchemaValidationIssue{}}
		column := ColumnConfig{
			Name: "profile",
			Type: "RECORD",
			Fields: []ColumnConfig{
				{Name: "name", Type: "STRING"},
			},
		}
		validateColumnConfig(column, fieldMap, result, "")
		gt.Value(t, len(result.Issues)).Equal(0)
	})

	t.Run("invalid nested field", func(t *testing.T) {
		result := &SchemaValidationResult{Valid: true, Issues: []SchemaValidationIssue{}}
		column := ColumnConfig{
			Name: "profile",
			Type: "RECORD",
			Fields: []ColumnConfig{
				{Name: "invalid_field", Type: "STRING"},
			},
		}
		validateColumnConfig(column, fieldMap, result, "")
		gt.Value(t, len(result.Issues)).Equal(1)
		gt.Value(t, result.Issues[0].Type).Equal("field_not_found")
	})
}

func TestFormatValidationReport(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := SchemaValidationResult{
			Valid:  true,
			Issues: []SchemaValidationIssue{},
		}
		report := formatValidationReport(result)
		gt.Value(t, strings.Contains(report, "✅ Schema validation passed")).Equal(true)
	})

	t.Run("invalid result with multiple issue types", func(t *testing.T) {
		result := SchemaValidationResult{
			Valid: false,
			Issues: []SchemaValidationIssue{
				{
					Type:      "field_not_found",
					FieldPath: "invalid_field",
					Message:   "Field 'invalid_field' does not exist",
				},
				{
					Type:         "type_mismatch",
					FieldPath:    "age",
					ExpectedType: "STRING",
					ActualType:   "INTEGER",
					Message:      "Field 'age' type mismatch",
				},
			},
		}
		report := formatValidationReport(result)

		gt.Value(t, strings.Contains(report, "❌ SCHEMA VALIDATION FAILED")).Equal(true)
		gt.Value(t, strings.Contains(report, "Found 2 issue(s)")).Equal(true)
		gt.Value(t, strings.Contains(report, "🔍 FIELDS NOT FOUND")).Equal(true)
		gt.Value(t, strings.Contains(report, "🔄 DATA TYPE MISMATCHES")).Equal(true)
		gt.Value(t, strings.Contains(report, "REQUIRED ACTIONS")).Equal(true)
		gt.Value(t, strings.Contains(report, "invalid_field")).Equal(true)
		gt.Value(t, strings.Contains(report, "age")).Equal(true)
	})
}
