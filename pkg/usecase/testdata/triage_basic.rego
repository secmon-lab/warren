package triage

# Basic triage policy for testing
import rego.v1

default publish := "notice"

title := "Updated Title"
publish := "alert"
