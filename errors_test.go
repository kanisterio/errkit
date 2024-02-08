package errkit_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/kanisterio/errkit"
	"github.com/kanisterio/errkit/internal/stack"
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
	predefinedStdError      = errors.New("TEST_ERR: Sample of predefined std error")
	predefinedSentinelError = errkit.NewSentinelErr("TEST_ERR: Sample of sentinel error")
	predefinedTestError     = newTestError("TEST_ERR: Sample error of custom test type")
)

type Check func(originalErr error, data []byte) error

func unmarshalJsonError(data []byte, target any) error {
	unmarshallingErr := json.Unmarshal(data, target)
	if unmarshallingErr != nil {
		return fmt.Errorf("Unable to unmarshal error %s\n%s", string(data), unmarshallingErr.Error())
	}

	return nil
}

func getMessageCheck(msg string) Check {
	return func(err error, data []byte) error {
		var unmarshalledError struct {
			Message string `json:"message,omitempty"`
		}

		if e := unmarshalJsonError(data, &unmarshalledError); e != nil {
			return e
		}

		if unmarshalledError.Message != msg {
			return fmt.Errorf("error message does not match the expectd\nexpected: %s\nactual: %s", msg, unmarshalledError.Message)
		}
		return nil
	}
}

func getTextCheck(msg string) Check {
	return func(origError error, _ []byte) error {
		if origError.Error() != msg {
			return fmt.Errorf("error text does not match the expected\nexpected: %s\nactual: %s", msg, origError.Error())
		}
		return nil
	}
}

func filenameCheck(_ error, data []byte) error {
	_, filename, _, _ := runtime.Caller(1)
	var unmarshalledError struct {
		File string `json:"file,omitempty"`
	}
	if e := unmarshalJsonError(data, &unmarshalledError); e != nil {
		return e
	}

	if !strings.HasPrefix(unmarshalledError.File, filename) {
		return fmt.Errorf("error occured in an unexpected file. expected: %s\ngot: %s", filename, unmarshalledError.File)
	}

	return nil
}

func getStackCheck(fnName string, lineNumber int) Check {
	return func(err error, data []byte) error {
		e := filenameCheck(err, data)
		if e != nil {
			return e
		}

		var unmarshalledError struct {
			LineNumber int    `json:"linenumber,omitempty"`
			Function   string `json:"function,omitempty"`
		}

		if e := unmarshalJsonError(data, &unmarshalledError); e != nil {
			return e
		}

		if unmarshalledError.LineNumber != lineNumber {
			return fmt.Errorf("Line number does not match\nexpected: %d\ngot: %d", lineNumber, unmarshalledError.LineNumber)
		}

		if unmarshalledError.Function != fnName {
			return fmt.Errorf("Function name does not match\nexpected: %s\ngot: %s", fnName, unmarshalledError.Function)
		}

		return nil
	}
}

func getErrkitIsCheck(cause error) Check {
	return func(origErr error, _ []byte) error {
		if !errkit.Is(origErr, cause) {
			return errors.New("error is not implementing requested type")
		}

		return nil
	}
}

func getUnwrapCheck(expected error) Check {
	return func(origErr error, _ []byte) error {
		err1 := errors.Unwrap(origErr)
		if err1 != expected {
			return errors.New("Unable to unwrap error")
		}

		return nil
	}
}

func getDetailsCheck(details errkit.ErrorDetails) Check {
	return func(_ error, data []byte) error {
		var unmarshalledError struct {
			Details errkit.ErrorDetails `json:"details,omitempty"`
		}

		if e := unmarshalJsonError(data, &unmarshalledError); e != nil {
			return e
		}

		if len(details) != len(unmarshalledError.Details) {
			return errors.New("details don't match")
		}

		for k, v := range details {
			if unmarshalledError.Details[k] != v {
				return errors.New("details don't match")
			}
		}

		return nil
	}
}

func checkErrorResult(t *testing.T, err error, checks ...Check) {
	t.Helper()

	got, e := json.Marshal(err)
	if e != nil {
		t.Errorf("Error marshaling failed: %s", e.Error())
		return
	}

	for _, checker := range checks {
		e := checker(err, got)
		if e != nil {
			t.Errorf("%s", e.Error())
			return
		}
	}
}

func TestErrorCreation(t *testing.T) {
	t.Run("It should be possible to create sentinel errors", func(t *testing.T) {
		e := predefinedSentinelError.Error()
		if e != "TEST_ERR: Sample of sentinel error" {
			t.Errorf("Unexpected result")
		}
	})
}

