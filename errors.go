package errkit

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/luci/go-render/render"

	"github.com/kanisterio/errkit/internal/caller"
)

var _ error = (*PureError)(nil)
var _ error = (*Error)(nil)
var _ json.Marshaler = (*Error)(nil)
var _ interface {
	Unwrap() error
} = (*Error)(nil)

// Make an aliases for errors.Is, errors.As, errors.Unwrap
// To avoid additional imports
var (
	Is     = errors.Is
	As     = errors.As
	Unwrap = errors.Unwrap
)

type PureError struct {
	error
}

func NewPureError(message string) *PureError {
	return &PureError{
		error: errors.New(message),
	}
}

type Error struct {
	error
	function   string
	file       string
	lineNumber int
	details    ErrorDetails
	cause      error
	baseTag    *int // detect shallow copies, not needed in json
	clonedFrom *Error
}

// New returns an error with the given message.
func New(message string, details ...any) *Error {
	return newError(message, 3, details...)
}

// Wrap returns a new Error that has the given message and err as the cause.
func Wrap(err error, message string, details ...any) *Error {
	return wrap(err, message, details...)
}

// WithStack wraps the given error with a struct that when serialized to JSON will return
// a JSON object with the error message and error stack data.
//
// Note, this function should only be used with error values. E.g.:
//
//	var ErrWellKnownError = errors.New("Well-known error")
//	...
//	if someCondition {
//	    return errkit.WithStack(ErrWellKnownError)
//	}
func WithStack(err error) *Error {
	var message string
	if kerr, ok := err.(*Error); ok {
		// We shouldn't pass *errkit.Error to this function, but this will
		// protect us from the situation when someone used errkit.New()
		// instead of errors.New() or errkit.NewPureError(). Otherwise, the error will be double encoded
		// JSON.
		message = kerr.Message()
	} else {
		message = err.Error()
	}

	e := newError(message, 3)
	e.cause = err
	return e
}

func wrap(err error, message string, details ...any) *Error {
	e := newError(message, 4, details...)
	e.cause = err
	return e
}

func newError(message string, stackDepth int, details ...any) *Error {
	c := caller.GetFrame(stackDepth)
	e := &Error{
		error:      errors.New(message),
		function:   c.Function,
		lineNumber: c.Line,
		// line number is intentionally appended to the file name
		// this reduces the time needed to read the info and
		// simplifies the navigation in the IDEs.
		file:    fmt.Sprintf("%s:%d", c.File, c.Line),
		details: ToErrorDetails(details),
	}
	// shallow copies (e.g. WithField(s)) share baseTag pointer to otherwise unused int
	tag := 42
	e.baseTag = &tag
	return e
}

// WithCause adds a cause to the given error.
func WithCause(err, cause error) error {
	c := caller.GetFrame(3)
	e := &Error{
		error:      err,
		function:   c.Function,
		lineNumber: c.Line,
		// line number is intentionally appended to the file name
		// this reduces the time needed to read the info and
		// simplifies the navigation in the IDEs.
		file:  fmt.Sprintf("%s:%d", c.File, c.Line),
		cause: cause,
	}
	// shallow copies (e.g. WithField(s)) share baseTag pointer to otherwise unused int
	tag := 42
	e.baseTag = &tag
	return e
}

// MarshalJSON returns a JSON encoding of e.
func (e *Error) MarshalJSON() ([]byte, error) {
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
func (e *Error) Unwrap() error {
	return e.cause
}

// Message returns the message for this error.
func (e *Error) Message() string {
	return e.error.Error()
}

// File returns the file path where the error was created.
func (e *Error) File() string {
	return e.file
}

// Function is the name of the function from where this error originated.
func (e *Error) Function() string {
	return e.function
}

// LineNumber is where this error originated.
func (e *Error) LineNumber() int {
	return e.lineNumber
}

// Details returns the map of details in this error.
func (e *Error) Details() ErrorDetails {
	return e.details
}

// WithDetail copies of this error and adds the given detail to the new error. The
// new error is returned.
func (e *Error) WithDetail(name string, value any) *Error {
	return e.WithDetails(ErrorDetails{name: value})
}

func (e *Error) WithDetails(details ...any) *Error {
	ne := *e // Shallow clone
	ne.clonedFrom = e

	if len(details) == 0 {
		return &ne
	}

	var newDetails ErrorDetails = make(ErrorDetails, len(e.details)+len(details))
	for k, v := range e.details {
		newDetails[k] = v
	}

	for k, v := range ToErrorDetails(details) {
		newDetails[k] = v
	}
	ne.details = newDetails
	return &ne
}

// Error returns a string representation of the error.
func (e *Error) Error() string {
	if b, err := e.MarshalJSON(); err == nil {
		return string(b)
	}
	return render.Render(*e)
}
