package errkit

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/luci/go-render/render"

	"github.com/kanisterio/errkit/internal/caller"
)

var _ error = (*errkitError)(nil)
var _ json.Marshaler = (*errkitError)(nil)
var _ interface {
	Unwrap() error
} = (*errkitError)(nil)

// Make an aliases for errors.Is, errors.As, errors.Unwrap
// To avoid additional imports
var (
	Is           = errors.Is
	As           = errors.As
	Unwrap       = errors.Unwrap
	NewPureError = errors.New
)

type errkitError struct {
	error
	function   string
	file       string
	lineNumber int
	details    ErrorDetails
	cause      error
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
	return newError(errors.New(message), 3, details...)
}

// Wrap returns a new errkitError that has the given message and err as the cause.
func Wrap(err error, message string, details ...any) error {
	e := newError(errors.New(message), 3, details...)
	e.cause = err
	return e
}

// WithStack wraps the given pure error with a struct that when serialized to JSON will return
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

	var message string
	if kerr, ok := err.(*errkitError); ok {
		// We shouldn't pass *errkit.errkitError to this function, but this will
		// protect us from the situation when someone used errkit.New()
		// instead of errors.New() or errkit.NewPureError().
		// Otherwise, the error will be double encoded JSON.
		message = kerr.Message()
	} else {
		message = err.Error()
	}

	e := newError(errors.New(message), 3, details...)
	e.cause = err
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
// var ErrNotFound = errkit.NewPureError("Resource not found")
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

	if kerr, ok := err.(*errkitError); ok {
		// We shouldn't pass *errkit.errkitError to this function, but this will
		// protect us from the situation when someone used errkit.New()
		// instead of errors.New() or errkit.NewPureError().
		// Otherwise, the error will be double encoded JSON.
		err = errors.New(kerr.Message())
	}

	e := newError(err, 3, details...)
	e.cause = cause
	return e
}

func newError(err error, stackDepth int, details ...any) *errkitError {
	c := caller.GetFrame(stackDepth)
	return &errkitError{
		error:      err,
		function:   c.Function,
		lineNumber: c.Line,
		// line number is intentionally appended to the file name
		// this reduces the time needed to read the info and
		// simplifies the navigation in the IDEs.
		file:    fmt.Sprintf("%s:%d", c.File, c.Line),
		details: ToErrorDetails(details),
	}
}

// MarshalJSON returns a JSON encoding of e.
func (e *errkitError) MarshalJSON() ([]byte, error) {
	je := JSONError{
		Message:    e.Message(),
		Function:   e.Function(),
		LineNumber: e.LineNumber(),
		File:       e.File(),
		Details:    e.Details(),
		Cause:      JSONMarshable(e.Unwrap()),
	}
	return json.Marshal(je)
}

// Unwrap returns the chained causal error, or nil if there is no causal error.
func (e *errkitError) Unwrap() error {
	return e.cause
}

// Message returns the message for this error.
func (e *errkitError) Message() string {
	return e.error.Error()
}

// File returns the file path where the error was created.
func (e *errkitError) File() string {
	return e.file
}

// Function is the name of the function from where this error originated.
func (e *errkitError) Function() string {
	return e.function
}

// LineNumber is where this error originated.
func (e *errkitError) LineNumber() int {
	return e.lineNumber
}

// Details returns the map of details in this error.
func (e *errkitError) Details() ErrorDetails {
	return e.details
}

// Error returns a string representation of the error.
func (e *errkitError) Error() string {
	if b, err := e.MarshalJSON(); err == nil {
		return string(b)
	}
	return render.Render(*e)
}
