package errkit

import (
	"encoding"
	"encoding/json"
)

type JSONError struct {
	Message    string       `json:"message,omitempty"`
	Function   string       `json:"function,omitempty"`
	LineNumber int          `json:"linenumber,omitempty"`
	File       string       `json:"file,omitempty"`
	Details    ErrorDetails `json:"details,omitempty"`
	Cause      any          `json:"cause,omitempty"`
}

// jsonError is a data structure which helps to deserialize error at once.
type jsonError struct {
	Message    string          `json:"message,omitempty"`
	Function   string          `json:"function,omitempty"`
	LineNumber int             `json:"linenumber,omitempty"`
	File       string          `json:"file,omitempty"`
	Details    ErrorDetails    `json:"fields,omitempty"`
	Cause      json.RawMessage `json:"cause,omitempty"`
}

// UnmarshalJSON return error unmarshaled into JSONError.
func (e *JSONError) UnmarshalJSON(source []byte) error {
	var parsedError *jsonError
	err := json.Unmarshal(source, &parsedError)
	if err != nil {
		return err
	}

	e.Message = parsedError.Message
	e.Function = parsedError.Function
	e.File = parsedError.File
	e.LineNumber = parsedError.LineNumber
	e.Details = parsedError.Details

	if parsedError.Cause == nil {
		return nil
	}

	// Trying to parse as JSONError
	var jsonErrorCause *JSONError
	err = json.Unmarshal(parsedError.Cause, &jsonErrorCause)
	if err == nil {
		e.Cause = jsonErrorCause
		return nil
	}

	//  fallback to any
	var cause any
	err = json.Unmarshal(parsedError.Cause, &cause)
	if err == nil {
		e.Cause = cause
	}
	return err
}

// JSONMarshable attempts to produce a JSON representation of the given err.
// If the resulting string is empty, then the JSON encoding of the err.Error()
// string is returned or empty if the Error() string cannot be encoded.
func JSONMarshable(err error) any {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case json.Marshaler, encoding.TextMarshaler:
		return err
	default:
		// Otherwise wrap the error with {"message":"…"}
		return JSONError{Message: err.Error()}
	}
}
