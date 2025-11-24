package enrich

# Enrich policy with query task for testing LLM integration
import rego.v1

prompts contains {
    "id": "analyze",
    "inline": "Analyze this alert",
    "format": "json",
}
