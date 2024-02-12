# Background:
Working on Kanister, we faced an [issue #1838](https://github.com/kanisterio/kanister/issues/1838), that currently used `github.com/pkg/errors` package is no longer maintained and the repository has been archived.

That is not a big deal, but it means that we won't get any security updates if any breaches will be discovered in this package. Also, this package lacks some functionality which we would like to have (for instance error details, serialization to JSON).

At the same moment we have internal implementation of errors package in our internal product.

# Proposed requirements
1. Own implementation of an error
2. Based on standard errors package.
3. Should support serialization to JSON
4. Should allow to attach details to an error
5. Should allow to capture current stack location
6. Should provide ability to collect multiple errors at once


# Wished functionality:

## Creation
### errkit.New
It should be possible to create an erorr with message. If needed, additional details could be passed as key-value pairs.
```go
	someError := errkit.New("Sample of error")
	someErrorWithDetails := errkit.New("Sample of error", "DetailName", "String detail value", "NumericDetailName", 123)
    anotherErrorWithDetails := errkit.WithDetails("Sample of error", errkit.ErrorDetails{
        "Some text detail": "String value",
        "Some numeric detail": 123,
    })
```

### errkit.NewSentinelErr
It should be possible to create an error without a stack trace and details, which is intended to be used for predefined errors.
```go
var (
    NotFoundError = errkit.NewSentinelErr("Not found")
)
```
## Wrapping and matching
### errkit.Wrap
It should be possible to encapsulate an error (either a standard error or an errkit error) with a higher-level message.
If needed, additional details could be passed as key-value pairs.

NOTE: `Wrap` is not the same as `WithCause`.
```go
    func foo() error {
		return errors.New("Sample of std error")
    }
	func bar() error {
		return errkit.New("Sample of errkit error")
    }
    
    ...
    
    wrappedStdError := errkit.Wrap(foo(), "Wrapped STD error")
    wrappedErrkitError := errkit.Wrap(bar(), "Wrapped errkit error")
}
```

### errkit.WithStack
It should be possible to add a stack trace to predefined errors (the same should work with regular errors).
If needed, additional details could be passed as key-value pairs.
```go
var (
    NotFoundError := errors.NewSentinelErr("NotFound")
)

err := errkit.WithStack(NotFoundError)
```

### errkit.WithCause
It should be possible to add a cause to predefined errors (the same should work with regular errors).
If needed, additional details could be passed as key-value pairs.
NOTE: `WithCause` is not the same as `Wrap`.
```go
var (
    NotFoundError := errors.NewSentinelErr("NotFound")
)

func FetchSomething(ID string) error {
    err := apiCall() // Here
    if err != nil {
        return errkit.WithCause(NotFoundError, err, "id", ID)
    }
	
    return nil
}

func FooBar() error {
    err := FetchSomething()
    if err == nil {
        return nil
    }
	
    if errkit.Is(err, NotFoundError) {
        return nil // Not found is an expected scenario here, do nothing
    }   
	
    // Errors other than NotFound should be returned as is
    return err
}
```

### errkit.Append
It should be possible to group errors. This is useful when some function executes multiple simultaneous actions and some of them could fail. 
```go
func performOperation(id int) {
    defer wg.Done()
    ...
    // Particular operation ended with an error
    errCh <- errkit.New("Something went wrong", "detailKey", detailValue)
}

// doOperationsConcurrently will return list of errors
func doOperationsConcurrently func() error {
    // Run operations in goroutines
    wg.Add(3)
    go performOperation(1) // This operation will fail
    go performOperation(2) // This operation will succeed
    go performOperation(3) // This operation will fail

    go func() {
        wg.Wait()
        close(errCh)
    }()

    // Collect errors from the channel
    var result error
    for err := range errCh {
        result = errkit.Append(result, err)
    }

    return result
}

func foo() error {
    // Now we can work with list
    err := doOperationsConcurrently()
    if err == nil {
        return nil
    }
    
    if errkit.Is(err, ErrNotFound) {
        // React on not found
        return
    }
    
    // Otherwise we can handle particular errors (very rarely needed case)
    errList, _ := err.(errkit.ErrorList)
    for e := range errList {
        // Analyze particular errors and react on them if needed				
    }  
}
```

### errors.Is and errors.As
It should be possible to use standard errors matching
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
It should be possible to use standard erorrs unwrapping mechanism
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
