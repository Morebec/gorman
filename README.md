# Gorman
Gorman is a lightweight Go package for managing goroutines. It provides a simple API to start and stop goroutines, and also allows for managing goroutine state.

## Usage
To use Gorman, you first need to import it in your Go project:

```go
import "github.com/morebec/gorman"
```

Next, you can create a new instance of a Manager:

```go
manager := gorman.NewManager()
```

### Registering a Goroutine
You can then register a new Goroutine with the manager:

```go
goroutine := gorman.NewGoroutine("MyGoroutine", func(ctx context.Context) error {
    // Goroutine logic goes here
})
manager.Register(goroutine)
```

### Starting a Goroutine
To start a registered Goroutine, you can call Start:

```go
err := manager.Start(context.Background(), "MyGoroutine")
if err != nil {
    // handle error
}
```

### Stopping a Goroutine
To stop a running Goroutine, you can call Stop:

```go
err := manager.Stop("MyGoroutine")
if err != nil {
    // handle error
}
```

> Note: For goroutines to be stoppable they should correctly listen to the ctx.Done() channel.

### Starting all Goroutines
To start all the goroutine, simply call the `StartAll` method:
```go
err := manager.StartAll()
if err != nil {
	panic(err)
}
```
### Stopping all Goroutines
To stop all the goroutine, simply call the `StopAll` method:
```go
err := manager.StopAll()
if err != nil {
	panic(err)
}
```

### Getting the current state of a goroutine
The manager exposes a chennel that allows to listen to Goroutine State changes:
```go
go func() {
    for {
        state := <-manager.StateChannel()
        fmt.Println("Here is the current state of %s running: %s", state.Name, state.Running)
    }
}()

```

Examples
For more usage examples, check the examples directory.

License
Gorman is released under the MIT License.