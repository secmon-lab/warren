package model

import "github.com/m-mizutani/goerr/v2"

type Lang string

const (
	English  Lang = "en"
	Japanese Lang = "ja"
)

var langNames = map[Lang]string{
	English:  "English",
	Japanese: "Japanese",
}

func (l Lang) Name() string {
	return langNames[l]
}

func (l Lang) Validate() error {
	if _, ok := langNames[l]; !ok {
		return goerr.New("invalid language", goerr.V("lang", l), goerr.V("valid", langNames))
	}
	return nil
}
