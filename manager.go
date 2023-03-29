package gorman

import (
	"bytes"
	"context"
	"fmt"
	"github.com/morebec/go-errors/errors"
	"golang.org/x/exp/slog"
	"sync"
	"time"
)

const GoroutineNotFoundErrorCode = "goroutine_not_found"

type Manager struct {
	goroutines map[string]*Goroutine
	mu         sync.Mutex
	cancelFunc context.CancelFunc
	logger     *slog.Logger
}

type Options struct {
	Logger *slog.Logger
}

func NewManager(opts Options) *Manager {

	if opts.Logger == nil {
		void := &bytes.Buffer{}
		opts.Logger = slog.New(slog.NewTextHandler(void))
	}

	return &Manager{
		goroutines: map[string]*Goroutine{},
		logger:     opts.Logger,
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

	if gr.Running() {
		return nil
	}

	m.logger.Info(fmt.Sprintf("starting %s...", name))

	listener := gr.Listen()
	gr.Start(ctx)
	go func() {
		for {
			select {
			case evt := <-listener:
				switch evt.(type) {
				case GoroutineStartedEvent:
					e := evt.(GoroutineStartedEvent)
					m.logger.Info(fmt.Sprintf("%s started", e.Name))
				case GoroutineStoppedEvent:
					e := evt.(GoroutineStoppedEvent)
					if e.Error != nil {
						m.logger.Info(fmt.Sprintf("%s encontered error %s", e.Name, e.Error.Error()))
					}
					m.logger.Info(fmt.Sprintf("%s stopped", e.Name))
					gr.Unlisten(listener)
				}

			}
		}
		//eventChan := gr.Listen()
		//
		//// started
		//started := <-eventChan
		//startedEvt := started.(GoroutineStartedEvent)
		//log.Printf("%s started \n", startedEvt.Name)
		//
		//// stopped
		//stopped := <-eventChan
		//stoppedEvt := stopped.(GoroutineStoppedEvent)
		//if stoppedEvt.Error != nil {
		//	log.Printf("%s encountered error %s \n", stoppedEvt.Name, stoppedEvt.Error)
		//}
		//log.Printf("%s stopped \n", stoppedEvt.Name)
	}()

	return nil
}

// Stop a Goroutine by its name.
func (m *Manager) Stop(name string) error {
	gr, err := m.findGoroutineByName(name)
	if err != nil {
		return err
	}

	if !gr.Running() {
		return nil
	}

	m.logger.Info(fmt.Sprintf("stopping %s...", name))
	return gr.Stop()
}

// Run the manager and all the registered Goroutine, until the manager is shutdown.
func (m *Manager) Run(ctx context.Context) {
	m.logger.Info("gorman started")

	// Start Goroutines

	ctx, cancel := context.WithCancel(ctx)
	m.cancelFunc = cancel

	m.StartAll(ctx)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			m.logger.Info("shutting down gorman...")
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
