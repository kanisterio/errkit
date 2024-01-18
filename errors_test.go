package errkit_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/kanisterio/errkit"
	"github.com/kanisterio/errkit/internal/caller"
)

type testErrorType struct {
	message string
}

func (e *testErrorType) Error() string {
	return e.message
}

func newTestError(msg string) *testErrorType {
	return &testErrorType{
		message: msg,
	}
}

var (
	predefinedStdError    = errors.New("TEST_ERR: Sample of predefined std error")
	predefinedErrkitError = errkit.NewPureError("TEST_ERR: Sample of errkit error")
	predefinedTestError   = newTestError("TEST_ERR: Sample error of custom test type")
)

type Check func(originalErr error, jsonErr errkit.JSONError) error

func getMessageCheck(msg string) Check {
	return func(_ error, err errkit.JSONError) error {
		if err.Message != msg {
			return fmt.Errorf("error message does not match the expectd\nexpected: %s\nactual: %s", msg, err.Message)
		}
		return nil
	}
}

func filenameCheck(_ error, err errkit.JSONError) error {
	_, filename, _, _ := runtime.Caller(1)
	if !strings.HasPrefix(err.File, filename) {
		return fmt.Errorf("error occured in an unexpected file: %s", err.File)
	}
	return nil
}

func getStackCheck(fnName string, lineNumber int) Check {
	return func(err error, jsonErr errkit.JSONError) error {
		e := filenameCheck(err, jsonErr)
		if e != nil {
			return e
		}

		if jsonErr.LineNumber != lineNumber {
			return fmt.Errorf("Line number does not match\nexpected: %d\ngot: %d", lineNumber, jsonErr.LineNumber)
		}

		if jsonErr.Function != fnName {
			return fmt.Errorf("Function name does not match\nexpected: %s\ngot: %s", fnName, jsonErr.Function)
		}

		return nil
	}
}

func getErrkitIsCheck(cause error) Check {
	return func(origErr error, jsonErr errkit.JSONError) error {
		if !errkit.Is(origErr, cause) {
			return errors.New("error is not implementing requested type")
		}

		return nil
	}
}

func getUnwrapCheck(expected error) Check {
	return func(origErr error, jsonErr errkit.JSONError) error {
		err1 := errors.Unwrap(origErr)
		if err1 != expected {
			return errors.New("Unable to unwrap error")
		}

		return nil
	}
}

func getDetailsCheck(details errkit.ErrorDetails) Check {
	return func(origErr error, jsonErr errkit.JSONError) error {
		if len(details) != len(jsonErr.Details) {
			return errors.New("details don't match")
		}

		for k, v := range details {
			if jsonErr.Details[k] != v {
				return errors.New("details don't match")
			}
		}

		return nil
	}
}

func checkErrorResult(t *testing.T, err error, checks ...Check) {
	t.Helper()

	got := err.Error()

	var unmarshalledError errkit.JSONError
	unmarshallingErr := unmarshalledError.UnmarshalJSON([]byte(got))
	if unmarshallingErr != nil {
		t.Errorf("serialized error is not a JSON: %s\ngot: %s", unmarshallingErr.Error(), err.Error())
		return
	}

	for _, checker := range checks {
		e := checker(err, unmarshalledError)
		if e != nil {
			t.Errorf("%s", e.Error())
			return
		}
	}
}

func TestErrorCreation(t *testing.T) {
	t.Run("It should be possible to create pure errors which could be used as named errors", func(t *testing.T) {
		e := predefinedErrkitError.Error()
		if e != "TEST_ERR: Sample of errkit error" {
			t.Errorf("Unexpected result")
		}
	})
}

