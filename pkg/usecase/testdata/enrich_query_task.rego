package enrich

# Enrich policy with query task for testing LLM integration
import rego.v1

query contains {
    "id": "analyze",
    "inline": "Analyze this alert",
    "format": "json",
}
