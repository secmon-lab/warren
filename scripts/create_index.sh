#!/bin/bash

# 引数の数をチェック
if [ $# -ne 2 ]; then
    echo "Usage: $0 <project_id> <database_id>"
    exit 1
fi

PROJECT_ID=$1
DATABASE_ID=$2

gcloud firestore indexes composite create \
    --project="${PROJECT_ID}" \
    --database="${DATABASE_ID}" \
    --collection-group=alerts \
    --query-scope=COLLECTION \
    --field-config=vector-config='{"dimension":"256","flat": "{}"}',field-path=Embedding

gcloud firestore indexes composite create \
    --project="${PROJECT_ID}" \
    --database="${DATABASE_ID}" \
    --collection-group=tickets \
    --query-scope=COLLECTION \
    --field-config=vector-config='{"dimension":"256","flat": "{}"}',field-path=Embedding