func TestErrorsWrapping(t *testing.T) {
	t.Run("It should be possible to wrap std error, which should be stored as cause", func(t *testing.T) {
		wrappedStdError := errkit.Wrap(predefinedStdError, "Wrapped STD error")
		checkErrorResult(t, wrappedStdError,
			getMessageCheck("Wrapped STD error"), // Checking what msg is serialized on top level
			filenameCheck,                        // Checking callstack capture
			getErrkitIsCheck(predefinedStdError), // Checking that original error was successfully wrapped
			getUnwrapCheck(predefinedStdError),   // Checking that it's possible to unwrap wrapped error
		)
	})

	t.Run("It should be possible to wrap errkit error, which should be stored as cause", func(t *testing.T) {
		wrappedErrkitError := errkit.Wrap(predefinedErrkitError, "Wrapped errkit error")
		checkErrorResult(t, wrappedErrkitError,
			getMessageCheck("Wrapped errkit error"), // Checking what msg is serialized on top level
			filenameCheck,                           // Checking callstack capture
			getErrkitIsCheck(predefinedErrkitError), // Checking that original error was successfully wrapped
		)
	})

	t.Run("It should be possible to wrap custom error implementing error interface, which should be stored as cause", func(t *testing.T) {
		wrappedTestError := errkit.Wrap(predefinedTestError, "Wrapped TEST error")
		checkErrorResult(t, wrappedTestError,
			getMessageCheck("Wrapped TEST error"), // Checking what msg is serialized on top level
			filenameCheck,                         // Checking callstack capture
			getErrkitIsCheck(predefinedTestError), // Checking that original error was successfully wrapped
			func(origErr error, jsonErr errkit.JSONError) error {
				var asErr *testErrorType
				if errors.As(origErr, &asErr) {
					if asErr.Error() == predefinedTestError.Error() {
						return nil
					}
					return errors.New("invalid casting of error cause")
				}

				return errors.New("unable to cast error to its cause")
			},
		)
	})

	t.Run("It should be possible to wrap predefined error with specific cause", func(t *testing.T) {
		errorNotFound := errkit.NewPureError("Resource not found")
		cause := errkit.New("Reason why resource not found")
		wrappedErr := errkit.WithCause(errorNotFound, cause)
		checkErrorResult(t, wrappedErr,
			getMessageCheck("Resource not found"), // Check top level msg
			filenameCheck,
			getErrkitIsCheck(cause),         // Check that cause was properly wrapped
			getErrkitIsCheck(errorNotFound), // Check that predefined error is also matchable
			getUnwrapCheck(cause),           // Check that unwrapping of error returns cause
		)
	})

	t.Run("It should be possible to wrap predefined error with specific cause and ErrorDetails", func(t *testing.T) {
		errorNotFound := errkit.NewPureError("Resource not found")
		cause := errkit.New("Reason why resource not found")
		wrappedErr := errkit.WithCause(errorNotFound, cause, "Key", "value")
		checkErrorResult(t, wrappedErr,
			getMessageCheck("Resource not found"), // Check top level msg
			filenameCheck,
			getErrkitIsCheck(cause),                              // Check that cause was properly wrapped
			getErrkitIsCheck(errorNotFound),                      // Check that predefined error is also matchable
			getUnwrapCheck(cause),                                // Check that unwrapping of error returns cause
			getDetailsCheck(errkit.ErrorDetails{"Key": "value"}), // Check that details were added
		)
	})

	t.Run("It should still be possible to wrap error created with errkit.New, despite the fact it is unwanted case", func(t *testing.T) {
		errorNotFound := errkit.New("Resource not found")
		cause := errkit.New("Reason why resource not found")
		wrappedErr := errkit.WithCause(errorNotFound, cause)
		checkErrorResult(t, wrappedErr,
			getMessageCheck("Resource not found"), // Check top level msg
			filenameCheck,
			getErrkitIsCheck(cause), // Check that cause was properly wrapped
			getUnwrapCheck(cause),   // Check that unwrapping of error returns cause
			func(origErr error, jsonErr errkit.JSONError) error { // Check that only message was taken from the errorNotFound
				if errkit.Is(origErr, errorNotFound) {
					return errors.New("error is implementing unexpected type")
				}

				return nil
			},
		)
	})

	t.Run("It should return nil when nil is passed", func(t *testing.T) {
		cause := errkit.New("Reason why resource not found")
		wrappedErr := errkit.WithCause(nil, cause)
		if wrappedErr != nil {
			t.Errorf("nil expected to be returned")
		}
	})
}

func TestErrorsWithDetails(t *testing.T) {
	// Expecting the following JSON (except stack) for most cases
	commonResult := "{\"message\":\"Some error with details\",\"details\":{\"Some numeric detail\":123,\"Some text detail\":\"String value\"}}"

	// Expecting the following JSON (except stack) for special case
	oddResult := "{\"message\":\"Some error with details\",\"details\":{\"Some numeric detail\":\"NOVAL\",\"Some text detail\":\"String value\"}}"
	invalidKeyResult := "{\"message\":\"Some error with details\",\"details\":{\"BADKEY:(123)\":456,\"Some text detail\":\"String value\"}}"
	wrappedResult := "{\"message\":\"Wrapped error\",\"details\":{\"Some numeric detail\":123,\"Some text detail\":\"String value\"},\"cause\":{\"message\":\"TEST_ERR: Sample of errkit error\"}}"

	getResultCheck := func(expected string) Check {
		return func(orig error, _ errkit.JSONError) error {
			errStr := orig.Error()
			type simplifiedStruct struct {
				Message string            `json:"message"`
				Details map[string]any    `json:"details,omitempty"`
				Cause   *simplifiedStruct `json:"cause,omitempty"`
			}
			var simpl simplifiedStruct
			e := json.Unmarshal([]byte(errStr), &simpl)
			if e != nil {
				return errors.New("unable to unmarshal json representation of an error")
			}

			simplStr, e := json.Marshal(simpl)
			if e != nil {
				return errors.New("unable to marshal simplified error representation to json")
			}

			if string(simplStr) != expected {
				return fmt.Errorf("serialized error value is not expected: %s\ngot: %s", expected, simplStr)
			}

			return nil
		}
	}

	t.Run("It should be possible to create an error with details", func(t *testing.T) {
		err := errkit.New("Some error with details", "Some text detail", "String value", "Some numeric detail", 123)
		checkErrorResult(t, err, getResultCheck(commonResult))
	})

	t.Run("It should be possible to create an error with details using ErrorDetails map", func(t *testing.T) {
		err := errkit.New("Some error with details", errkit.ErrorDetails{"Some text detail": "String value", "Some numeric detail": 123})
		checkErrorResult(t, err, getResultCheck(commonResult))
	})

	t.Run("It should be possible to wrap an error and add details at once", func(t *testing.T) {
		err := errkit.Wrap(predefinedErrkitError, "Wrapped error", "Some text detail", "String value", "Some numeric detail", 123)
		checkErrorResult(t, err, getResultCheck(wrappedResult))
	})

	t.Run("It should be possible to wrap an error and add details at once using ErrorDetails map", func(t *testing.T) {
		err := errkit.Wrap(predefinedErrkitError, "Wrapped error", errkit.ErrorDetails{"Some text detail": "String value", "Some numeric detail": 123})
		checkErrorResult(t, err, getResultCheck(wrappedResult))
	})

	t.Run("It should be possible to create an error with details, even when odd number of values passed", func(t *testing.T) {
		err := errkit.New("Some error with details", "Some text detail", "String value", "Some numeric detail")
		checkErrorResult(t, err, getResultCheck(oddResult))
	})

	t.Run("It should be possible to create an error with details, even when detail name is not a string", func(t *testing.T) {
		err := errkit.New("Some error with details", "Some text detail", "String value", 123, 456)
		checkErrorResult(t, err, getResultCheck(invalidKeyResult))
	})
}

