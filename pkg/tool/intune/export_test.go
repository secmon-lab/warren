package intune

// SetTokenEndpoint overrides the token endpoint for testing.
func (x *Action) SetTokenEndpoint(endpoint string) {
	x.tokenEndpoint = endpoint
}

// SetBaseURL overrides the base URL for testing.
func (x *Action) SetBaseURL(baseURL string) {
	x.baseURL = baseURL
}
