package errkit

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type ErrorList []error

var _ error = ErrorList{}
var _ json.Marshaler = ErrorList{}

func (e ErrorList) String() string {
	sep := ""
	var buf bytes.Buffer
	buf.WriteRune('[')
	for _, err := range e {
		buf.WriteString(sep)
		sep = ","
		buf.WriteString(strconv.Quote(err.Error()))
	}
	buf.WriteRune(']')
	return buf.String()
}

func (e ErrorList) Error() string {
	return e.String()
}

// As allows error.As to work against any error in the list.
func (e ErrorList) As(target any) bool {
	for _, err := range e {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}

// Is allows error.Is to work against any error in the list.
func (e ErrorList) Is(target error) bool {
	for _, err := range e {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (e ErrorList) MarshalJSON() ([]byte, error) {
	var je struct {
		Message string            `json:"message"`
		Errors  []json.RawMessage `json:"errors"`
	}

	switch len(e) {
	case 0:
		// no errors
		return []byte("null"), nil
	case 1:
		// this is unlikely to happen as kerrors.Append won't allow having just a single error on the list
		je.Message = "1 error has occurred"
	default:
		je.Message = fmt.Sprintf("%d errors have occurred", len(e))
	}

	je.Errors = make([]json.RawMessage, 0, len(e))
	for i := range e {
		raw, err := json.Marshal(JSONMarshable(e[i]))
		if err != nil {
			return nil, err
		}

		je.Errors = append(je.Errors, raw)
	}

	return json.Marshal(je)
}

// Append creates a new combined error from err1, err2. If either error is nil,
// then the other error is returned.
func Append(err1, err2 error) error {
	if err1 == nil {
		return err2
	}
	if err2 == nil {
		return err1
	}
	el1, ok1 := err1.(ErrorList)
	el2, ok2 := err2.(ErrorList)
	switch {
	case ok1 && ok2:
		return append(el1, el2...)
	case ok1:
		return append(el1, err2)
	case ok2:
		return append(el2, err1)
	}
	return ErrorList{err1, err2}
}
