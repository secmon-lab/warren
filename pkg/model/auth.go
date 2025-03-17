package model

type AuthContext struct {
	Google map[string]interface{} `json:"google"`
	SNS    *SNSMessage            `json:"sns"`

	Req *AuthHTTPRequest  `json:"req"`
	Env map[string]string `json:"env"`
}

type AuthHTTPRequest struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Body   string              `json:"body"`
	Header map[string][]string `json:"header"`
}
