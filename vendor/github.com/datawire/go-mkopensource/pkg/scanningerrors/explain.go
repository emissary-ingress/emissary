// -*- fill-column: 100 -*-

package scanningerrors

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	licenseIssue      string = "license-approval"
	licenseDetection  string = "license-detection"
	internalUsageOnly string = "intended-usage"
	licenseForbidden  string = "license-forbidden"
)

func categorizeError(errStr string) string {
	switch {
	case strings.Contains(errStr, "something hokey is going on"):
		return licenseIssue
	case strings.Contains(errStr, "is missing a license identifier"):
		return licenseDetection
	case strings.Contains(errStr, "is forbidden"):
		return licenseForbidden
	case strings.Contains(errStr, "which is not allowed on applications"):
		return internalUsageOnly
	default:
		return licenseDetection
	}
}

var errCategoryExplanations = map[string]string{

	licenseIssue: `This probably means that you added or upgraded a dependency, and the
		automated opensource-license-checker objects to what it sees.  This may because of a
		bug in the checker (github.com/datawire/go-mkopensource) that you need to go fix, or
		it may be because of an actual license issue that prevents you from being allowed to
		use a package, and you need to find an alternative.`,

	licenseDetection: `This probably means that you added or upgraded a dependency, and the
		automated opensource-license-checker can't confidently detect what the license is.
		(This is a good thing, because it is reminding you to check the license of libraries
		before using them.)

		Some possible causes for  this issue are:

		- Dependency is proprietary Ambassador Labs software: Update function 
		  IsAmbassadorProprietarySoftware() to correctly identify the 
		  dependency

		- License information can't be identified: Add an entry to 
          hardcodedGoDependencies, hardcodedPythonDependencies 
          or hardcodedJsDependencies depending on the dependency that
          was not identified.`,

	internalUsageOnly: `To solve this error, replace the dependency with another that uses an acceptable license.

        Refer to https://www.notion.so/datawire/License-Management-5194ca50c9684ff4b301143806c92157#1cd50aeeafa7456bba24c761c0a2d173 
        for more details.`,

	licenseForbidden: `To solve this error, replace the dependency with another that uses an acceptable license.

        Refer to https://www.notion.so/datawire/License-Management-5194ca50c9684ff4b301143806c92157#1cd50aeeafa7456bba24c761c0a2d173 
        for more details.`,
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
			_, _ = fmt.Fprintf(msg, "1 %s error:\n", cat)
		} else {
			_, _ = fmt.Fprintf(msg, "%d %s errors:\n", len(errStrs), cat)
			sort.Strings(errStrs)
		}
		for i, errStr := range errStrs {
			_, _ = fmt.Fprintf(msg, " %d. %s\n", i+1, errStr)
			if errStr == `Package "github.com/josharian/intern": could not identify a license for all sources (had no global LICENSE file)` {
				explanation += `

					For github.com/josharian/intern in particular, this probably
					means that you are depending on an old version; upgrading to
					intern v1.0.1-0.20211109044230-42b52b674af5 or later should
					resolve this.`
			}
		}
		_, _ = fmt.Fprintln(msg, Wordwrap(4, 72, explanation))
	}
	return errors.New(strings.TrimRight(msg.String(), "\n"))
}
