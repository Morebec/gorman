package main

import (
	"context"
	"github.com/lmittmann/tint"
	"github.com/morebec/gorman"
	"golang.org/x/exp/slog"
	_ "net/http/pprof"
	"os"
	"time"
)

// This example starts three goroutines, where go1 after two seconds stops go2, which in turn after two seconds,
// will stop go3, which will finally shut down the manager and exist the program.
func main() {
	man := gorman.NewManager(slog.New(tint.NewHandler(os.Stderr)))
	man.Add("go1", func(ctx context.Context) error {
		time.Sleep(time.Second * 2)
		return man.Stop("go2")
	})

	man.Add("go2", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			time.Sleep(time.Second * 2)
			return man.Stop("go3")
		}
	})

	man.Add("go3", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			time.Sleep(time.Second * 8)
			man.Shutdown()
			return nil
		}
	})

	man.Run(context.Background())
}

//func main() {
//
//
//
//	// Create a new Manager instance
//	g := gorman.NewManager()
//
//	go func() {
//		for {
//			select {
//			case s := <-g.StateChannel:
//
//				fmt.Printf("State: %#v\n", s)
//			}
//		}
//	}()
//
//	// Register some goroutines
//	g.Register(gorman.NewGoroutine("routine1", func(ctx context.Context) error {
//		for {
//			select {
//			case <-ctx.Done():
//				fmt.Println("routine1: stopped")
//				return nil
//			default:
//				fmt.Println("routine1: running")
//				time.Sleep(1 * time.Second)
//			}
//		}
//	}))
//	g.Register(gorman.NewGoroutine("routine2", func(ctx context.Context) error {
//		for {
//			select {
//			case <-ctx.Done():
//				fmt.Println("routine2: stopped")
//				return nil
//			default:
//				fmt.Println("routine2: running")
//				time.Sleep(1 * time.Second)
//			}
//		}
//	}))
//
//	ctx := gorman.NewCtx()
//
//	// StartAsync the goroutines
//	err := g.StartAsync(ctx, "routine1")
//	if err != nil {
//		fmt.Println("error starting routine1:", err)
//	}
//	err = g.StartAsync(ctx, "routine2")
//	if err != nil {
//		fmt.Println("error starting routine2:", err)
//	}
//
//	// Run for some time
//	time.Sleep(5 * time.Second)
//
//	// Show State of Goroutines
//	state, err := g.State("routine1")
//	if err != nil {
//		fmt.Println("failed getting state of routine1", state)
//	}
//	fmt.Printf("%#v\n", state)
//
//	time.Sleep(5 * time.Second)
//
//	// StopAsync the goroutines
//	fmt.Println("stopping routine1", err)
//	err = g.StopAsync("routine1")
//	if err != nil {
//		fmt.Println("error stopping routine1:", err)
//	}
//
//	fmt.Println("stopping routine2", err)
//	err = g.StopAsync("routine2")
//	if err != nil {
//		fmt.Println("error stopping routine2:", err)
//	}
//
//	fmt.Println("All finished", err)
//
//	// Run for some time
//	time.Sleep(2 * time.Second)
//
//	fmt.Println("Ciao!", err)
//	g.Shutdown()
//}
