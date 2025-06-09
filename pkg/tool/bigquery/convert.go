package bigquery

import (
	"fmt"

	"cloud.google.com/go/bigquery"
)

// convertBigQueryValue converts BigQuery values to JSON-safe types for Vertex AI
func convertBigQueryValue(value bigquery.Value) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string, int, int64, float64, bool:
		return v
	case []bigquery.Value:
		// Convert arrays
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = convertBigQueryValue(item)
		}
		return result
	case map[string]bigquery.Value:
		// Convert structs/records
		result := make(map[string]any)
		for key, val := range v {
			result[key] = convertBigQueryValue(val)
		}
		return result
	default:
		// Convert unknown types to string representation
		return fmt.Sprintf("%v", v)
	}
}
