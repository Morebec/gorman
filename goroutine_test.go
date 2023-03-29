package gorman

import (
	"context"
	"github.com/morebec/go-errors/errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGoroutine_Start(t *testing.T) {
	g := Goroutine{
		Name: "start_test",
		Func: func(ctx context.Context) error {
			return nil
		},
	}

	g.Start(context.Background())
	assert.True(t, g.Running())
}

func TestGoroutine_Stop(t *testing.T) {
	g := Goroutine{
		Name: "stop_test",
		Func: func(ctx context.Context) error {
			t := time.NewTimer(time.Second * 2)
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				return errors.NewWithMessage("failure", "failed")
			}
		},
	}

	g.Start(context.Background())
	assert.True(t, g.Running())

	g.Stop()

	_, stop := <-g.Listen(), <-g.Listen()
	stopEvent := stop.(GoroutineStoppedEvent)
	assert.False(t, g.Running())
	assert.NoError(t, stopEvent.Error)
}

func TestGoroutine_StopByCancellingContext(t *testing.T) {
	g := Goroutine{
		Name: "stop_test",
		Func: func(ctx context.Context) error {
			t := time.NewTimer(time.Second * 2)
			select {
			case <-ctx.Done():
				return nil
			case <-t.C:
				return errors.NewWithMessage("failure", "failed")
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	g.Start(ctx)
	assert.True(t, g.Running())

	cancel()

	_, stop := <-g.Listen(), <-g.Listen()
	stopEvent := stop.(GoroutineStoppedEvent)
	assert.False(t, g.Running())
	assert.NoError(t, stopEvent.Error)
}

func TestGoroutine_Wait(t *testing.T) {
	g := Goroutine{
		Name: "wait_test",
		Func: func(ctx context.Context) error {
			time.Sleep(time.Second * 1)
			return errors.NewWithMessage("failure", "failed")
		},
	}

	err := g.Wait(context.Background())
	assert.False(t, g.Running())
	assert.Error(t, err)
}

func TestGoroutine_Running(t *testing.T) {
	g := Goroutine{
		Name: "listen_test",
		Func: func(ctx context.Context) error {
			time.Sleep(time.Second * 1)
			return errors.NewWithMessage("failure", "failed")
		},
	}

	g.Start(context.Background())
	assert.True(t, g.Running())
	g.Stop()
	assert.True(t, g.Running())
}