func TestErrorsWrapping(t *testing.T) {
	t.Run("It should be possible to wrap std error, which should be stored as cause", func(t *testing.T) {
		wrappedStdError := errkit.Wrap(predefinedStdError, "Wrapped STD error")
		checkErrorResult(t, wrappedStdError,
			getMessageCheck("Wrapped STD error"),                                        // Checking what msg is serialized on top level
			getTextCheck("Wrapped STD error: TEST_ERR: Sample of predefined std error"), // Checking what error message is generated
			filenameCheck,                        // Checking callstack capture
			getErrkitIsCheck(predefinedStdError), // Checking that original error was successfully wrapped
			getUnwrapCheck(predefinedStdError),   // Checking that it's possible to unwrap wrapped error
		)
	})

	t.Run("It should be possible to wrap errkit sentinel error, which should be stored as cause", func(t *testing.T) {
		wrappedErrkitError := errkit.Wrap(predefinedSentinelError, "Wrapped errkit error")
		checkErrorResult(t, wrappedErrkitError,
			getMessageCheck("Wrapped errkit error"), // Checking what msg is serialized on top level
			getTextCheck("Wrapped errkit error: TEST_ERR: Sample of sentinel error"),
			filenameCheck, // Checking callstack capture
			getErrkitIsCheck(predefinedSentinelError), // Checking that original error was successfully wrapped
		)
	})

	t.Run("It should be possible to wrap errkit error, which should be stored as cause", func(t *testing.T) {
		someErrkitError := errkit.New("Original errkit error")
		wrappedErrkitError := errkit.Wrap(someErrkitError, "Wrapped errkit error")
		checkErrorResult(t, wrappedErrkitError,
			getMessageCheck("Wrapped errkit error"), // Checking what msg is serialized on top level
			getTextCheck("Wrapped errkit error: Original errkit error"),
			filenameCheck,                     // Checking callstack capture
			getErrkitIsCheck(someErrkitError), // Checking that original error was successfully wrapped
		)
	})

	t.Run("It should be possible to wrap custom error implementing error interface, which should be stored as cause", func(t *testing.T) {
		wrappedTestError := errkit.Wrap(predefinedTestError, "Wrapped TEST error")
		checkErrorResult(t, wrappedTestError,
			getMessageCheck("Wrapped TEST error"), // Checking what msg is serialized on top level
			filenameCheck,                         // Checking callstack capture
			getErrkitIsCheck(predefinedTestError), // Checking that original error was successfully wrapped
			func(origErr error, _ []byte) error {
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
		errorNotFound := errkit.NewSentinelErr("Resource not found")
		cause := errkit.New("Reason why resource not found")
		wrappedErr := errkit.WithCause(errorNotFound, cause)
		checkErrorResult(t, wrappedErr,
			getMessageCheck("Resource not found"), // Check top level msg
			getTextCheck("Resource not found: Reason why resource not found"),
			filenameCheck,
			getErrkitIsCheck(cause),         // Check that cause was properly wrapped
			getErrkitIsCheck(errorNotFound), // Check that predefined error is also matchable
			getUnwrapCheck(cause),           // Check that unwrapping of error returns cause
		)
	})

	t.Run("It should be possible to wrap predefined error with specific cause and ErrorDetails", func(t *testing.T) {
		errorNotFound := errkit.NewSentinelErr("Resource not found")
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
			getErrkitIsCheck(cause),         // Check that cause was properly wrapped
			getErrkitIsCheck(errorNotFound), // Check that predefined error is also matchable
			getUnwrapCheck(cause),           // Check that unwrapping of error returns cause
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
	wrappedResult := "{\"message\":\"Wrapped error\",\"details\":{\"Some numeric detail\":123,\"Some text detail\":\"String value\"},\"cause\":{\"message\":\"TEST_ERR: Sample of sentinel error\"}}"

	getResultCheck := func(expected string) Check {
		return func(orig error, _ []byte) error {
			b, _ := json.Marshal(orig)
			errStr := string(b)
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
		err := errkit.Wrap(predefinedSentinelError, "Wrapped error", "Some text detail", "String value", "Some numeric detail", 123)
		checkErrorResult(t, err, getResultCheck(wrappedResult))
	})

	t.Run("It should be possible to wrap an error and add details at once using ErrorDetails map", func(t *testing.T) {
		err := errkit.Wrap(predefinedSentinelError, "Wrapped error", errkit.ErrorDetails{"Some text detail": "String value", "Some numeric detail": 123})
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
	fpcs := make([]uintptr, 1)
	num := runtime.Callers(2, fpcs)
	fn, _, line := stack.GetLocationFromStack(fpcs, num)
	return fn, line
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
			getErrkitIsCheck(errorNotFound), // Check that errorNotFound is still matchable
			getUnwrapCheck(nil),             // Check that we are able to unwrap original error
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
		err := errkit.Append(predefinedSentinelError, someErr)
		str := err.Error()

		someErrStr := someErr.Error()
		predefinedErrStr := predefinedSentinelError.Error()

		arr := append(make([]string, 0), predefinedErrStr, someErrStr)
		arrStr, _ := json.Marshal(arr)

		expectedStr := string(arrStr)

		if str != expectedStr {
			t.Errorf("unexpected serialized output\nexpected: %s\ngot     : %s", expectedStr, str)
			return
		}
	})

	t.Run("It should return list of errors when trying to append nil to error", func(t *testing.T) {
		err1 := errors.New("First error is an stderror")
		err2 := newTestError("Second error is a test error")
		err := errkit.Append(err1, nil)
		str := err.Error()
		expectedStr := "[\"First error is an stderror\"]"

		if str != expectedStr {
			t.Errorf("Unexpected result.\nexpected: %s\ngot: %s", expectedStr, str)
			return
		}

		err = errkit.Append(nil, err2)
		str = err.Error()
		expectedStr = "[\"Second error is a test error\"]"

		if str != expectedStr {
			t.Errorf("Unexpected result.\nexpected: %s\ngot: %s", expectedStr, str)
			return
		}

		err = errkit.Append(err, err1)
		str = err.Error()
		expectedStr = "[\"Second error is a test error\",\"First error is an stderror\"]"

		if str != expectedStr {
			t.Errorf("Unexpected result.\nexpected: %s\ngot: %s", expectedStr, str)
			return
		}
	})
}

func TestStackViaGoroutine(t *testing.T) {
	t.Run("It should be possible to keep erorr stack when passing an error via goroutine", func(t *testing.T) {
		var wg sync.WaitGroup
		errCh := make(chan error)

		sentinelErr := errkit.NewSentinelErr("Sentinel error")

		fnName, lineNumber := getStackInfo()
		var lock sync.Mutex
		var orderedFailures []int
		performOperation := func(id int) {
			defer wg.Done()

			// Simulate an operation resulting in an error
			if id != 2 {
				lock.Lock()
				defer lock.Unlock()
				orderedFailures = append(orderedFailures, id)
				errCh <- errkit.WithStack(sentinelErr, "id", id)
			}
		}

		doOperationsConcurrently := func() error {
			// Run operations in goroutines
			wg.Add(3)
			go performOperation(1) // This operation will fail
			go performOperation(2) // This operation will succeed
			go performOperation(3) // This operation will fail

			go func() {
				wg.Wait()
				close(errCh)
			}()
			var result error

			// Collect errors from the channel
			for err := range errCh {
				result = errkit.Append(result, err)
			}

			return result
		}

		err := doOperationsConcurrently()
		expectedErrorString := "[\"Sentinel error\",\"Sentinel error\"]"
		if err.Error() != expectedErrorString {
			t.Errorf("Unexpected result.\nexpected: %s\ngot: %s", expectedErrorString, err.Error())
			return
		}
		checkErrorResult(t, err,
			getMessageCheck("2 errors have occurred"), // Check top level msg
			getErrkitIsCheck(sentinelErr),             // Check that sentinel error is still matchable
			getUnwrapCheck(nil),                       // Check that we unwrap does not work on error list
		)

		errList, ok := err.(errkit.ErrorList)
		if !ok {
			t.Errorf("Unexpected error type.")
			return
		}

		err1 := errList[0]
		err2 := errList[1]

		checkErrorResult(t, err1,
			getMessageCheck("Sentinel error"),
			getStackCheck(fnName+".1", lineNumber+11),
			getDetailsCheck(errkit.ErrorDetails{
				"id": float64(orderedFailures[0]),
			}),
		)

		checkErrorResult(t, err2,
			getMessageCheck("Sentinel error"),
			getStackCheck(fnName+".1", lineNumber+11),
			getDetailsCheck(errkit.ErrorDetails{
				"id": float64(orderedFailures[1]),
			}),
		)
	})
}
