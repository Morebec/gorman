package gorman

import (
	"context"
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
	stateChan chan GoroutineEvent

	eventChan chan GoroutineEvent
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
	g.eventChan = make(chan GoroutineEvent, 2)
	g.stateChan = make(chan GoroutineEvent, 2)

	// Listen to parent context for cancellation signals.
	go func() {
		done := ctx.Done()
		if done == nil {
			return
		}
		select {
		case <-done:
			g.Stop()
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
		startedEvent := GoroutineStartedEvent{
			Name:      g.Name,
			StartedAt: startedAt,
		}

		g.stateChan <- startedEvent
		g.eventChan <- startedEvent

		// Execute function
		err := g.Func(g.ctx)

		// Send stopped event
		stoppedEvent := GoroutineStoppedEvent{
			Name:      g.Name,
			StartedAt: startedAt,
			EndedAt:   time.Now(),
			Error:     err,
		}
		g.stateChan <- stoppedEvent
		g.eventChan <- stoppedEvent

		/// cleanup
		close(g.eventChan)
		close(g.stateChan)
		g.cancelFunc = nil
		g.ctx = nil
	}()
}

// Stop requests for the goroutine to stop, by calling its internal cancellation function from its start context.
func (g *Goroutine) Stop() {
	if g.cancelFunc == nil {
		return
	}
	g.cancelFunc()
}

// Wait starts a goroutine and waits for it to be completed.
func (g *Goroutine) Wait(ctx context.Context) error {
	g.Start(ctx)
	_, stopEvent := <-g.eventChan, <-g.eventChan

	stop := stopEvent.(GoroutineStoppedEvent)
	if stop.Error != nil {
		return stop.Error
	}

	return nil
}

// Running indicates if the goroutine is currently running.
func (g *Goroutine) Running() bool {
	return g.ctx != nil
}

// Listen returns the internal event channel of this Goroutine that can be used to listen to
// its execution events.
func (g *Goroutine) Listen() chan GoroutineEvent {
	return g.eventChan
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
