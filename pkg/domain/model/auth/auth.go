package auth

import "github.com/secmon-lab/warren/pkg/domain/model/message"

type Context struct {
	Google map[string]interface{} `json:"google"`
	IAP    map[string]interface{} `json:"iap"`
	SNS    *message.SNS           `json:"sns"`

	Req *HTTPRequest      `json:"req"`
	Env map[string]string `json:"env" masq:"secret"`
}

type HTTPRequest struct {
	Method string              `json:"method"`
	Path   string              `json:"path"`
	Body   string              `json:"body"`
	Header map[string][]string `json:"header"`
}
