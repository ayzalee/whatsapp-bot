package plugins

import (
	"sort"
	"strings"
	"zaelix/lang"
)

type Strings = lang.Strings

func T() *Strings {
	code := BotSettings.GetLanguage()
	if fn, ok := lang.All[code]; ok {
		return fn()
	}
	return lang.EN()
}

var LangNames = lang.Names

func langList() string {
	codes := make([]string, 0, len(lang.Names))
	for code := range lang.Names {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	parts := make([]string, len(codes))
	for i, code := range codes {
		parts[i] = code + " (" + lang.Names[code] + ")"
	}
	return strings.Join(parts, ", ")
}
