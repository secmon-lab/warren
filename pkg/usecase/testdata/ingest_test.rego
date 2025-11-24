package ingest.test_schema

# Test ingest policy for unit tests
import rego.v1

alert contains {
    "title": title,
    "description": description,
} if {
    input.test
    title := "Test Alert"
    description := "Test Description"
}
