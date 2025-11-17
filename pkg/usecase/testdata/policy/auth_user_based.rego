package auth.agent

# Test policy: Allow only specific Slack user
allow if {
    input.auth.slack.id == "U_ALLOWED_USER"
}
