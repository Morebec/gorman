package gorman

import (
	"context"
	"sync"
	"time"
)

// GoroutineEvent represents an event that happened during the execution of a Goroutine.
type GoroutineEvent interface {
	IsGoroutineEvent()
}

// GoroutineStartedEvent represents the fact that a Goroutine was started.
type GoroutineStartedEvent struct {
	Name      string
	StartedAt time.Time
}

func (g GoroutineStartedEvent) IsGoroutineEvent() {}

// GoroutineStoppedEvent represents the fact that a Goroutine was stopped.
type GoroutineStoppedEvent struct {
	Name      string
	StartedAt time.Time
	EndedAt   time.Time
	Error     error
}

func (g GoroutineStoppedEvent) IsGoroutineEvent() {}

// Goroutine is a wrapper around a goroutine to control its execution using start/stop semantics.
type Goroutine struct {
	Name string
	Func GoroutineFunc

	ctx        context.Context
	cancelFunc context.CancelFunc

	State     GoroutineState
	stateChan <-chan GoroutineEvent

	mu        sync.Mutex
	listeners []chan GoroutineEvent
}

func NewGoroutine(name string, Func GoroutineFunc) *Goroutine {
	return &Goroutine{Name: name, Func: Func}
}

// Start executes a goroutine's function inside a goroutine. It does not wait for it to finish
// before returning.
// It allows passing a context that can be used for cancellation.
// it also allows passing a channel to be notified of events happening within the execution of the goroutine.
func (g *Goroutine) Start(ctx context.Context) {
	if g.Running() {
		return
	}
	g.State.Name = g.Name

	g.ctx, g.cancelFunc = context.WithCancel(ctx)
	if g.stateChan == nil {
		g.stateChan = g.Listen()
	}

	// Listen to parent context for cancellation signals.
	go func() {
		done := ctx.Done()
		if done == nil {
			return
		}
		select {
		case <-done:
			_ = g.Stop()
			return
		}
	}()

	// Update State projection
	go func() {
		for g.Running() {
			select {
			case evt := <-g.stateChan:
				switch evt.(type) {
				case GoroutineStartedEvent:
					e := evt.(GoroutineStartedEvent)
					g.State.StartedAt = &e.StartedAt
					g.State.Running = true
				case GoroutineStoppedEvent:
					e := evt.(GoroutineStoppedEvent)
					g.State.EndedAt = &e.EndedAt
					g.State.Errors = append(g.State.Errors, e.Error)
					g.State.Running = false
				}
			}
		}
	}()

	// Run the function.
	go func() {
		// Send start event
		startedAt := time.Now()
		g.broadcastEvent(GoroutineStartedEvent{
			Name:      g.Name,
			StartedAt: startedAt,
		})

		// Execute function
		err := g.Func(g.ctx)

		// Send stopped event
		g.broadcastEvent(GoroutineStoppedEvent{
			Name:      g.Name,
			StartedAt: startedAt,
			EndedAt:   time.Now(),
			Error:     err,
		})
		// cleanup
		g.cancelFunc = nil
		g.ctx = nil
	}()
}

// Stop requests for the goroutine to stop, by calling its internal cancellation function from its start context.
// waits until the goroutine is stopped.
func (g *Goroutine) Stop() error {
	if g.cancelFunc == nil {
		return nil
	}
	listen := g.Listen()
	defer g.Unlisten(listen)
	g.cancelFunc()

	for evt := range listen {
		switch evt.(type) {
		case GoroutineStoppedEvent:
			e := evt.(GoroutineStoppedEvent)
			return e.Error
		}
	}

	return nil
}

// Wait starts a goroutine and waits for it to be completed.
func (g *Goroutine) Wait(ctx context.Context) error {
	listen := g.Listen()
	defer g.Unlisten(listen)
	g.Start(ctx)

	for evt := range listen {
		switch evt.(type) {
		case GoroutineStoppedEvent:
			e := evt.(GoroutineStoppedEvent)
			return e.Error
		}
	}

	return nil
}

// Running indicates if the goroutine is currently running.
func (g *Goroutine) Running() bool {
	return g.ctx != nil
}

// Listen returns the internal event channel of this Goroutine that can be used to listen to
// its execution events.
func (g *Goroutine) Listen() <-chan GoroutineEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	l := make(chan GoroutineEvent, 8)
	g.listeners = append(g.listeners, l)
	return l
}

func (g *Goroutine) Unlisten(l <-chan GoroutineEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var newListeners []chan GoroutineEvent
	for _, li := range g.listeners {
		if li != l {
			newListeners = append(newListeners, li)
		}
	}
	g.listeners = newListeners
}

func (g *Goroutine) broadcastEvent(event GoroutineEvent) {
	g.mu.Lock()
	listeners := g.listeners
	g.mu.Unlock()

	for _, l := range listeners {
		l := l
		go func() {
			l <- event
		}()

	}
}

// GoroutineFunc represents a function to be executed inside a goroutine.
type GoroutineFunc func(ctx context.Context) error

// GoroutineState represents the current state of a goroutine.
type GoroutineState struct {
	Name      string
	Errors    []error
	StartedAt *time.Time
	EndedAt   *time.Time
	Running   bool
}

// TODO: Allow STOP AND WAIT TO SUPPORT DEADLINES/TIMEOUTS
// TODO: ADD slog to manager usng tint.
// TODO: ADD SERVER AND CLIENT + CLI.
