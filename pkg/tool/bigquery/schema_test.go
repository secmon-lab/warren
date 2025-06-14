package bigquery_test

import (
	_ "embed"
	"encoding/json"
	"os"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"

	bq "github.com/secmon-lab/warren/pkg/tool/bigquery"
)

//go:embed testdata/schema.json
var schemaJSON []byte

func TestFlattenSchemaWithData(t *testing.T) {
	var schema bigquery.Schema
	err := json.Unmarshal(schemaJSON, &schema)
	gt.Nil(t, err)

	gt.Array(t, schema).Length(16)
	gt.Equal(t, schema[0].Name, "logName")
	gt.Equal(t, schema[0].Type, bigquery.StringFieldType)

	flattened := bq.FlattenSchema(schema, []string{})
	gt.Array(t, flattened).Length(974)
	gt.Equal(t, flattened[0].Name, "logName")
	gt.Equal(t, flattened[0].Type, "STRING")
	gt.Equal(t, flattened[2].Name, "resource.type")
	gt.Equal(t, flattened[2].Type, "STRING")

	// To generate test data for pkg/service/llm/summary_test.go
	if v, ok := os.LookupEnv("TEST_SCHEMA_DUMP_PATH"); ok {
		fd, err := os.Create(v)
		gt.NoError(t, err).Required()
		gt.NoError(t, json.NewEncoder(fd).Encode(flattened))
	}
}

func TestFlattenSchema(t *testing.T) {
	schema := bigquery.Schema{
		&bigquery.FieldSchema{
			Name:        "field1",
			Type:        bigquery.StringFieldType,
			Repeated:    false,
			Required:    true,
			Description: "First field description",
		},
		&bigquery.FieldSchema{
			Name:        "nested",
			Type:        bigquery.RecordFieldType,
			Repeated:    false,
			Required:    false,
			Description: "Nested record description",
			Schema: bigquery.Schema{
				&bigquery.FieldSchema{
					Name:        "field2",
					Type:        bigquery.IntegerFieldType,
					Repeated:    false,
					Required:    true,
					Description: "Second field description",
				},
				&bigquery.FieldSchema{
					Name:        "deep",
					Type:        bigquery.RecordFieldType,
					Repeated:    true,
					Required:    false,
					Description: "Deep nested record description",
					Schema: bigquery.Schema{
						&bigquery.FieldSchema{
							Name:        "field3",
							Type:        bigquery.BooleanFieldType,
							Repeated:    false,
							Required:    false,
							Description: "Third field description",
						},
					},
				},
			},
		},
	}

	expected := []bq.SchemaField{
		{
			Name:        "field1",
			Type:        "STRING",
			Repeated:    false,
			Description: "First field description",
		},
		{
			Name:        "nested",
			Type:        "RECORD",
			Repeated:    false,
			Description: "Nested record description",
		},
		{
			Name:        "nested.field2",
			Type:        "INTEGER",
			Repeated:    false,
			Description: "Second field description",
		},
		{
			Name:        "nested.deep",
			Type:        "RECORD",
			Repeated:    true,
			Description: "Deep nested record description",
		},
		{
			Name:        "nested.deep.field3",
			Type:        "BOOLEAN",
			Repeated:    false,
			Description: "Third field description",
		},
	}

	result := bq.FlattenSchema(schema, []string{})
	gt.Array(t, result).Equal(expected)
}
