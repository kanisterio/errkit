# Background:
Working on Kanister, we faced an [issue #1838](https://github.com/kanisterio/kanister/issues/1838), that currently used `github.com/pkg/errors` package is no longer maintained and the repository has been archived.

That is not a big deal, but it means that we won't get any security updates if any breache will be discovered in this package. Also this package lacks some functionality which we would like to have (for instance error details, serialization to JSON). 

At the same moment we have internal implementation of errors package in our internal product.

# Proposed requirements
1. Own implementation of an error
2. Based on standard errors package.
3. Should support serialization to JSON
4. Should allow to attach details to an error
5. Should allow to capture current stack location
6. Should provide ability to collect multiple errors at once


# Functionality:

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
