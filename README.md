# Gorman
Gorman is a lightweight Go package for managing and monitoring the execution 
of goroutines. 

- It provides two core concepts:
- `Goroutine`: Which is a struct wrapping a goroutine and allowing to control its execution using Start/Stop semantics.
- `Manager`: Which allows controlling multiple `Goroutines` at once.

The `Manager` can be helpful to create Goroutine supervisors with a HTTP or CLI implementation
to start and stop specific Goroutines.

## Installation
To install Gorman, use go get:

```shell
go get -u github.com/morebec/gorman
```

## Usage
### Manager
To use Gorman, you first need to import it in your Go project:

```go
import "github.com/morebec/gorman"
```

Next, you can create a new instance of a Manager:

```go
man := gorman.NewManager(gorman.Options{
    Logger: slog.New(tint.Options{
        Level:      slog.LevelDebug,
        TimeFormat: time.TimeOnly,
    }.NewHandler(os.Stdout)),
})
```

#### Registering a Goroutine
You can then register a new Goroutine with the manager:

```go
man.Add("MyGoroutine", func(ctx context.Context) error {
    // Goroutine logic goes here
	select {
	case <-ctx.Done():
		return nil
    }
})
```

#### Running the Manager
The `Run` method, allows running the manager and starting all the goroutines that were added to it:
```go
man.Run(context.Background())
```

This method will run until the context is canceled, or the `Shutdown` method is called.

#### Stopping a Goroutine
To stop a running Goroutine, you can call `Stop`:

```go
err := manager.Stop("MyGoroutine")
if err != nil {
    // handle error
}
```

> Note: For goroutines to be stoppable they should correctly listen to the ctx.Done() channel.
> Go routines will also stop whenever they return.


#### Getting the current state of goroutines
The manager exposes the Status method which returns a slice of `GoroutineState` 
which represents the current state of Goroutines.

### Goroutine
Goroutines are the building blocks of Gorman and wrap the execution of a go routine
to allow start/stop semantics. They can be used independently of the manager to control
specific goroutines in isolation.

#### Creating a Goroutine
```go
g := gorman.NewGoroutine("name", func (ctx context.Context) error {
	// Perform work here.
})
```
### Starting a Goroutine
```go
g.Start(context.Background())
```

### Stopping a Goroutine
The `Stop` method will cancel the goroutine's context and will wait for it
to be stopped.
```go
err := g.Stop()
```

Alternatively a goroutine can be stopped through its context.
```go
ctx, cancel := context.WithCancel(context.Background())
g.Start(ctx)
cancel()
```

### LIstening to Goroutines
Goroutines have an internal broadcasting system that allows subscribers to listen to Goroutine events
such as  `GoroutineStartedEvent` and `GoroutineEndedEvent`.

Here's an example:
```go
g := gorman.NewGoroutine("mygoroutine", func(ctx context.Context) error {
	// do something
	return nil
})
eventChan := g.Listen()

g.Start(context.Background())

for event := range eventChan {
	switch event.(type) {
	case gorman.GoroutineStartedEvent:
		fmt.Printf("Goroutine %s started\n", g.State.Name)
	case gorman.GoroutineStoppedEvent:
		e := event.(GoroutineStoppedEvent)
		if e.Error != nil {
			fmt.Printf("Goroutine %s stopped with error: %s\n", event.Name, event.Error.Error())
		} else {
			fmt.Printf("Goroutine %s stopped\n", event.Name)
		}
	}
}

g.Unlisten(eventChan)
```

This mechanism can be useful to react to a goroutine's execution. For instance
the `Manager` uses this monitor the lifecycle of the Goroutines.

> Note: Every call to the Listen() method will return a new channel. When done with using the
> channel, it should be released using the `Unlisten` method.

## Examples
For more usage examples, check the `examples` directory.

## License
Gorman is released under the MIT License.