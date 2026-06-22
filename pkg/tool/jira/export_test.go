package jira

// SetConfig sets the connection parameters directly (test-only).
func (x *Action) SetConfig(baseURL, email, apiToken string) {
	x.baseURL = baseURL
	x.email = email
	x.apiToken = apiToken
}
