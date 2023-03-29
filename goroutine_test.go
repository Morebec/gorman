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
	time.Sleep(time.Millisecond * 10)
	assert.Len(t, g.executions, 1)
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

	err := g.Stop()
	assert.NoError(t, err)

	assert.False(t, g.Running())
	exec, _ := g.LastExecution()
	assert.NoError(t, exec.Error)
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

	listen := g.Listen()
	cancel()
	<-listen
	<-listen

	assert.False(t, g.Running())
	exec, _ := g.LastExecution()
	assert.NoError(t, exec.Error)
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
	err := g.Stop()
	assert.Error(t, err)
	assert.False(t, g.Running())
}
