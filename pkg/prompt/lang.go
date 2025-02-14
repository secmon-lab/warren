package prompt

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

func SetDefaultLang(lang Lang) {
	defaultLang = lang
}

func (l Lang) name() string {
	return langNames[l]
}
