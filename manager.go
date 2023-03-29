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

type Options struct {
	Logger *slog.Logger
}

// RestartPolicy is an interface responsible for deciding whether a Goroutine should be restarted when its stops
// from the Manager's point of view.
type RestartPolicy interface {
	// MustRestart indicates if a goroutine should be restarted according to this policy.
	// This method will return true to indicate that the provided Goroutine should be restarted,
	// otherwise it will return false.
	// To apply a delay between restarts, one can simply make this function wait before returning the result.
	MustRestart(g *Goroutine) bool
}

type RestartPolicyFunc func(g *Goroutine) bool

func (r RestartPolicyFunc) MustRestart(g *Goroutine) bool {
	return r(g)
}

// AlwaysRestart this policy indicates that a Goroutine should always be restarted.
func AlwaysRestart() RestartPolicy {
	return RestartPolicyFunc(func(g *Goroutine) bool {
		return true
	})
}

// NeverRestart returns a policy that indicates that a Goroutine should never be restarted.
func NeverRestart() RestartPolicy {
	return RestartPolicyFunc(func(g *Goroutine) bool {
		return false
	})
}

// RestartOnError returns a policy that indicates that a Goroutine should be restarted when an error occurred.
func RestartOnError() RestartPolicy {
	return RestartPolicyFunc(func(g *Goroutine) bool {
		exec, _ := g.LastExecution()
		return exec.Error != nil
	})
}

// managedGoroutine is a decorator for Goroutine that allows specifying restart policies.
type managedGoroutine struct {
	Goroutine
	restartPolicy RestartPolicy
}

func (g *managedGoroutine) mustRestart() bool {
	return g.restartPolicy.MustRestart(&g.Goroutine)
}

type Manager struct {
	goroutines   map[string]*managedGoroutine
	mu           sync.Mutex
	cancelFunc   context.CancelFunc
	logger       *slog.Logger
	shuttingDown bool
}

func NewManager(opts Options) *Manager {

	if opts.Logger == nil {
		void := &bytes.Buffer{}
		opts.Logger = slog.New(slog.NewTextHandler(void))
	}

	return &Manager{
		goroutines: map[string]*managedGoroutine{},
		logger:     opts.Logger,
	}
}

// Add a Goroutine to this manager.
func (m *Manager) Add(name string, f GoroutineFunc, rp RestartPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.goroutines == nil {
		m.goroutines = map[string]*managedGoroutine{}
	}

	if rp == nil {
		rp = NeverRestart()
	}

	m.goroutines[name] = &managedGoroutine{
		Goroutine: Goroutine{
			Name: name,
			Func: f,
		},
		restartPolicy: rp,
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
						m.logger.Error(fmt.Sprintf("%s encontered error %s", e.Name, e.Error.Error()))
					}
					m.logger.Info(fmt.Sprintf("%s stopped", e.Name))
					gr.Unlisten(listener)
					if !m.shuttingDown && gr.mustRestart() {
						m.logger.Info(fmt.Sprintf("will restart %s ...", e.Name))
						if err := m.Start(ctx, gr.Name); err != nil {
							m.logger.Error(fmt.Sprintf("failed restarting %s, encontered error %s", e.Name, e.Error.Error()))
						}
					}
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

// Status returns the current executions of all Goroutine managed by this manager.
func (m *Manager) Status() map[string]GoroutineExecution {
	m.mu.Lock()
	defer m.mu.Unlock()

	states := map[string]GoroutineExecution{}
	for _, g := range m.goroutines {
		exec, ok := g.LastExecution()
		if !ok {
			exec = GoroutineExecution{}
		}
		states[g.Name] = exec
	}

	return states
}

// StatusByName returns the status of a Goroutine by its name, or an error with code GoroutineNotFoundErrorCode.
func (m *Manager) StatusByName(name string) (GoroutineExecution, error) {
	_, err := m.findGoroutineByName(name)
	if err != nil {
		return GoroutineExecution{}, err
	}

	return m.Status()[name], nil
}

// Shutdown requests shutting down the manager and all its Goroutine.
func (m *Manager) Shutdown() {
	if m.cancelFunc == nil {
		return
	}
	m.cancelFunc()
	m.shuttingDown = true
}

// returns a Goroutine by its name, or returns an error with code GoroutineNotFoundErrorCode.
func (m *Manager) findGoroutineByName(name string) (*managedGoroutine, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	gr, ok := m.goroutines[name]
	if !ok {
		return nil, errors.NewWithMessage(GoroutineNotFoundErrorCode, fmt.Sprintf("no goroutine named %s", name))
	}
	return gr, nil
}
