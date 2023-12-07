package errkit

import (
	"fmt"
)

const (
	noVal  = "NOVAL"
	badKey = "BADKEY"
)

type ErrorDetails map[string]any

// ToErrorDetails accepts either an even size array which contains pais of key/value
// or array of one element of ErrorDetails type.
// Result of function is an ErrorDetails
func ToErrorDetails(details []any) ErrorDetails {
	if len(details) == 0 {
		return nil
	}

	if len(details) == 1 {
		if dp, ok := details[0].(ErrorDetails); ok {
			// Actually we have ErrorDetails on input, so just make a copy
			errorDetails := make(ErrorDetails, len(dp))
			for k, v := range dp {
				errorDetails[k] = v
			}
			return errorDetails
		}
	}

	// It might happen that odd number of elements will be passed, trying our best to handle this case
	if len(details)%2 != 0 {
		details = append(details, noVal)
	}

	errorDetails := make(ErrorDetails, len(details)/2)
	for i := 0; i < len(details); i += 2 {
		name := details[i]
		nameStr, ok := name.(string)
		if !ok {
			nameStr = fmt.Sprintf("%s:(%v)", badKey, name)
		}

		errorDetails[nameStr] = details[i+1]
	}

	return errorDetails
}
