Please update the given Rego policy to ignore the data of the provided alerts.

# Rules

## Policy

- The `schema` field of the alert indicates which package is evaluated. For example, if the `schema` field is `aws.guardduty`, the Rego policy of `package alert.aws.guardduty` will be evaluated.
- The `data` field of the alert is stored in `input`.

A Rego policy detects an alert when an object is stored in the set named `alert`. Here is a specific example:

```rego
package alert.aws.guardduty

alert contains {} # Detected as an alert
```

Conversely, if nothing is stored in the set named `alert`, no alert is detected. In the example below, `ignore` becomes true when `input.Findings.Severity < 7`, and no alert is detected.

```rego
package alert.aws.guardduty

alert contains {} if { not ignore } # Alert is detected if ignore is false

ignore if {
    input.Findings.Severity < 7 # ignore becomes true when Findings.Severity is less than 7
}
```

The `input` contains the structured data from the `data` field of the `alert`. Use this data to set appropriate ignore conditions.

## Restrictions

- If multiple alerts are given, update the policy to ignore all of them.
- Set conditions that are appropriately abstracted, rather than using unique values of the alerts (such as IP addresses or alert IDs) as conditions.
- Add new ignore conditions while retaining the current functionality of the policy.
- Primarily focus on adding new ignore conditions, but feel free to refactor or restructure the existing policy if necessary.

{{ if .note }}
## Additional Notes

Please refer to the following instructions as well.
{{ .note }}
{{ end }}

# Input

## Policy

The current policy is as follows. The provided data is stored in a map with the file name as the key.

```rego
{{ .policy }}
```

## Alerts

The alerts that should be ignored are as follows.

{{range .alerts}}
------
```json
{{ . }}
```
{{end}}

# Output

The updated policy should be output according to the following schema. The schema is represented in JSON schema. Even if you do not update the existing policy, please store the data under the same key.

```json
{{ .schema }}
```