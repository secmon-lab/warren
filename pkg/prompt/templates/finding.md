Based on the investigation so far, please provide your conclusion about this alert.

# Output

```json
{{ .schema }}
```

## `severity`: The severity of the alert.

It can be one of the following:

- `not_available`: You cannot decide the severity because of insufficient information.
- `low`: This alert is considered to have low potential impact and does not require urgent attention.
- `medium`: This alert may have impact under certain conditions and requires attention within 24 hours.
- `high`: This alert has serious impact or potential ongoing impact and requires attention within 1 hour.
- `critical`: This alert strongly indicates severe impact may already be occurring and requires immediate attention.

## `summary`: The summary of the alert.

It should be a short summary of the alert.

## `reason`: The reason for the severity.

It should be a detailed explanation of the reason for the severity.

## `recommendation`: The recommendation for the alert.

It should be a recommended next action for the alert.
