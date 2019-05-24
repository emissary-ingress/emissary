package util

import (
	"fmt"

	"github.com/pkg/errors"
)

// causer is not exported by github.com/pkg/errors.
type causer interface {
	Cause() error
}

// stackTracer is not exported by github.com/pkg/errors.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

// featurefulError documents the features of
// github.com/pkg/errors.Wrap().
type featurefulError interface {
	error
	//causer
	stackTracer
	fmt.Formatter
}

type panicError struct {
	featurefulError
}

func (pe panicError) Cause() error { return pe.featurefulError }

var _ causer = panicError{}
var _ featurefulError = panicError{}

// PanicToError takes an arbitrary object returned from recover(), and
// returns an appropriate error.
//
// If the input is nil, then nil is returned.
//
// If the input is an error returned from a previus call to
// PanicToError(), then it is returned verbatim.
//
// If the input is an error, it is wrapped with the message "PANIC:"
// and has a stack trace attached to it.
//
// If the input is anything else, it is formatted with "%+v" and
// returned as an error with a stack trace attached.
func PanicToError(rec interface{}) error {
	if rec == nil {
		return nil
	}
	switch rec := rec.(type) {
	case panicError:
		return rec
	case error:
		return panicError{errors.Wrap(rec, "PANIC").(featurefulError)}
	default:
		return panicError{errors.Errorf("PANIC: %+v", rec).(featurefulError)}
	}
}
