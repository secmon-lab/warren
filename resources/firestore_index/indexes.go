package main

// DefineRequiredIndexes returns all required indexes for Warren
func DefineRequiredIndexes() []IndexConfig {
	collections := []string{"alerts", "tickets", "lists"}
	memoryCollections := []string{"execution_memories", "ticket_memories"}
	var requiredIndexes []IndexConfig

	// Indexes for alerts, tickets, lists (with Embedding field)
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

	// Indexes for memory collections (with query_embedding field)
	for _, collection := range memoryCollections {
		// Single-field query_embedding index
		requiredIndexes = append(requiredIndexes, IndexConfig{
			CollectionGroup: collection,
			Fields: []IndexField{
				{
					FieldPath:    "query_embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})

		// query_embedding + created_at index
		requiredIndexes = append(requiredIndexes, IndexConfig{
			CollectionGroup: collection,
			Fields: []IndexField{
				{
					FieldPath: "created_at",
					Order:     "DESCENDING",
				},
				{
					FieldPath:    "query_embedding",
					VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
				},
			},
		})
	}

	// Index for memories subcollection (COLLECTION_GROUP query scope)
	// This is used for agent-specific memory searches: agents/{agentID}/memories/*
	requiredIndexes = append(requiredIndexes, IndexConfig{
		CollectionGroup: "memories",
		Fields: []IndexField{
			{
				FieldPath:    "query_embedding",
				VectorConfig: map[string]interface{}{"dimension": 256, "flat": map[string]interface{}{}},
			},
		},
	})

	return requiredIndexes
}
