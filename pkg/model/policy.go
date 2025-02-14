package model

type PolicyResult struct {
	Alert []PolicyAlert `json:"alert"`
}

type PolicyAlert struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Attrs       []Attribute `json:"attrs"`
	Data        any         `json:"data"`
}

type PolicyAuth struct {
	Allow bool `json:"allow"`
}
