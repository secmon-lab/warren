package falcon

// SetCredentials sets the client credentials directly (test-only).
func (x *Action) SetCredentials(clientID, clientSecret string) {
	x.clientID = clientID
	x.clientSecret = clientSecret
}

// SetTestURL points the underlying toolset at a stub server (test-only).
func (x *Action) SetTestURL(url string) {
	x.baseURL = url
}
