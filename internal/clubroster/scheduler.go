package clubroster

import (
	"context"
	"log"
	"time"
)

// RunDaily runs fn immediately, then every interval, until ctx is
// cancelled. It's the mechanism that makes the club roster sync run on a
// schedule without manual triggering. RunDaily is meant to run in its own
// goroutine for the life of the process, so a panic in one run is recovered
// rather than taking the whole server down with it.
func RunDaily(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	runRecovered(ctx, fn)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runRecovered(ctx, fn)
		}
	}
}

func runRecovered(ctx context.Context, fn func(context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("clubroster: recovered from panic: %v", r)
		}
	}()
	fn(ctx)
}
