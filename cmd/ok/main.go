package main

import "context"

func main() {
	RunIt(context.Background())
}

func RunIt(ctx context.Context) {
	if ctx.Done() == nil {
		return
	}
	select {
	case <-ctx.Done():
		return
	}
}
