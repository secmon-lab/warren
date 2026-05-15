package webfetch

import "net/http"

// Extract is the test-only export of the package-private extract function.
var Extract = extract

// Analyze is the test-only export of the package-private analyze function.
var Analyze = analyze

// SetHTTPClient injects a custom http.Client for tests (e.g. with httptest server).
func (x *Action) SetHTTPClient(c *http.Client) {
	x.client = c
}
