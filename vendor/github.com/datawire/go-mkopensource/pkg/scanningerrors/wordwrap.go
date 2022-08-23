package scanningerrors

import (
	"strings"
)

const whitespace = " \t\n"

func indexWordSep(str string) int {
	bs := []byte(str)
	for i := 0; i < len(bs); i++ {
		switch {
		case strings.HasPrefix(str[i:], ". "):
			// First space after a period is a non-breaking-space.
			i++
		case strings.ContainsRune(whitespace, rune(bs[i])):
			return i
		}
	}
	return -1
}

func Wordwrap(indent, width int, str string) string {
	// 1. Tokenize the input
	var words []string
	str = strings.TrimLeft(str, whitespace)
	for len(str) > 0 {
		sep := indexWordSep(str)
		if sep < 0 {
			sep = len(str)
		}
		words = append(words, strings.TrimRight(str[:sep], " "))
		str = str[sep:]
		if strings.HasPrefix(str, "\n\n") {
			words = append(words, "\n")
		}
		str = strings.TrimLeft(str, whitespace)
	}
	// 2. Build the output
	linewidth := 0
	ret := new(strings.Builder)
	sep := strings.Repeat(" ", indent)
	for _, word := range words {
		switch {
		case word == "\n":
			ret.WriteString("\n\n")
			linewidth = 0
			sep = strings.Repeat(" ", indent)
		case linewidth > indent && linewidth+len(sep)+len(word) > width:
			ret.WriteString("\n")
			linewidth = 0
			sep = strings.Repeat(" ", indent)
			fallthrough
		default:
			ret.WriteString(sep)
			ret.WriteString(word)
			linewidth += len(sep) + len(word)
			if strings.HasSuffix(word, ".") {
				sep = "  "
			} else {
				sep = " "
			}
		}
	}
	return ret.String()
}
