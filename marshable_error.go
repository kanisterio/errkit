package errkit

import (
	"encoding"
	"encoding/json"
	"runtime"
	"strings"
)

func getLocationFromStack(stack []uintptr, callers int) (function, file string, line int) {
	if callers < 1 {
		// Failure potentially due to wrongly specified depth
		return "Unknown", "Unknown", 0
	}

	frames := runtime.CallersFrames(stack[:callers])
	var frame runtime.Frame
	frame, _ = frames.Next()
	filename := frame.File
	if paths := strings.SplitAfterN(frame.File, "/go/src/", 2); len(paths) > 1 {
		filename = paths[1]
	}

	return frame.Function, filename, frame.Line
}

type jsonError struct {
	Message    string       `json:"message,omitempty"`
	Function   string       `json:"function,omitempty"`
	LineNumber int          `json:"linenumber,omitempty"`
	File       string       `json:"file,omitempty"`
	Details    ErrorDetails `json:"details,omitempty"`
	Cause      any          `json:"cause,omitempty"`
}

// UnmarshalJSON return error unmarshaled into jsonError.
func (e *jsonError) UnmarshalJSON(source []byte) error {
	var parsedError struct {
		Message    string          `json:"message,omitempty"`
		Function   string          `json:"function,omitempty"`
		LineNumber int             `json:"linenumber,omitempty"`
		File       string          `json:"file,omitempty"`
		Details    ErrorDetails    `json:"details,omitempty"`
		Cause      json.RawMessage `json:"cause,omitempty"`
	}
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

	// Trying to parse as jsonError
	var jsonErrorCause *jsonError
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

// jsonMarshable attempts to produce a JSON representation of the given err.
// If the resulting string is empty, then the JSON encoding of the err.Error()
// string is returned or empty if the Error() string cannot be encoded.
func jsonMarshable(err error) any {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case json.Marshaler, encoding.TextMarshaler:
		return err
	default:
		// Otherwise wrap the error with {"message":"…"}
		return jsonError{Message: err.Error()}
	}
}

func MarshalErrkitErrorToJSON(err *errkitError) ([]byte, error) {
	if err == nil {
		return nil, nil
	}

	function, file, line := getLocationFromStack(err.stack, err.callers)

	result := jsonError{
		Message:    err.Message(),
		Function:   function,
		LineNumber: line,
		File:       file,
		Details:    err.Details(),
	}

	if err.cause != nil {
		if kerr, ok := err.cause.(*errkitError); ok {
			causeJSON, err := MarshalErrkitErrorToJSON(kerr)
			if err != nil {
				return nil, err
			}

			result.Cause = json.RawMessage(causeJSON)
		} else {
			result.Cause = jsonMarshable(err.cause)
		}
	}

	return json.Marshal(result)
}
