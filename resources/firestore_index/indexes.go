package main

// DefineRequiredIndexes returns all required indexes for Warren
func DefineRequiredIndexes() []IndexConfig {
	collections := []string{"alerts", "tickets", "lists"}
	var requiredIndexes []IndexConfig

	for _, collection := range collections {
		// Single-field Embedding index
		requiredIndexes = append(requiredIndexes, IndexConfig{
			CollectionGroup: collection,
			Fields: []IndexField{
				{
					FieldPath:    "Embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})

		// Embedding + CreatedAt index
		requiredIndexes = append(requiredIndexes, IndexConfig{
			CollectionGroup: collection,
			Fields: []IndexField{
				{
					FieldPath: "CreatedAt",
					Order:     "DESCENDING",
				},
				{
					FieldPath:    "Embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})

		// Status + CreatedAt index only for 'tickets'
		if collection == "tickets" {
			requiredIndexes = append(requiredIndexes, IndexConfig{
				CollectionGroup: collection,
				Fields: []IndexField{
					{
						FieldPath: "Status",
						Order:     "ASCENDING",
					},
					{
						FieldPath: "CreatedAt",
						Order:     "DESCENDING",
					},
				},
			})
		}
	}

	return requiredIndexes
}
