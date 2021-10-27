package detectlicense

import (
	"regexp"
)

func reWrap(str string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllLiteralString(str, `\s+`)
}

func reCompile(re string) *regexp.Regexp {
	return regexp.MustCompile(re)
}

func reCaseInsensitive(re string) string {
	return `(?i:` + re + `)`
}

func reQuote(str string) string {
	return regexp.QuoteMeta(str)
}

func reMatch(re *regexp.Regexp, str []byte) bool {
	return regexp.MustCompile(`\A` + re.String() + `\z`).Match(str)
}
