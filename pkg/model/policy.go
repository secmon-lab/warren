package model

type PolicyResult struct {
	Alert []PolicyAlert `json:"alert"`
}

type PolicyAlert struct {
	Title string      `json:"title"`
	Attrs []Attribute `json:"attrs"`
}
