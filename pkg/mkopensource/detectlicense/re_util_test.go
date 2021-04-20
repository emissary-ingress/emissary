package detectlicense

import (
	"regexp"
	"testing"
)

// reTest isn't used normally, but is super duper useful for debuging
// the complex license regexes.
func reTest(t *testing.T, re *regexp.Regexp, str string) { //nolint:unused
	t.Helper()
	if reMatch(re, []byte(str)) {
		return
	}
	t.Fail()
	reStr := re.String()
	t.Logf("regexp is: %#q", reStr)

	// try to create some helpful feedback
	var n int
	for n = len(reStr); n >= 0; n-- {
		_re, err := regexp.Compile(`\A` + reStr[:n])
		for s := `)`; err != nil && len(s) < 5; s += `)` {
			_re, err = regexp.Compile(`\A` + reStr[:n] + s)
		}
		if err == nil && _re.MatchString(str) {
			break
		}
	}
	if n < len(reStr) {
		t.Logf("working prefix is: %#q", reStr[:n])
	} else {
		t.Logf("unmatched text at the end: %#q", regexp.MustCompile(`\A`+reStr).ReplaceAllString(str, ``))
	}
}

//func TestDebug(t *testing.T) {
//	reTest(t, YOUR_REGEX, `YOUR_TEST`)
//}
