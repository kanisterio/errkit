package errkit

import (
	"errors"
	"fmt"
	"runtime"
)

var _ error = (*errkitError)(nil)
var _ interface {
	Unwrap() error
} = (*errkitError)(nil)

// Make an aliases for errors.Is, errors.As, errors.Unwrap
// To avoid additional imports
var (
	Is             = errors.Is
	As             = errors.As
	Unwrap         = errors.Unwrap
	NewSentinelErr = errors.New
)

type errkitError struct {
	error
	cause   error
	details ErrorDetails
	stack   []uintptr
	callers int
}

func (e *errkitError) Is(target error) bool {
	if target == nil {
		return e == target
	}

	// Check if the target error is of the same type and value
	return errors.Is(e.error, target)
}

// New returns an error with the given message.
func New(message string, details ...any) error {
	return newError(errors.New(message), 2, details...)
}

// Wrap returns a new errkitError that has the given message and err as the cause.
func Wrap(err error, message string, details ...any) error {
	e := newError(errors.New(message), 2, details...)
	e.cause = err
	return e
}

// WithStack wraps the given error with a struct that when serialized to JSON will return
// a JSON object with the error message and error stack data.
//
// Returns nil when nil is passed.
//
// NOTE: You should not pass result of errkit.New, errkit.Wrap, errkit.WithCause here.
//
//	var ErrWellKnownError = errors.New("Well-known error")
//	...
//	if someCondition {
//	    return errkit.WithStack(ErrWellKnownError)
//	}
func WithStack(err error, details ...any) error {
	if err == nil {
		return nil
	}

	e := newError(err, 2, details...)
	return e
}

// WithCause adds a cause to the given pure error.
// It returns nil when passed error is nil.
//
// NOTE: You should not pass result of errkit.New, errkit.Wrap, errkit.WithStack here.
//
// Intended for use when a function wants to return a well known error,
// but at the same time wants to add a reason E.g.:
//
// var ErrNotFound = errkit.NewSentinelErr("Resource not found")
// ...
//
//	func SomeFunc() error {
//	  ...
//	  err := DoSomething()
//	  if err != nil {
//	     return errkit.WithCause(ErrNotFound, err)
//	  }
//	  ...
//	}
func WithCause(err, cause error, details ...any) error {
	if err == nil {
		return nil
	}

	e := newError(err, 2, details...)
	e.cause = cause
	return e
}

func newError(err error, stackDepth int, details ...any) *errkitError {
	result := &errkitError{
		error:   err,
		details: ToErrorDetails(details),
		stack:   make([]uintptr, 1),
	}

	result.callers = runtime.Callers(stackDepth+1, result.stack)

	return result
}

// Unwrap returns the chained causal error, or nil if there is no causal error.
func (e *errkitError) Unwrap() error {
	return e.cause
}

// Message returns the message for this error.
func (e *errkitError) Message() string {
	return e.error.Error()
}

// Details returns the map of details in this error.
func (e *errkitError) Details() ErrorDetails {
	return e.details
}

// Error returns a string representation of the error.
func (e *errkitError) Error() string {
	if e.cause == nil {
		return e.error.Error()
	}

	return fmt.Sprintf("%s: %s", e.error.Error(), e.cause.Error())
}

// MarshalJSON is helping json logger to log error in a json format
func (e *errkitError) MarshalJSON() ([]byte, error) {
	return MarshalErrkitErrorToJSON(e)
}
