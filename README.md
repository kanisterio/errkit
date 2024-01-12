# Usage:

## Creation
When creating an error, we can use one of the following ways. All errors created this way will capture the current line and filename.

```go
    // Creation of error with some message.
    someError := errkit.New("Sample of std error")
    
    // Creation of error with some additional details
    anotherError := errkit.New("Some error with details", "TextDetail", "Text value", "NumericDetail", 123)
```

Sometimes it could be useful to create predefined errors. Such errors will not capture the stack.
```go
var (
    NotFoundError := errkit.NewPureError("Not found")
    AlreadyExists := errkit.NewPureError("Already exists")
)

func Foo() error {
    ...
    return NotFoundError
}
```

## Wrapping

### Adding stack trace
If you are interested in adding information about the line and filename where the error happened, you can do the following:
```go
func Foo() error {
    ...
    err := errkit.WithStack(NotFoundError)
    return err
}

func Bar() error {
    err := Foo()
    if err != nil && errkit.Is(err, NotFoundError) {
        fmt.Println("Resource not found, do nothing")
        return nil
    }
    ...
}
```

### Adding error cause information
Sometimes you might be interested in returning a predefined error, but also add some cause error to it, in such cases you can do the following:
```go
func FetchSomething(ID string) error {
    err := doSomething() // Here we have an error 
    if err != nil { // At this step we decide that in such a case we'd like to say that the resource is not found
        return errkit.WithCause(NotFoundError, err)
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

### Wrapping an error with a high-level message
Sometimes you might want to add some high-level information to an error before passing it up to the invoker.
```go
func LoadProfile() error {
    err := makeAnApiCall()
    if err != nil {
        return errkit.Wrap(err, "Unable to load profile")
    }
    return nil
}

```

### Unwrapping errors
If needed, you can always get the wrapped error using the standard errors.Unwrap method, it also has an alias errkit.Unwrap
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

### Matching an errors
You can use standard errors matching methods like errors.Is and errors.As, they also have aliases errkit.Is and errkit.As
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
