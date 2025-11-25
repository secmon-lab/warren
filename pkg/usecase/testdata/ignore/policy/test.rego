package ingest.guardduty

alerts contains {} if {
    not ignore
}

ignore if {
    input.Findings[0].Type == "Stealth:S3/ServerAccessLoggingDisabled"
}
