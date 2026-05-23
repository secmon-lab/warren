package ingest.test

import rego.v1

alerts contains alert if {
	input.event_type == "test"
	alert := {
		"title": input.title,
		"description": "sample policy",
	}
}
