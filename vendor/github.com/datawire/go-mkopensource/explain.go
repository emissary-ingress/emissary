// -*- fill-column: 100 -*-

package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func categorizeError(errStr string) string {
	switch {
	case strings.Contains(errStr, "something hokey is going on"):
		return "license-approval"
	case strings.Contains(errStr, "unacceptable license"):
		return "license-approval"
	default:
		return "license-detection"
	}
}

var errCategoryExplanations = map[string]string{

	"license-approval": `This probably means that you added or upgraded a dependency, and the
		automated opensource-license-checker objects to what it sees.  This may because of a
		bug in the checker (github.com/datawire/go-mkopensource) that you need to go fix, or
		it may be because of an actual license issue that prevents you from being allowed to
		use a package, and you need to find an alternative.`,

	"license-detection": `This probably means that you added or upgraded a dependency, and the
		automated opensource-license-checker can't confidently detect what the license is.
		(This is a good thing, because it is reminding you to check the license of libraries
		before using them.)

		You need to update the
		"github.com/datawire/go-mkopensource/pkg/detectlicense/licenses.go" file to
		correctly detect the license.`,
}

func ExplainErrors(errs []error) error {
	buckets := make(map[string][]string)
	for _, err := range errs {
		errStr := err.Error()
		cat := categorizeError(errStr)
		buckets[cat] = append(buckets[cat], errStr)
	}

	cats := make([]string, 0, len(buckets))
	for cat := range buckets {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	msg := new(strings.Builder)
	for _, cat := range cats {
		explanation := errCategoryExplanations[cat]
		errStrs := buckets[cat]
		if len(errs) == 1 {
			fmt.Fprintf(msg, "1 %s error:\n", cat)
		} else {
			fmt.Fprintf(msg, "%d %s errors:\n", len(errStrs), cat)
			sort.Strings(errStrs)
		}
		for i, errStr := range errStrs {
			fmt.Fprintf(msg, " %d. %s\n", i+1, errStr)
			if errStr == `package "github.com/josharian/intern": could not identify a license for all sources (had no global LICENSE file)` {
				explanation += `

					For github.com/josharian/intern in particular, this probably
					means that you are depending on an old version; upgrading to
					intern v1.0.1-0.20211109044230-42b52b674af5 or later should
					resolve this.`
			}
		}
		fmt.Fprintln(msg, wordwrap(4, 72, explanation))
	}
	return errors.New(strings.TrimRight(msg.String(), "\n"))
}
