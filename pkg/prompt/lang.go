package prompt

import (
	"github.com/m-mizutani/goerr/v2"
)

type Lang string

var defaultLang = LangEnglish

const (
	LangEnglish  Lang = "en"
	LangJapanese Lang = "ja"
)

var langNames = map[Lang]string{
	LangEnglish:  "English",
	LangJapanese: "Japanese",
}

func SetDefaultLang(lang string) error {
	if _, ok := langNames[Lang(lang)]; !ok {
		return goerr.New("invalid language", goerr.V("lang", lang), goerr.V("valid", langNames))
	}
	defaultLang = Lang(lang)
	return nil
}

func (l Lang) name() string {
	return langNames[l]
}
