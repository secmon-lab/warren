package auth

import "github.com/secmon-lab/warren/pkg/domain/model/message"

type Context struct {
	Google map[string]interface{} `json:"google"`
	SNS    *message.SNS           `json:"sns"`

	Req *HTTPRequest      `json:"req"`
	Env map[string]string `json:"-"`
}

type HTTPRequest struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Body   string              `json:"body"`
	Header map[string][]string `json:"header"`
}
