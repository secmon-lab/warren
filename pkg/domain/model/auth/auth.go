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

// AgentContext represents the authorization context for agent execution
type AgentContext struct {
	Message string            `json:"message"`           // User's message to the agent
	Env     map[string]string `json:"env" masq:"secret"` // Environment variables
	Auth    *AgentAuthInfo    `json:"auth,omitempty"`    // Authenticated subject information
}

// AgentAuthInfo represents authentication information for agent execution
type AgentAuthInfo struct {
	Slack *SlackAuthInfo `json:"slack,omitempty"`
}

// SlackAuthInfo represents Slack authentication information for authorization
type SlackAuthInfo struct {
	ID string `json:"id"`
}
