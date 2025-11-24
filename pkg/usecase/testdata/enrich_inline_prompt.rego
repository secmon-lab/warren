package enrich

# Enrich policy with inline prompt (text format)
import rego.v1

prompts contains {
    "id": "task1",
    "inline": "Analyze this alert",
    "format": "text",
}
