package lang

var All = map[string]func() *Strings{
	"en": EN,
	"es": ES,
	"pt": PT,
	"ar": AR,
	"hi": HI,
	"fr": FR,
	"de": DE,
	"ru": RU,
	"tr": TR,
	"sw": SW,
	"it": IT,
}

var Names = map[string]string{
	"en": "English",
	"es": "Spanish",
	"pt": "Portuguese",
	"ar": "Arabic",
	"hi": "Hindi",
	"fr": "French",
	"de": "German",
	"ru": "Russian",
	"tr": "Turkish",
	"sw": "Swahili",
	"it": "Italian",
}