func getStackInfo() (string, int) {
	c := caller.GetFrame(2)
	return c.Function, c.Line
}

func TestErrorsWithStack(t *testing.T) {
	t.Run("It should be possible to bind predefined error to current execution location", func(t *testing.T) {
		fnName, lineNumber := getStackInfo()
		err := errkit.WithStack(predefinedTestError)
		checkErrorResult(t, err,
			getStackCheck(fnName, lineNumber+1),
		)
	})

	t.Run("It should be possible to bind predefined error to current execution location and add some details", func(t *testing.T) {
		fnName, lineNumber := getStackInfo()
		err := errkit.WithStack(predefinedTestError, "Key", "value")
		checkErrorResult(t, err,
			getStackCheck(fnName, lineNumber+1),
			getDetailsCheck(errkit.ErrorDetails{"Key": "value"}),
		)
	})

	t.Run("It should be possible to bind error created with errkit.New, despite the fact it is unwanted case", func(t *testing.T) {
		errorNotFound := errkit.New("Resource not found")
		fnName, lineNumber := getStackInfo()
		err := errkit.WithStack(errorNotFound)
		checkErrorResult(t, err,
			getMessageCheck("Resource not found"), // Check top level msg
			getStackCheck(fnName, lineNumber+1),
			getErrkitIsCheck(errorNotFound), // Check that errorNotWanted is still matchable
			getUnwrapCheck(errorNotFound),   // Check that we are able to unwrap original error
		)
	})

	t.Run("It should return nil when nil is passed", func(t *testing.T) {
		wrappedErr := errkit.WithStack(nil)
		if wrappedErr != nil {
			t.Errorf("nil expected to be returned")
		}
	})
}

func TestMultipleErrors(t *testing.T) {
	t.Run("It should be possible to append errors of different types", func(t *testing.T) {
		err1 := errors.New("First error is an stderror")
		err2 := newTestError("Second error is a test erorr")
		err := errkit.Append(err1, err2)
		str := err.Error()
		expectedStr := "[\"First error is an stderror\",\"Second error is a test erorr\"]"

		if str != expectedStr {
			t.Errorf("Unexpected result.\nexpected: %s\ngot: %s", expectedStr, str)
			return
		}
	})

	t.Run("It should be possible to use errors.Is and errors.As with error list", func(t *testing.T) {
		err := errkit.Append(predefinedStdError, predefinedTestError)

		if !errors.Is(err, predefinedTestError) {
			t.Errorf("Predefined error of test error type is not found in an errors list")
			return
		}

		if !errors.Is(err, predefinedStdError) {
			t.Errorf("Predefined error of std error type is not found in an errors list")
			return
		}

		var testErr *testErrorType
		if !errors.As(err, &testErr) {
			t.Errorf("Unable to reassign error to test type")
			return
		}
	})

	t.Run("It should NOT be possible to unwrap an error from errors list", func(t *testing.T) {
		err := errkit.Append(predefinedStdError, predefinedTestError)
		if errors.Unwrap(err) != nil {
			t.Errorf("Unexpected unwrapping result")
			return
		}
	})

	t.Run("It should be possible to append multiple errkit.errkitError to errors list", func(t *testing.T) {
		someErr := errkit.New("Some test error")
		err := errkit.Append(predefinedErrkitError, someErr)
		str := err.Error()

		someErrStr := someErr.Error()
		predefinedErrStr := predefinedErrkitError.Error()

		arr := append(make([]string, 0), predefinedErrStr, someErrStr)
		arrStr, _ := json.Marshal(arr)

		expectedStr := string(arrStr)

		if str != expectedStr {
			t.Errorf("unexpected serialized output\nexpected: %s\ngot     : %s", expectedStr, str)
			return
		}
	})
}
