package gorman

import (
	"context"
	"testing"
	"time"
)

func TestManager_Add(t *testing.T) {
	manager := NewManager(Options{})

	g := func(ctx context.Context) error {
		return nil
	}

	manager.Add("test", g, NeverRestart())

	if len(manager.goroutines) != 1 {
		t.Errorf("expected one goroutine to be added, got %d", len(manager.goroutines))
	}

	if manager.goroutines["test"].Func == nil {
		t.Errorf("expected goroutine function to be added")
	}
}

func TestManager_Start(t *testing.T) {
	manager := NewManager(Options{})

	g := func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}

	manager.Add("test", g, NeverRestart())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.Start(ctx, "test")

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	time.Sleep(time.Millisecond * 100)

	if !manager.goroutines["test"].Running() {
		t.Errorf("expected goroutine to be running")
	}

	err = manager.Start(ctx, "invalid")

	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestManager_Stop(t *testing.T) {
	manager := NewManager(Options{})

	g := func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}

	manager.Add("test", g, NeverRestart())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := manager.Start(ctx, "test")

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	time.Sleep(time.Millisecond * 100)

	err = manager.Stop("test")

	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if manager.goroutines["test"].Running() {
		t.Errorf("expected goroutine to be stopped")
	}

	err = manager.Stop("invalid")

	if err == nil {
		t.Errorf("expected an error")
	}
}

func TestManager_Run(t *testing.T) {
	manager := NewManager(Options{})

	g := func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}

	manager.Add("test", g, NeverRestart())

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()
	manager.Run(ctx)
}
