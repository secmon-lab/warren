#!/bin/bash

# Check number of arguments
if [ $# -ne 2 ]; then
    echo "Usage: $0 <project_id> <database_id>"
    exit 1
fi

PROJECT_ID=$1
DATABASE_ID=$2

# Function to check if index exists and create if not
create_index_if_not_exists() {
    local collection_group=$1
    local field_path=$2

    # Get existing indexes
    local existing_indexes
    existing_indexes=$(gcloud firestore indexes composite list \
        --project="${PROJECT_ID}" \
        --database="${DATABASE_ID}" \
        --format="json")

    # Check if index exists (check both collectionGroup and field-path)
    local matching_index
    matching_index=$(echo "${existing_indexes}" | jq -e ".[] | select(.name | contains(\"collectionGroups/${collection_group}\"))")

    if [ $? -eq 0 ] && [ -n "${matching_index}" ]; then
        echo "Index for ${collection_group} with field ${field_path} already exists."
    else
        echo "Creating index for ${collection_group}..."
        gcloud firestore indexes composite create \
            --project="${PROJECT_ID}" \
            --database="${DATABASE_ID}" \
            --collection-group="${collection_group}" \
            --query-scope=COLLECTION \
            --field-config=vector-config='{"dimension":"256","flat": "{}"}',field-path="${field_path}"
    fi
}

# Create indexes for each collection
create_index_if_not_exists "alerts" "Embedding"
create_index_if_not_exists "tickets" "Embedding"
create_index_if_not_exists "lists" "Embedding"
