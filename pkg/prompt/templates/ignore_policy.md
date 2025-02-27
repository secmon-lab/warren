Please update the given Rego policy to ignore the data of the provided alerts.

# Rules

## Policy

- The `schema` field of the alert indicates which package is evaluated. For example, if the `schema` field is `guardduty`, the Rego policy of `package alert.guardduty` will be evaluated.
- The `data` field of the alert is stored in `input`.

A Rego policy detects an alert when an object is stored in the set named `alert`. Here is a specific example:

```rego
package alert.guardduty

alert contains {} # Detected as an alert
```

Conversely, if nothing is stored in the set named `alert`, no alert is detected. In the example below, `ignore` becomes true when `input.Findings.Severity < 7`, and no alert is detected.

```rego
package alert.guardduty

alert contains {} if { not ignore } # Alert is detected if ignore is false

ignore if {
    input.Findings.Severity < 7 # ignore becomes true when Findings.Severity is less than 7
}
```

The `input` contains the structured data from the `data` field of the `alert`. Use this data to set appropriate ignore conditions.

Example of the `input` variable in above policy:

```json
{
    "Findings": {
        "Severity": 5
    }
}
```

The `input` field is stored in the `data` field of the `alert`.

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
{{ .output }}
```

- The `title` field is the title. It should be a short description of the policy changes.
- The `description` field is the description. It should be a detailed description of the policy changes.
- The `policy` field is the updated policy. It should be a map of the file name and the updated policy.

`title` and `description` should be in {{ .lang }}. They would be used as the title and description of the pull request.

Example of the correct output:

```json
{
    "title": "Ignore alerts with severity less than 7",
    "description": "Ignore alerts with severity less than 7. This is a test.",
    "policy": {
        "some_dir/some_file.rego": "package alert.guardduty\n\nalert contains {}\n\nignore if { input.SeverityScore < 7 }",
        "some_dir/some_other_file.rego": "package alert.guardduty\n\nignore if { input.Name == \"example-alert\" }"
    }
}
```
