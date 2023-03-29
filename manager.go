package gorman

import (
	"context"
	"fmt"
	"github.com/morebec/go-errors/errors"
	"log"
	"sync"
	"time"
)

const GoroutineNotFoundErrorCode = "goroutine_not_found"

type Manager struct {
	goroutines map[string]*Goroutine

	eventChan chan GoroutineEvent

	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

func NewManager() *Manager {
	return &Manager{
		goroutines: map[string]*Goroutine{},
		eventChan:  make(chan GoroutineEvent, 20),
	}
}

// Add a Goroutine to this manager.
func (m *Manager) Add(name string, f GoroutineFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.goroutines == nil {
		m.goroutines = map[string]*Goroutine{}
	}
	m.goroutines[name] = &Goroutine{
		Name: name,
		Func: f,
	}
}

// Start a Goroutine or returns an error if there was a problem starting the goroutine.
// The goroutine must have been added to the manager.
func (m *Manager) Start(ctx context.Context, name string) error {
	gr, err := m.findGoroutineByName(name)
	if err != nil {
		return err
	}

	gr.Start(ctx)
	go func() {
		eventChan := gr.Listen()
		// started
		started := <-eventChan
		startedEvt := started.(GoroutineStartedEvent)
		log.Printf("%s started \n", startedEvt.Name)

		// stopped
		stopped := <-eventChan
		stoppedEvt := stopped.(GoroutineStoppedEvent)
		if stoppedEvt.Error != nil {
			log.Printf("%s encountered error %s \n", stoppedEvt.Name, stoppedEvt.Error)
		}
		log.Printf("%s stopped \n", stoppedEvt.Name)
	}()

	return nil
}

// Stop a Goroutine by its name.
func (m *Manager) Stop(name string) error {
	gr, err := m.findGoroutineByName(name)
	if err != nil {
		return err
	}
	gr.Stop()
	return nil
}

// Run the manager and all the registered Goroutine, until the manager is shutdown.
func (m *Manager) Run(ctx context.Context) {
	log.Printf("gorman started")

	// Start Goroutines

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	m.StartAll(ctx)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			// Perform shutdown
			m.StopAll(true)
			wg.Done()
		}
		return
	}()

	wg.Wait()
}

// StartAll the Goroutine managed by this manager.
func (m *Manager) StartAll(ctx context.Context) {
	m.mu.Lock()
	goroutines := m.goroutines
	m.mu.Unlock()
	for name := range goroutines {
		if err := m.Start(ctx, name); err != nil {
			continue
		}
	}
}

// StopAll the Goroutine managed by this manager.
func (m *Manager) StopAll(wait bool) {
	m.mu.Lock()
	goroutines := m.goroutines
	m.mu.Unlock()

	for name := range goroutines {
		if err := m.Stop(name); err != nil {
			continue
		}
	}

	for wait {
		wait = false
		m.mu.Lock()
		goroutines = m.goroutines
		m.mu.Unlock()
		for _, gr := range goroutines {
			if gr.Running() {
				wait = true
				break
			}
		}
		if wait {
			time.Sleep(time.Second * 2)
		}
	}
}

// Status returns the current state of all Goroutine managed by this manager.
func (m *Manager) Status() map[string]GoroutineState {
	m.mu.Lock()
	defer m.mu.Unlock()

	states := map[string]GoroutineState{}
	for _, g := range m.goroutines {
		states[g.Name] = g.State
	}

	return states
}

// StatusByName returns the status of a Goroutine by its name, or an error with code GoroutineNotFoundErrorCode.
func (m *Manager) StatusByName(name string) (GoroutineState, error) {
	gr, err := m.findGoroutineByName(name)
	if err != nil {
		return GoroutineState{}, err
	}

	return gr.State, nil
}

// Shutdown requests shutting down the manager and all its Goroutine.
func (m *Manager) Shutdown() {
	if m.cancelFunc == nil {
		return
	}
	m.cancelFunc()
}

// returns a Goroutine by its name, or returns an error with code GoroutineNotFoundErrorCode.
func (m *Manager) findGoroutineByName(name string) (*Goroutine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	gr, ok := m.goroutines[name]
	if !ok {
		return nil, errors.NewWithMessage(GoroutineNotFoundErrorCode, fmt.Sprintf("no goroutine named %s", name))
	}
	return gr, nil
}
