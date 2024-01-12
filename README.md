# Usage:

## Creation
### errkit.New
```go
	someError := errkit.New("Sample of std error")
```

### errkit.WithStack
```go
var (
    somePredefinedError := errors.New("Sample of errkit error")
)

err := errkit.WithStack(predefinedTestError)
```

### errkit.WithDetail
```go
    err := errkit.New("Some error with details")
        .WithDetail("Some text detail", "String value")
        .WithDetail("Some numeric detail", 123)
```
### errkit.WithDetails
```go
    err := errkit.New("Some error with details").WithDetails(
		errkit.ErrorDetails{
			"Some text detail":    "String value",
			"Some numeric detail": 123,
		})

```

## Wrapping and matching
### errkit.Wrap
Allows to wrap an error (either std error or errkit error) with higher level message.
```go
    var (
        predefinedStdError := errors.New("Sample of std error")
    )

    ...

    wrappedStdError := errkit.Wrap(predefinedStdError, "Wrapped STD error")
```

### errors.Is and errors.As
Allows to use standard errors matching

`Errkit` also has an aliases `errkit.Is` and `errkit.As`
```go
    // testErrorType which implements std error interface
    type testErrorType struct {
        message string
    }
	
    var (
        predefinedTestError = newTestError("Sample error of custom type")
    )

    wrappedTestError := errkit.Wrap(predefinedTestError, "Wrapped TEST error")
    if !errors.Is(wrappedTestError, predefinedTestError) {
        return errors.New("error is not implementing requested type")
    }

    var asErr *testErrorType
    if !errors.As(origErr, &asErr) {
        return errors.New("unable to cast error to its cause")
    }
```

### errors.Unwrap
Allows to use standard erorrs unwrapping

`Errkit` also has an alias `errkit.Unwrap`
```go
    var (
		predefinedError = errors.New("Some predefined error")
    )   

    wrappedError := errkit.Wrap(predefinedError, "Wrapped error")

    err := errors.Unwrap(wrappedError)
    if err != predefinedStdError {
        return errors.New("Unable to wrap error cause")
    }
```
