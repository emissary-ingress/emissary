package derror

import (
	"errors"
	"fmt"
	"strings"
)

// MultiError is an `error` type that represents a list of errors.
//
// Yes, another one.  What's wrong with all of the existing ones?  Looking at the ones that Emissary
// already imports:
//
//  - `github.com/asaskevich/govalidator.Errors` : (1) Doesn't implement .Is().  (2) Output is all
//    on one line and hard to read.
//
//  - `github.com/getkin/kin-openapi/openapi3.MultiError` : Output is all on one line and hard to
//    read.
//
//  - `github.com/prometheus/client_golang/prometheus.MultiError` : (1) Doesn't implement .Is().
//    (2) Output is pretty good, but doesn't handle child errors having newlines in them.
//
//  - `google.golang.org/appengine.MultiError` : (1) Doesn't implement .Is().  (2) Output is
//    "helpfully" summarized to only print the first error and then say "(and X other errors)".
//
//  - `k8s.io/apimachinery/pkg/util/errors.Aggregate` : (1) Output is all on one line and hard to
//    read.  (2) Isn't a simple []error type, and so is clunky to work with.
//
//  - `sigs.k8s.io/controller-tools/pkg/loader.ErrList` : (1) Doesn't implement .Is().  (2) Output
//    is all on one line and hard to read.
//
// Note that (like k8s.io/apimachinery/pkg/util/errors.Aggregate) we don't implement .As(), because
// we buy in to the same justification:
//
//   // Errors.As() is not supported, because the caller presumably cares about a
//   // specific error of potentially multiple that match the given type.
type MultiError []error

func (errs MultiError) Error() string {
	switch len(errs) {
	case 0:
		// This should not happen.
		return "(0 errors; BUG: this should not be reported as an error)"
	case 1:
		return errs[0].Error()
	default:
		var buf strings.Builder
		fmt.Fprintf(&buf, "%d errors:", len(errs))
		for i, err := range errs {
			prefix := fmt.Sprintf("\n %d. ", i+1)
			for j, line := range strings.SplitAfter(err.Error(), "\n") {
				if j == 1 {
					// After the first line (j==0), change the prefix to just be
					// spaces.  Exclude the leading "\n" from the prefix because
					// .SplitAfter leaves each line with a trailing "\n".
					prefix = strings.Repeat(" ", len(prefix)-1)
				}
				buf.WriteString(prefix)
				buf.WriteString(line)
			}
		}
		return buf.String()
	}
}

func (errs MultiError) Is(target error) bool {
	for _, err := range errs {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
