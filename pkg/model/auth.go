package model

type AuthContext struct {
	Google map[string]interface{} `json:"google"`
	SNS    *SNSMessage            `json:"sns"`
}
