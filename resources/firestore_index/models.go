package main

// IndexField represents a field in a Firestore index
type IndexField struct {
	FieldPath    string                 `json:"fieldPath"`
	Order        string                 `json:"order,omitempty"`
	VectorConfig map[string]interface{} `json:"vectorConfig,omitempty"`
}

// Index represents a Firestore composite index
type Index struct {
	Name            string       `json:"name"`
	CollectionGroup string       `json:"collectionGroup"`
	Fields          []IndexField `json:"fields"`
	QueryScope      string       `json:"queryScope"`
}

// IndexConfig represents the configuration for creating an index
type IndexConfig struct {
	CollectionGroup string
	Fields          []IndexField
}
