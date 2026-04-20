package ingest.test_schema

# Test ingest policy that emits an alert without a title so that
# FillMetadata runs during the pipeline. Used for tag-inference tests.
import rego.v1

alerts contains {} if {
    input.test
}
